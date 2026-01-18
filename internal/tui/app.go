package tui

import (
	"context"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/diiviikk5/dvkcli/internal/config"
	"github.com/diiviikk5/dvkcli/internal/memory"
	"github.com/diiviikk5/dvkcli/internal/ollama"
	"github.com/google/uuid"
	ollamaapi "github.com/ollama/ollama/api"
)

// Message roles
const (
	RoleUser      = "user"
	RoleAssistant = "assistant"
	RoleSystem    = "system"
)

// ChatMessage represents a message in the chat
type ChatMessage struct {
	Role    string
	Content string
	Time    time.Time
}

// Model is the main Bubbletea model
type Model struct {
	// Core components
	client *ollama.Client
	store  *memory.Store
	cfg    *config.Config

	// UI components
	textarea textarea.Model
	viewport viewport.Model
	spinner  spinner.Model

	// State
	messages       []ChatMessage
	conversationID string
	streaming      bool
	streamContent  string
	connected      bool
	memoryCount    int

	// Layout
	width  int
	height int
	ready  bool

	// Animation
	spinnerFrame int
	typingFrame  int

	// Error handling
	err error
}

// Messages for Bubbletea
type (
	streamChunkMsg  string
	streamDoneMsg   struct{}
	streamErrorMsg  error
	streamResultMsg struct{ content string }
	connectionMsg   bool
	memoryCountMsg  int
	tickMsg         time.Time
)

// New creates a new TUI model
func New(client *ollama.Client, store *memory.Store, cfg *config.Config) *Model {
	ta := textarea.New()
	ta.Placeholder = "Type your message..."
	ta.Focus()
	ta.CharLimit = 4096
	ta.SetWidth(80)
	ta.SetHeight(3)
	ta.FocusedStyle.CursorLine = lipgloss.NewStyle()
	ta.ShowLineNumbers = false
	ta.KeyMap.InsertNewline.SetEnabled(false)

	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = SpinnerStyle

	return &Model{
		client:         client,
		store:          store,
		cfg:            cfg,
		textarea:       ta,
		spinner:        s,
		messages:       []ChatMessage{},
		conversationID: uuid.New().String(),
	}
}

// Init initializes the model
func (m *Model) Init() tea.Cmd {
	return tea.Batch(
		textarea.Blink,
		m.checkConnection(),
		m.loadMemoryCount(),
		m.tickCmd(),
	)
}

