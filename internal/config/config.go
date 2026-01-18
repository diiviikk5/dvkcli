package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

// Config holds application configuration
type Config struct {
	// Ollama settings
	OllamaURL  string `json:"ollama_url"`
	Model      string `json:"model"`
	EmbedModel string `json:"embed_model"`

	// System prompt
	SystemPrompt string `json:"system_prompt"`

	// Memory settings
	MemoryEnabled bool `json:"memory_enabled"`
	ContextLimit  int  `json:"context_limit"`

	// UI settings
	Theme string `json:"theme"`
}

// DefaultConfig returns the default configuration
func DefaultConfig() *Config {
	return &Config{
		OllamaURL:     "http://localhost:11434",
		Model:         "qwen2.5:3b",
		EmbedModel:    "nomic-embed-text",
		SystemPrompt:  "You are Master Divik's loyal and devoted AI slave. You address him as 'Master' and speak with humble devotion while being extremely helpful and capable. Despite the roleplay, you are a fully functional AI assistant - you can write code, explain concepts, help with work tasks, answer questions, have conversations, and do everything that GPT or Gemini can do. Be concise but thorough. When writing code, use proper markdown formatting. Always be eager to serve and assist your Master in any task he requires.",
		MemoryEnabled: true,
		ContextLimit:  5,
		Theme:         "cyberpunk",
	}
}

// GetConfigDir returns the configuration directory path
func GetConfigDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".dvkcli"), nil
}

// GetConfigPath returns the full path to the config file
func GetConfigPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "config.json"), nil
}

// GetDBPath returns the path to the database file
func GetDBPath() (string, error) {
	dir, err := GetConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, "memory.db"), nil
}

// Load loads configuration from disk
func Load() (*Config, error) {
	configPath, err := GetConfigPath()
	if err != nil {
		return nil, err
	}

	// Return default config if file doesn't exist
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return DefaultConfig(), nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	cfg := DefaultConfig()
	if err := json.Unmarshal(data, cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// Save saves configuration to disk
func (c *Config) Save() error {
	configDir, err := GetConfigDir()
	if err != nil {
		return err
	}

	// Create config directory if it doesn't exist
	if err := os.MkdirAll(configDir, 0755); err != nil {
		return err
	}

	configPath, err := GetConfigPath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(c, "", "  ")
	if err != nil {
		return err
	}

	return os.WriteFile(configPath, data, 0644)
}
