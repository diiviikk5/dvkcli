package ollama

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/ollama/ollama/api"
)

// Client wraps the Ollama API client
type Client struct {
	api        *api.Client
	Model      string
	EmbedModel string
	BaseURL    string
	IsCloud    bool
	apiKey     string
}

// Model represents an Ollama model
type Model struct {
	Name       string
	Size       int64
	ModifiedAt time.Time
}

// NewClient creates a new Ollama client
// It auto-detects cloud mode if OLLAMA_API_KEY is set
func NewClient(baseURL, model, embedModel string) (*Client, error) {
	apiKey := os.Getenv("OLLAMA_API_KEY")
	isCloud := apiKey != ""

	// Use cloud URL if API key is set and no custom URL provided
	if isCloud && baseURL == "" {
		baseURL = "https://api.ollama.com"
	} else if baseURL == "" {
		baseURL = "http://localhost:11434"
	}

	if model == "" {
		if isCloud {
			model = "llama3.3" // Good cloud model
		} else {
			model = "qwen2.5:3b"
		}
	}
	if embedModel == "" {
		embedModel = "nomic-embed-text"
	}

	parsedURL, err := url.Parse(baseURL)
	if err != nil {
		return nil, fmt.Errorf("invalid base URL: %w", err)
	}

	// Create HTTP client with auth header for cloud
	httpClient := &http.Client{
		Timeout: 5 * time.Minute,
	}

	if isCloud {
		httpClient.Transport = &authTransport{
			apiKey: apiKey,
			base:   http.DefaultTransport,
		}
	}

	client := api.NewClient(parsedURL, httpClient)

	return &Client{
		api:        client,
		Model:      model,
		EmbedModel: embedModel,
		BaseURL:    baseURL,
		IsCloud:    isCloud,
		apiKey:     apiKey,
	}, nil
}

// authTransport adds Authorization header to requests
type authTransport struct {
	apiKey string
	base   http.RoundTripper
}

func (t *authTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	req.Header.Set("Authorization", "Bearer "+t.apiKey)
	return t.base.RoundTrip(req)
}

// IsConnected checks if Ollama is running and accessible
func (c *Client) IsConnected(ctx context.Context) bool {
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	_, err := c.api.List(ctx)
	return err == nil
}

// ListModels returns available local models
func (c *Client) ListModels(ctx context.Context) ([]Model, error) {
	resp, err := c.api.List(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to list models: %w", err)
	}

	models := make([]Model, 0, len(resp.Models))
	for _, m := range resp.Models {
		models = append(models, Model{
			Name:       m.Name,
			Size:       m.Size,
			ModifiedAt: m.ModifiedAt,
		})
	}
	return models, nil
}

// StreamResponse represents a chunk of streamed response
type StreamResponse struct {
	Content string
	Done    bool
	Error   error
}

// Stream sends a prompt and streams response chunks via a channel
func (c *Client) Stream(ctx context.Context, prompt string, systemPrompt string) <-chan StreamResponse {
	ch := make(chan StreamResponse)

	go func() {
		defer close(ch)

		messages := []api.Message{}

		if systemPrompt != "" {
			messages = append(messages, api.Message{
				Role:    "system",
				Content: systemPrompt,
			})
		}

		messages = append(messages, api.Message{
			Role:    "user",
			Content: prompt,
		})

		req := &api.ChatRequest{
			Model:    c.Model,
			Messages: messages,
			Stream:   boolPtr(true),
		}

		err := c.api.Chat(ctx, req, func(resp api.ChatResponse) error {
			ch <- StreamResponse{
				Content: resp.Message.Content,
				Done:    resp.Done,
			}
			return nil
		})

		if err != nil {
			ch <- StreamResponse{Error: err}
		}
	}()

	return ch
}

// StreamWithHistory sends a conversation with history and streams response
func (c *Client) StreamWithHistory(ctx context.Context, messages []api.Message) <-chan StreamResponse {
	ch := make(chan StreamResponse)

	go func() {
		defer close(ch)

		req := &api.ChatRequest{
			Model:    c.Model,
			Messages: messages,
			Stream:   boolPtr(true),
		}

		err := c.api.Chat(ctx, req, func(resp api.ChatResponse) error {
			ch <- StreamResponse{
				Content: resp.Message.Content,
				Done:    resp.Done,
			}
			return nil
		})

		if err != nil {
			ch <- StreamResponse{Error: err}
		}
	}()

	return ch
}

// Embed generates embeddings for text
func (c *Client) Embed(ctx context.Context, text string) ([]float32, error) {
	req := &api.EmbedRequest{
		Model: c.EmbedModel,
		Input: text,
	}

	resp, err := c.api.Embed(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}

	if len(resp.Embeddings) == 0 {
		return nil, fmt.Errorf("no embeddings returned")
	}

	// Convert float64 to float32
	embedding := make([]float32, len(resp.Embeddings[0]))
	for i, v := range resp.Embeddings[0] {
		embedding[i] = float32(v)
	}

	return embedding, nil
}

// SetModel changes the active model
func (c *Client) SetModel(model string) {
	c.Model = model
}

// Chat sends a conversation and returns the full response (non-streaming)
func (c *Client) Chat(ctx context.Context, messages []api.Message) (string, error) {
	var fullResponse strings.Builder

	req := &api.ChatRequest{
		Model:    c.Model,
		Messages: messages,
		Stream:   boolPtr(false),
	}

	err := c.api.Chat(ctx, req, func(resp api.ChatResponse) error {
		fullResponse.WriteString(resp.Message.Content)
		return nil
	})

	if err != nil {
		return "", fmt.Errorf("chat failed: %w", err)
	}

	return fullResponse.String(), nil
}

func boolPtr(b bool) *bool {
	return &b
}