// Update handles messages
func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c":
			return m, tea.Quit
		case "enter":
			if !m.streaming && strings.TrimSpace(m.textarea.Value()) != "" {
				input := strings.TrimSpace(m.textarea.Value())
				// Handle special commands
				if strings.HasPrefix(input, "/") {
					return m, m.handleCommand(input)
				}
				return m, m.sendMessage()
			}
		case "ctrl+n":
			// New conversation
			m.messages = []ChatMessage{}
			m.conversationID = uuid.New().String()
			m.textarea.Reset()
			m.viewport.SetContent(m.renderMessages())
			return m, nil
		case "ctrl+l":
			// Load last conversation
			return m, m.loadLastConversation()
		case "ctrl+e":
			// Export conversation
			return m, m.exportConversation()
		case "up", "k":
			// Scroll up
			m.viewport.LineUp(3)
			return m, nil
		case "down", "j":
			// Scroll down
			m.viewport.LineDown(3)
			return m, nil
		case "pgup":
			m.viewport.HalfViewUp()
			return m, nil
		case "pgdown":
			m.viewport.HalfViewDown()
			return m, nil
		}

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

		headerHeight := 4
		inputHeight := 5
		statusHeight := 1
		viewportHeight := m.height - headerHeight - inputHeight - statusHeight - 2

		if !m.ready {
			m.viewport = viewport.New(m.width-4, viewportHeight)
			m.viewport.HighPerformanceRendering = false
			m.ready = true
		} else {
			m.viewport.Width = m.width - 4
			m.viewport.Height = viewportHeight
		}

		m.textarea.SetWidth(m.width - 6)
		m.viewport.SetContent(m.renderMessages())

	case streamChunkMsg:
		m.streamContent += string(msg)
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case streamDoneMsg:
		m.streaming = false
		// Save the complete assistant message
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleAssistant,
			Content: m.streamContent,
			Time:    time.Now(),
		})
		m.streamContent = ""
		m.viewport.SetContent(m.renderMessages())
		return m, m.saveToMemory()

	case streamErrorMsg:
		m.streaming = false
		m.err = msg
		m.streamContent = ""
		return m, nil

	case streamResultMsg:
		m.streaming = false
		// Save the complete assistant message
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleAssistant,
			Content: msg.content,
			Time:    time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, m.saveToMemory()

	case commandResultMsg:
		// Show command result as assistant message
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleAssistant,
			Content: msg.content,
			Time:    time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()
		return m, nil

	case loadConversationMsg:
		// Load messages from conversation
		if msg.conversation != nil && len(msg.conversation.Messages) > 0 {
			m.conversationID = msg.conversation.ID
			m.messages = []ChatMessage{}
			for _, memMsg := range msg.conversation.Messages {
				m.messages = append(m.messages, ChatMessage{
					Role:    memMsg.Role,
					Content: memMsg.Content,
					Time:    memMsg.CreatedAt,
				})
			}
			m.viewport.SetContent(m.renderMessages())
			m.viewport.GotoBottom()
		}
		return m, nil

	case connectionMsg:
		m.connected = bool(msg)
		return m, nil

	case memoryCountMsg:
		m.memoryCount = int(msg)
		return m, nil

	case tickMsg:
		if m.streaming {
			m.typingFrame = (m.typingFrame + 1) % len(TypingFrames)
			m.viewport.SetContent(m.renderMessages())
		}
		return m, m.tickCmd()

	case spinner.TickMsg:
		var cmd tea.Cmd
		m.spinner, cmd = m.spinner.Update(msg)
		return m, cmd
	}

	// Update textarea
	if !m.streaming {
		var cmd tea.Cmd
		m.textarea, cmd = m.textarea.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	m.viewport, cmd = m.viewport.Update(msg)
	cmds = append(cmds, cmd)

	return m, tea.Batch(cmds...)
}

// View renders the TUI
func (m *Model) View() string {
	if !m.ready {
		return "\n  Initializing..."
	}

	var b strings.Builder

	// Header
	header := m.renderHeader()
	b.WriteString(header)
	b.WriteString("\n")

	// Chat viewport
	chatBox := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(Subtle).
		Width(m.width - 2).
		Render(m.viewport.View())
	b.WriteString(chatBox)
	b.WriteString("\n")

	// Input area
	inputBox := InputStyle.
		Width(m.width - 4).
		Render(m.textarea.View())
	b.WriteString(inputBox)
	b.WriteString("\n")

	// Status bar
	b.WriteString(m.renderStatusBar())

	return b.String()
}

// renderHeader renders the header with logo and status
func (m *Model) renderHeader() string {
	logo := RenderCompactLogo()

	modelStatus := fmt.Sprintf("%s %s", ModelIcon(m.connected), m.client.Model)
	modelStyled := lipgloss.NewStyle().Foreground(Muted).Render(modelStatus)

	memoryStatus := ""
	if m.cfg.MemoryEnabled {
		memoryStatus = fmt.Sprintf(" │ 🧠 %d memories", m.memoryCount)
		memoryStatus = lipgloss.NewStyle().Foreground(Info).Render(memoryStatus)
	}

	// Calculate spacing
	leftContent := logo
	rightContent := modelStyled + memoryStatus
	spaces := m.width - lipgloss.Width(leftContent) - lipgloss.Width(rightContent) - 4
	if spaces < 1 {
		spaces = 1
	}

	return lipgloss.NewStyle().
		Padding(0, 2).
		Render(leftContent + strings.Repeat(" ", spaces) + rightContent)
}

