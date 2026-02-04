package context

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sync"
	"testing"

	"github.com/erg0nix/kontekst/internal/core"
)

func TestSessionFile_LoadTail_EmptyFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "empty.jsonl")

	if err := os.WriteFile(path, []byte{}, 0o644); err != nil {
		t.Fatal(err)
	}

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msgs != nil {
		t.Errorf("expected nil for empty file, got %v", msgs)
	}
}

func TestSessionFile_LoadTail_NonExistent(t *testing.T) {
	sf := NewSessionFile("/nonexistent/path/file.jsonl")
	msgs, err := sf.LoadTail(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if msgs != nil {
		t.Errorf("expected nil for non-existent file, got %v", msgs)
	}
}

func TestSessionFile_LoadTail_SingleMessage(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	msg := core.Message{Role: core.RoleUser, Content: "hello", Tokens: 10}
	writeMessages(t, path, msg)

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}

	if msgs[0].Content != "hello" {
		t.Errorf("expected content 'hello', got %q", msgs[0].Content)
	}
}

func TestSessionFile_LoadTail_MultipleMessages(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	messages := []core.Message{
		{Role: core.RoleUser, Content: "first", Tokens: 10},
		{Role: core.RoleAssistant, Content: "second", Tokens: 20},
		{Role: core.RoleUser, Content: "third", Tokens: 15},
	}
	writeMessages(t, path, messages...)

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(100)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages, got %d", len(msgs))
	}

	if msgs[0].Content != "first" || msgs[1].Content != "second" || msgs[2].Content != "third" {
		t.Errorf("messages not in expected order: %v", msgs)
	}
}

func TestSessionFile_LoadTail_BudgetExceeded(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	messages := []core.Message{
		{Role: core.RoleUser, Content: "first", Tokens: 50},
		{Role: core.RoleAssistant, Content: "second", Tokens: 50},
		{Role: core.RoleUser, Content: "third", Tokens: 50},
		{Role: core.RoleAssistant, Content: "fourth", Tokens: 50},
	}
	writeMessages(t, path, messages...)

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(120)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages within budget, got %d", len(msgs))
	}

	if msgs[0].Content != "third" || msgs[1].Content != "fourth" {
		t.Errorf("expected last 2 messages, got: %v", msgs)
	}
}

func TestSessionFile_LoadTail_ZeroBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "session.jsonl")

	messages := []core.Message{
		{Role: core.RoleUser, Content: "first", Tokens: 10},
		{Role: core.RoleAssistant, Content: "second", Tokens: 20},
	}
	writeMessages(t, path, messages...)

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(0)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 1 {
		t.Fatalf("expected 1 message (always include at least one), got %d", len(msgs))
	}

	if msgs[0].Content != "second" {
		t.Errorf("expected last message 'second', got %q", msgs[0].Content)
	}
}

func TestSessionFile_LoadTail_LargeFileSmallBudget(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "large.jsonl")

	var messages []core.Message
	for i := 0; i < 1000; i++ {
		messages = append(messages, core.Message{
			Role:    core.RoleUser,
			Content: "message content that is reasonably long to fill up space",
			Tokens:  100,
		})
	}
	writeMessages(t, path, messages...)

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(250)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages within budget (200 tokens), got %d", len(msgs))
	}
}

func TestSessionFile_LoadTail_MalformedJSON(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "malformed.jsonl")

	content := `{"role":"user","content":"valid1","tokens":10}
not valid json
{"role":"user","content":"valid2","tokens":10}
{broken json
{"role":"user","content":"valid3","tokens":10}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 3 {
		t.Fatalf("expected 3 valid messages, got %d", len(msgs))
	}
}

func TestSessionFile_Append_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "new.jsonl")

	sf := NewSessionFile(path)
	msg := core.Message{Role: core.RoleUser, Content: "hello", Tokens: 10}

	if err := sf.Append(msg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	var decoded core.Message
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("failed to decode: %v", err)
	}

	if decoded.Content != "hello" {
		t.Errorf("expected content 'hello', got %q", decoded.Content)
	}
}

func TestSessionFile_Append_AppendsToExisting(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "existing.jsonl")

	sf := NewSessionFile(path)
	msg1 := core.Message{Role: core.RoleUser, Content: "first", Tokens: 10}
	msg2 := core.Message{Role: core.RoleAssistant, Content: "second", Tokens: 20}

	if err := sf.Append(msg1); err != nil {
		t.Fatalf("unexpected error on first append: %v", err)
	}

	if err := sf.Append(msg2); err != nil {
		t.Fatalf("unexpected error on second append: %v", err)
	}

	msgs, err := sf.LoadTail(1000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}

	if msgs[0].Content != "first" || msgs[1].Content != "second" {
		t.Errorf("messages not in expected order: %v", msgs)
	}
}

func TestSessionFile_Append_ConcurrentWrites(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "concurrent.jsonl")

	sf := NewSessionFile(path)
	numWriters := 10
	messagesPerWriter := 100

	var wg sync.WaitGroup
	wg.Add(numWriters)

	for w := 0; w < numWriters; w++ {
		go func(writerID int) {
			defer wg.Done()
			for i := 0; i < messagesPerWriter; i++ {
				msg := core.Message{
					Role:    core.RoleUser,
					Content: "msg",
					Tokens:  1,
				}
				if err := sf.Append(msg); err != nil {
					t.Errorf("writer %d: unexpected error: %v", writerID, err)
					return
				}
			}
		}(w)
	}

	wg.Wait()

	msgs, err := sf.LoadAll()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	expected := numWriters * messagesPerWriter
	if len(msgs) != expected {
		t.Errorf("expected %d messages, got %d", expected, len(msgs))
	}
}

func TestSessionFile_LoadTail_ChunkBoundary(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "boundary.jsonl")

	var messages []core.Message
	for i := 0; i < 200; i++ {
		messages = append(messages, core.Message{
			Role:    core.RoleUser,
			Content: "x",
			Tokens:  10,
		})
	}
	writeMessages(t, path, messages...)

	sf := NewSessionFile(path)
	msgs, err := sf.LoadTail(10000)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(msgs) != 200 {
		t.Errorf("expected 200 messages, got %d", len(msgs))
	}
}

func writeMessages(t *testing.T, path string, messages ...core.Message) {
	t.Helper()

	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	defer file.Close()

	enc := json.NewEncoder(file)
	for _, msg := range messages {
		if err := enc.Encode(msg); err != nil {
			t.Fatal(err)
		}
	}
}
