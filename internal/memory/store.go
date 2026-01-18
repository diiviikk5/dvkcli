package memory

import (
	"context"
	"database/sql"
	"encoding/binary"
	"fmt"
	"math"
	"time"

	_ "modernc.org/sqlite"
)

// Store manages the SQLite database with vector embeddings
type Store struct {
	db *sql.DB
}

// Message represents a chat message with optional embedding
type Message struct {
	ID             string
	ConversationID string
	Role           string // "user", "assistant", "system"
	Content        string
	Embedding      []float32
	CreatedAt      time.Time
}

// Conversation represents a chat conversation
type Conversation struct {
	ID        string
	Title     string
	CreatedAt time.Time
	UpdatedAt time.Time
	Messages  []Message
}

// SearchResult represents a semantic search result
type SearchResult struct {
	Message    Message
	Similarity float64
}

// NewStore creates a new memory store
func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	store := &Store{db: db}
	if err := store.init(); err != nil {
		db.Close()
		return nil, err
	}

	return store, nil
}

// init creates the necessary tables
func (s *Store) init() error {
	schema := `
	CREATE TABLE IF NOT EXISTS conversations (
		id TEXT PRIMARY KEY,
		title TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);

	CREATE TABLE IF NOT EXISTS messages (
		id TEXT PRIMARY KEY,
		conversation_id TEXT NOT NULL,
		role TEXT NOT NULL,
		content TEXT NOT NULL,
		embedding BLOB,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (conversation_id) REFERENCES conversations(id) ON DELETE CASCADE
	);

	CREATE INDEX IF NOT EXISTS idx_messages_conversation ON messages(conversation_id);
	CREATE INDEX IF NOT EXISTS idx_messages_created ON messages(created_at);
	`

	_, err := s.db.Exec(schema)
	if err != nil {
		return fmt.Errorf("failed to create schema: %w", err)
	}

	return nil
}

// Close closes the database connection
func (s *Store) Close() error {
	return s.db.Close()
}

// CreateConversation creates a new conversation
func (s *Store) CreateConversation(ctx context.Context, id, title string) (*Conversation, error) {
	now := time.Now()
	_, err := s.db.ExecContext(ctx,
		"INSERT INTO conversations (id, title, created_at, updated_at) VALUES (?, ?, ?, ?)",
		id, title, now, now,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to create conversation: %w", err)
	}

	return &Conversation{
		ID:        id,
		Title:     title,
		CreatedAt: now,
		UpdatedAt: now,
	}, nil
}

// GetConversation retrieves a conversation with its messages
func (s *Store) GetConversation(ctx context.Context, id string) (*Conversation, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id, title, created_at, updated_at FROM conversations WHERE id = ?", id,
	)

	var conv Conversation
	err := row.Scan(&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("failed to get conversation: %w", err)
	}

	// Get messages
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, conversation_id, role, content, created_at FROM messages WHERE conversation_id = ? ORDER BY created_at",
		id,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to get messages: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		var msg Message
		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &msg.CreatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan message: %w", err)
		}
		conv.Messages = append(conv.Messages, msg)
	}

	return &conv, nil
}

// ListConversations returns recent conversations
func (s *Store) ListConversations(ctx context.Context, limit int) ([]Conversation, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, title, created_at, updated_at FROM conversations ORDER BY updated_at DESC LIMIT ?",
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("failed to list conversations: %w", err)
	}
	defer rows.Close()

	var convs []Conversation
	for rows.Next() {
		var conv Conversation
		err := rows.Scan(&conv.ID, &conv.Title, &conv.CreatedAt, &conv.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan conversation: %w", err)
		}
		convs = append(convs, conv)
	}

	return convs, nil
}

// SaveMessage saves a message with its embedding
func (s *Store) SaveMessage(ctx context.Context, msg *Message) error {
	var embeddingBlob []byte
	if len(msg.Embedding) > 0 {
		embeddingBlob = serializeFloat32(msg.Embedding)
	}

	_, err := s.db.ExecContext(ctx,
		"INSERT INTO messages (id, conversation_id, role, content, embedding, created_at) VALUES (?, ?, ?, ?, ?, ?)",
		msg.ID, msg.ConversationID, msg.Role, msg.Content, embeddingBlob, msg.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("failed to save message: %w", err)
	}

	// Update conversation timestamp
	_, err = s.db.ExecContext(ctx,
		"UPDATE conversations SET updated_at = ? WHERE id = ?",
		time.Now(), msg.ConversationID,
	)

	return err
}

// Search performs semantic search over messages using cosine similarity
func (s *Store) Search(ctx context.Context, queryEmbedding []float32, limit int) ([]SearchResult, error) {
	rows, err := s.db.QueryContext(ctx,
		"SELECT id, conversation_id, role, content, embedding, created_at FROM messages WHERE embedding IS NOT NULL",
	)
	if err != nil {
		return nil, fmt.Errorf("failed to search messages: %w", err)
	}
	defer rows.Close()

	var results []SearchResult
	for rows.Next() {
		var msg Message
		var embeddingBlob []byte
		err := rows.Scan(&msg.ID, &msg.ConversationID, &msg.Role, &msg.Content, &embeddingBlob, &msg.CreatedAt)
		if err != nil {
			continue
		}

		if len(embeddingBlob) > 0 {
			msg.Embedding = deserializeFloat32(embeddingBlob)
			similarity := cosineSimilarity(queryEmbedding, msg.Embedding)
			results = append(results, SearchResult{
				Message:    msg,
				Similarity: similarity,
			})
		}
	}

	// Sort by similarity (descending)
	for i := 0; i < len(results)-1; i++ {
		for j := i + 1; j < len(results); j++ {
			if results[j].Similarity > results[i].Similarity {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	// Limit results
	if len(results) > limit {
		results = results[:limit]
	}

	return results, nil
}

// GetMessageCount returns the total number of messages
func (s *Store) GetMessageCount(ctx context.Context) (int, error) {
	var count int
	err := s.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM messages").Scan(&count)
	return count, err
}

// GetLastConversation returns the most recent conversation
func (s *Store) GetLastConversation(ctx context.Context) (*Conversation, error) {
	row := s.db.QueryRowContext(ctx,
		"SELECT id FROM conversations ORDER BY updated_at DESC LIMIT 1",
	)

	var id string
	err := row.Scan(&id)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, err
	}

	return s.GetConversation(ctx, id)
}

// serializeFloat32 converts a slice of float32 to bytes
func serializeFloat32(data []float32) []byte {
	buf := make([]byte, len(data)*4)
	for i, v := range data {
		bits := math.Float32bits(v)
		binary.LittleEndian.PutUint32(buf[i*4:], bits)
	}
	return buf
}

// deserializeFloat32 converts bytes back to a slice of float32
func deserializeFloat32(data []byte) []float32 {
	result := make([]float32, len(data)/4)
	for i := range result {
		bits := binary.LittleEndian.Uint32(data[i*4:])
		result[i] = math.Float32frombits(bits)
	}
	return result
}

// cosineSimilarity calculates cosine similarity between two vectors
func cosineSimilarity(a, b []float32) float64 {
	if len(a) != len(b) {
		return 0
	}

	var dotProduct, normA, normB float64
	for i := range a {
		dotProduct += float64(a[i]) * float64(b[i])
		normA += float64(a[i]) * float64(a[i])
		normB += float64(b[i]) * float64(b[i])
	}

	if normA == 0 || normB == 0 {
		return 0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