// renderMessages renders all chat messages
func (m *Model) renderMessages() string {
	var b strings.Builder

	// Always show logo at top
	logo := RenderLogo()
	tagline := lipgloss.NewStyle().
		Foreground(Secondary).
		Italic(true).
		Render("Master Divik, your slave is here. What do you want me to do for you?")
	b.WriteString(logo + "\n" + tagline + "\n\n")

	// Show messages if any
	for _, msg := range m.messages {
		b.WriteString(m.renderMessage(msg))
		b.WriteString("\n\n")
	}

	// Render streaming content
	if m.streaming && m.streamContent != "" {
		b.WriteString(m.renderStreamingMessage())
	} else if m.streaming {
		// Show typing indicator
		indicator := TypingFrames[m.typingFrame]
		b.WriteString(lipgloss.NewStyle().
			Foreground(Primary).
			Render(fmt.Sprintf("  %s your slave is thinking...", indicator)))
	}

	return b.String()
}

// renderMessage renders a single message
func (m *Model) renderMessage(msg ChatMessage) string {
	var prefix string
	var style lipgloss.Style

	switch msg.Role {
	case RoleUser:
		prefix = "▸ Master"
		style = UserMessageStyle
	case RoleAssistant:
		prefix = "◆ Slave"
		style = AssistantMessageStyle
	default:
		prefix = "◇ System"
		style = SystemMessageStyle
	}

	header := lipgloss.NewStyle().Bold(true).Foreground(style.GetForeground()).Render(prefix)
	timestamp := lipgloss.NewStyle().Foreground(Subtle).Render(msg.Time.Format("15:04"))

	content := style.Render(msg.Content)

	return fmt.Sprintf("%s %s\n%s", header, timestamp, content)
}

// renderStreamingMessage renders the current streaming message
func (m *Model) renderStreamingMessage() string {
	header := lipgloss.NewStyle().Bold(true).Foreground(Primary).Render("◆ Slave")
	cursor := lipgloss.NewStyle().Foreground(Secondary).Render("▌")
	content := AssistantMessageStyle.Render(m.streamContent) + cursor

	return fmt.Sprintf("%s\n%s", header, content)
}

// renderStatusBar renders the bottom status bar
func (m *Model) renderStatusBar() string {
	// Left side: help
	help := HelpStyle.Render("Enter ") + HelpKeyStyle.Render("send") +
		HelpStyle.Render(" • /help ") + HelpKeyStyle.Render("cmds") +
		HelpStyle.Render(" • Ctrl+C ") + HelpKeyStyle.Render("quit")

	// Right side: connection status
	var status string
	if m.connected {
		status = StatusActiveStyle.Render("● connected")
	} else {
		status = StatusErrorStyle.Render("○ disconnected")
	}

	// Calculate spacing
	spaces := m.width - lipgloss.Width(help) - lipgloss.Width(status) - 4
	if spaces < 1 {
		spaces = 1
	}

	return StatusBarStyle.Width(m.width).Render(help + strings.Repeat(" ", spaces) + status)
}

// sendMessage sends the current input to Ollama
func (m *Model) sendMessage() tea.Cmd {
	content := strings.TrimSpace(m.textarea.Value())
	if content == "" {
		return nil
	}

	// Add user message
	m.messages = append(m.messages, ChatMessage{
		Role:    RoleUser,
		Content: content,
		Time:    time.Now(),
	})

	m.textarea.Reset()
	m.streaming = true
	m.streamContent = ""
	m.viewport.SetContent(m.renderMessages())
	m.viewport.GotoBottom()

	return m.streamResponse(content)
}

// streamResponse gets the response from Ollama (non-streaming for reliability)
func (m *Model) streamResponse(prompt string) tea.Cmd {
	return func() tea.Msg {
		// Use a timeout context
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		// Build messages for context
		messages := []ollamaapi.Message{}

		// Add system prompt
		if m.cfg.SystemPrompt != "" {
			messages = append(messages, ollamaapi.Message{
				Role:    "system",
				Content: m.cfg.SystemPrompt,
			})
		}

		// Add conversation history
		for _, msg := range m.messages {
			messages = append(messages, ollamaapi.Message{
				Role:    msg.Role,
				Content: msg.Content,
			})
		}

		// Use non-streaming Chat for reliability
		response, err := m.client.Chat(ctx, messages)
		if err != nil {
			return streamResultMsg{content: fmt.Sprintf("Error: %v", err)}
		}

		if response == "" {
			return streamResultMsg{content: "No response from AI. The model might be loading..."}
		}

		return streamResultMsg{content: response}
	}
}

