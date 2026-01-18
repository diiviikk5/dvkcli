package tui

import "github.com/charmbracelet/lipgloss"

// Royal Theme - Wine Red + Gold for Master Divik
// Designed for a regal, powerful aesthetic

var (
	// Primary palette - Wine & Gold
	Primary    = lipgloss.Color("#8B0000") // Dark wine red
	Secondary  = lipgloss.Color("#FFD700") // Royal gold
	Tertiary   = lipgloss.Color("#DC143C") // Crimson accent
	Background = lipgloss.Color("#0D0D0D") // Deep black
	Surface    = lipgloss.Color("#1A0A0A") // Dark wine surface
	SurfaceAlt = lipgloss.Color("#2D1515") // Lighter wine surface

	// Text colors
	Text     = lipgloss.Color("#F5E6D3") // Warm cream text
	TextBold = lipgloss.Color("#FFFAF0") // Floral white
	Muted    = lipgloss.Color("#8B7355") // Muted bronze
	Subtle   = lipgloss.Color("#5C4033") // Dark bronze

	// Status colors
	Success = lipgloss.Color("#228B22") // Forest green
	Warning = lipgloss.Color("#FFD700") // Gold
	Error   = lipgloss.Color("#DC143C") // Crimson
	Info    = lipgloss.Color("#CD853F") // Peru/Bronze

	// Gradient colors for logo
	GradientStart = lipgloss.Color("#8B0000") // Wine
	GradientMid   = lipgloss.Color("#B22222") // Firebrick
	GradientEnd   = lipgloss.Color("#FFD700") // Gold
)

// Pre-defined styles for consistent UI
var (
	// Logo style with bold gradient-like effect
	LogoStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(Primary)

	// Title styles
	TitleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(TextBold).
			MarginBottom(1)

	SubtitleStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Italic(true)

	// Message styles
	UserMessageStyle = lipgloss.NewStyle().
				Foreground(Tertiary).
				Bold(true)

	AssistantMessageStyle = lipgloss.NewStyle().
				Foreground(Text)

	SystemMessageStyle = lipgloss.NewStyle().
				Foreground(Muted).
				Italic(true)

	// Input area
	InputStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Primary).
			Padding(0, 1)

	InputPromptStyle = lipgloss.NewStyle().
				Foreground(Secondary).
				Bold(true)

	// Status bar
	StatusBarStyle = lipgloss.NewStyle().
			Background(Surface).
			Foreground(Muted).
			Padding(0, 1)

	StatusItemStyle = lipgloss.NewStyle().
			Foreground(Text).
			Padding(0, 1)

	StatusActiveStyle = lipgloss.NewStyle().
				Foreground(Success).
				Bold(true)

	StatusErrorStyle = lipgloss.NewStyle().
				Foreground(Error).
				Bold(true)

	// Panels and containers
	PanelStyle = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(Subtle).
			Padding(1, 2)

	HighlightPanelStyle = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(Primary).
				Padding(1, 2)

	// Code blocks
	CodeBlockStyle = lipgloss.NewStyle().
			Background(Surface).
			Foreground(Tertiary).
			Padding(1, 2)

	InlineCodeStyle = lipgloss.NewStyle().
			Background(Surface).
			Foreground(Secondary).
			Padding(0, 1)

	// Spinner/Loading
	SpinnerStyle = lipgloss.NewStyle().
			Foreground(Primary)

	// Help text
	HelpStyle = lipgloss.NewStyle().
			Foreground(Subtle)

	HelpKeyStyle = lipgloss.NewStyle().
			Foreground(Muted).
			Bold(true)

	// Memory/context indicator
	MemoryStyle = lipgloss.NewStyle().
			Foreground(Info).
			Bold(true)

	MemoryCountStyle = lipgloss.NewStyle().
				Foreground(Tertiary)
)

// ASCII Art Logo with gradient effect - DIVIK
const LogoASCII = `
 ██████╗  ██╗ ██╗   ██╗ ██╗ ██╗  ██╗
 ██╔══██╗ ██║ ██║   ██║ ██║ ██║ ██╔╝
 ██║  ██║ ██║ ██║   ██║ ██║ █████╔╝ 
 ██║  ██║ ██║ ╚██╗ ██╔╝ ██║ ██╔═██╗ 
 ██████╔╝ ██║  ╚████╔╝  ██║ ██║  ██╗
 ╚═════╝  ╚═╝   ╚═══╝   ╚═╝ ╚═╝  ╚═╝`

// Compact logo for header
const LogoCompact = "DIVIK"

// RenderGradientText applies a gradient effect to text (purple -> pink -> cyan)
func RenderGradientText(text string) string {
	colors := []lipgloss.Color{Primary, Secondary, Tertiary}
	result := ""
	for i, char := range text {
		color := colors[i%len(colors)]
		result += lipgloss.NewStyle().Foreground(color).Render(string(char))
	}
	return result
}

// RenderLogo renders the full ASCII logo with gradient coloring
func RenderLogo() string {
	lines := []string{
		" ██████╗  ██╗ ██╗   ██╗ ██╗ ██╗  ██╗",
		" ██╔══██╗ ██║ ██║   ██║ ██║ ██║ ██╔╝",
		" ██║  ██║ ██║ ██║   ██║ ██║ █████╔╝ ",
		" ██║  ██║ ██║ ╚██╗ ██╔╝ ██║ ██╔═██╗ ",
		" ██████╔╝ ██║  ╚████╔╝  ██║ ██║  ██╗",
		" ╚═════╝  ╚═╝   ╚═══╝   ╚═╝ ╚═╝  ╚═╝",
	}

	colors := []lipgloss.Color{
		"#8B0000", // Dark wine
		"#A52A2A", // Brown/wine
		"#B22222", // Firebrick
		"#CD5C5C", // Indian red
		"#DAA520", // Goldenrod
		"#FFD700", // Gold
	}

	result := ""
	for i, line := range lines {
		colorIdx := i % len(colors)
		styled := lipgloss.NewStyle().Foreground(colors[colorIdx]).Bold(true).Render(line)
		result += styled + "\n"
	}
	return result
}

// RenderCompactLogo renders a small logo for the header
func RenderCompactLogo() string {
	return lipgloss.NewStyle().
		Bold(true).
		Foreground(Primary).
		Render("✦ ") +
		RenderGradientText("DIVIK")
}

// SpinnerFrames for animated loading indicator
var SpinnerFrames = []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}

// TypingIndicator frames
var TypingFrames = []string{"●○○", "○●○", "○○●", "○●○"}

// ModelIcon returns an icon for model status
func ModelIcon(connected bool) string {
	if connected {
		return lipgloss.NewStyle().Foreground(Success).Render("●")
	}
	return lipgloss.NewStyle().Foreground(Error).Render("○")
}
