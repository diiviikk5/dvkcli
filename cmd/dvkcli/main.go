package main

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/diiviikk5/dvkcli/internal/config"
	"github.com/diiviikk5/dvkcli/internal/memory"
	"github.com/diiviikk5/dvkcli/internal/ollama"
	"github.com/diiviikk5/dvkcli/internal/tui"
)

const version = "0.1.0"

func main() {
	// Handle version flag
	if len(os.Args) > 1 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Printf("dvkcli v%s\n", version)
		fmt.Println("Local-first AI terminal assistant")
		fmt.Println("https://github.com/diiviikk5/dvkcli")
		os.Exit(0)
	}

	// Load configuration
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error loading config: %v\n", err)
		os.Exit(1)
	}

	// Initialize Ollama client
	client, err := ollama.NewClient(cfg.OllamaURL, cfg.Model, cfg.EmbedModel)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating Ollama client: %v\n", err)
		os.Exit(1)
	}

	// Initialize memory store
	var store *memory.Store
	if cfg.MemoryEnabled {
		dbPath, err := config.GetDBPath()
		if err == nil {
			// Ensure config directory exists
			configDir, _ := config.GetConfigDir()
			os.MkdirAll(configDir, 0755)

			store, err = memory.NewStore(dbPath)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: Could not initialize memory store: %v\n", err)
				// Continue without memory
			}
		}
	}

	// Ensure store is closed on exit
	if store != nil {
		defer store.Close()
	}

	// Print welcome logo
	fmt.Print("\033[H\033[2J") // Clear screen
	fmt.Println(tui.RenderLogo())
	fmt.Println()

	// Create and run the TUI
	model := tui.New(client, store, cfg)
	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running dvkcli: %v\n", err)
		os.Exit(1)
	}

	// Save config on exit
	cfg.Save()
}