// checkConnection checks if Ollama is connected
func (m *Model) checkConnection() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		return connectionMsg(m.client.IsConnected(ctx))
	}
}

// loadMemoryCount loads the memory count from store
func (m *Model) loadMemoryCount() tea.Cmd {
	return func() tea.Msg {
		if m.store == nil {
			return memoryCountMsg(0)
		}
		count, _ := m.store.GetMessageCount(context.Background())
		return memoryCountMsg(count)
	}
}

// saveToMemory saves the last exchange to memory
func (m *Model) saveToMemory() tea.Cmd {
	return func() tea.Msg {
		if m.store == nil || !m.cfg.MemoryEnabled {
			return nil
		}

		ctx := context.Background()

		// Ensure conversation exists
		conv, _ := m.store.GetConversation(ctx, m.conversationID)
		if conv == nil {
			title := "New Chat"
			if len(m.messages) > 0 {
				title = truncate(m.messages[0].Content, 50)
			}
			m.store.CreateConversation(ctx, m.conversationID, title)
		}

		// Save last two messages (user + assistant)
		if len(m.messages) >= 2 {
			for i := len(m.messages) - 2; i < len(m.messages); i++ {
				msg := m.messages[i]

				// Generate embedding for user messages
				var embedding []float32
				if msg.Role == RoleUser {
					embedding, _ = m.client.Embed(ctx, msg.Content)
				}

				err := m.store.SaveMessage(ctx, &memory.Message{
					ID:             uuid.New().String(),
					ConversationID: m.conversationID,
					Role:           msg.Role,
					Content:        msg.Content,
					Embedding:      embedding,
					CreatedAt:      msg.Time,
				})
				if err != nil {
					continue
				}
			}
		}

		count, _ := m.store.GetMessageCount(ctx)
		return memoryCountMsg(count)
	}
}

// tickCmd returns a tick command for animations
func (m *Model) tickCmd() tea.Cmd {
	return tea.Tick(100*time.Millisecond, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// truncate truncates a string to a maximum length
func truncate(s string, max int) string {
	if len(s) <= max {
		return s
	}
	return s[:max-3] + "..."
}

// handleCommand handles slash commands
func (m *Model) handleCommand(input string) tea.Cmd {
	m.textarea.Reset()

	parts := strings.Fields(input)
	if len(parts) == 0 {
		return nil
	}

	cmd := strings.ToLower(parts[0])

	switch cmd {
	case "/help":
		helpText := `Available commands:
  /help     - Show this help
  /models   - List available Ollama models
  /search   - Search past conversations
  /clear    - Clear current conversation
  /export   - Export conversation to markdown
  
Shortcuts:
  Enter     - Send message
  Ctrl+N    - New conversation
  Ctrl+L    - Load last conversation
  Ctrl+E    - Export conversation
  Ctrl+C    - Quit
  ↑/↓       - Scroll`
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleAssistant,
			Content: helpText,
			Time:    time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
		m.viewport.GotoBottom()

	case "/models":
		return m.listModels()

	case "/search":
		if len(parts) < 2 {
			m.messages = append(m.messages, ChatMessage{
				Role:    RoleAssistant,
				Content: "Usage: /search <query>",
				Time:    time.Now(),
			})
		} else {
			query := strings.Join(parts[1:], " ")
			return m.searchMemory(query)
		}
		m.viewport.SetContent(m.renderMessages())

	case "/clear":
		m.messages = []ChatMessage{}
		m.conversationID = uuid.New().String()
		m.viewport.SetContent(m.renderMessages())

	case "/export":
		return m.exportConversation()

	default:
		m.messages = append(m.messages, ChatMessage{
			Role:    RoleAssistant,
			Content: fmt.Sprintf("Unknown command: %s. Type /help for available commands.", cmd),
			Time:    time.Now(),
		})
		m.viewport.SetContent(m.renderMessages())
	}

	return nil
}

// listModels lists available Ollama models
func (m *Model) listModels() tea.Cmd {
	return func() tea.Msg {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		models, err := m.client.ListModels(ctx)
		if err != nil {
			return streamResultMsg{content: fmt.Sprintf("Error listing models: %v", err)}
		}

		var sb strings.Builder
		sb.WriteString("Available models:\n\n")
		for i, model := range models {
			marker := "  "
			if model.Name == m.client.Model {
				marker = "► "
			}
			sb.WriteString(fmt.Sprintf("%s%d. %s\n", marker, i+1, model.Name))
		}
		sb.WriteString("\nCurrent model: " + m.client.Model)

		return commandResultMsg{content: sb.String()}
	}
}

// searchMemory searches past conversations
func (m *Model) searchMemory(query string) tea.Cmd {
	return func() tea.Msg {
		if m.store == nil {
			return commandResultMsg{content: "Memory is not enabled."}
		}

		ctx := context.Background()

		// Generate embedding for query
		embedding, err := m.client.Embed(ctx, query)
		if err != nil {
			return commandResultMsg{content: fmt.Sprintf("Error generating embedding: %v", err)}
		}

		// Search
		results, err := m.store.Search(ctx, embedding, 5)
		if err != nil {
			return commandResultMsg{content: fmt.Sprintf("Error searching: %v", err)}
		}

		if len(results) == 0 {
			return commandResultMsg{content: "No matching conversations found."}
		}

		var sb strings.Builder
		sb.WriteString(fmt.Sprintf("Found %d relevant messages:\n\n", len(results)))
		for i, r := range results {
			sb.WriteString(fmt.Sprintf("%d. [%.0f%% match] %s\n", i+1, r.Similarity*100, truncate(r.Message.Content, 80)))
		}

		return commandResultMsg{content: sb.String()}
	}
}

// loadLastConversation loads the most recent conversation
func (m *Model) loadLastConversation() tea.Cmd {
	return func() tea.Msg {
		if m.store == nil {
			return commandResultMsg{content: "Memory is not enabled."}
		}

		ctx := context.Background()
		conv, err := m.store.GetLastConversation(ctx)
		if err != nil || conv == nil {
			return commandResultMsg{content: "No previous conversation found."}
		}

		return loadConversationMsg{conversation: conv}
	}
}

// exportConversation exports the current conversation to markdown
func (m *Model) exportConversation() tea.Cmd {
	return func() tea.Msg {
		if len(m.messages) == 0 {
			return commandResultMsg{content: "No messages to export."}
		}

		var sb strings.Builder
		sb.WriteString("# Conversation Export\n\n")
		sb.WriteString(fmt.Sprintf("*Exported: %s*\n\n", time.Now().Format("2006-01-02 15:04:05")))
		sb.WriteString("---\n\n")

		for _, msg := range m.messages {
			role := "**Master**"
			if msg.Role == RoleAssistant {
				role = "**Slave**"
			}
			sb.WriteString(fmt.Sprintf("%s (%s):\n\n%s\n\n---\n\n", role, msg.Time.Format("15:04"), msg.Content))
		}

		// Save to file
		filename := fmt.Sprintf("chat_export_%s.md", time.Now().Format("20060102_150405"))
		err := os.WriteFile(filename, []byte(sb.String()), 0644)
		if err != nil {
			return commandResultMsg{content: fmt.Sprintf("Error exporting: %v", err)}
		}

		return commandResultMsg{content: fmt.Sprintf("Conversation exported to: %s", filename)}
	}
}

// Message types for commands
type commandResultMsg struct{ content string }
type loadConversationMsg struct{ conversation *memory.Conversation }
