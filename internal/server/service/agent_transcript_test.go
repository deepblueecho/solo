package service

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestReadAgentTranscriptParsesClaudeJSONL(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	raw := `{"type":"user","timestamp":"2026-06-25T09:43:53Z","message":{"content":"hello"}}
{"type":"assistant","timestamp":"2026-06-25T09:44:03Z","message":{"usage":{"input_tokens":12,"output_tokens":3,"cache_creation_input_tokens":4,"cache_read_input_tokens":5},"content":[{"type":"thinking","thinking":"checking"},{"type":"tool_use","id":"toolu_1","name":"Bash","input":{"command":"pwd"}},{"type":"text","text":"done"}]}}
{"type":"user","timestamp":"2026-06-25T09:44:04Z","message":{"content":[{"type":"tool_result","tool_use_id":"toolu_1","content":"workspace"}]}}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAgentTranscript(path, 100)
	if err != nil {
		t.Fatalf("ReadAgentTranscript: %v", err)
	}
	if len(entries) != 5 {
		t.Fatalf("entry count = %d, want 5: %+v", len(entries), entries)
	}
	if entries[1].Type != "thinking" || entries[1].Text != "checking" {
		t.Fatalf("thinking entry = %+v", entries[1])
	}
	if entries[1].Usage == nil || entries[1].Usage.InputTokens != 12 || entries[1].Usage.OutputTokens != 3 || entries[1].Usage.CacheCreationInputTokens != 4 || entries[1].Usage.CacheReadInputTokens != 5 {
		t.Fatalf("usage entry = %+v", entries[1].Usage)
	}
	if entries[2].Type != "tool_use" || entries[2].ToolName != "Bash" || string(entries[2].Input) != `{"command":"pwd"}` {
		t.Fatalf("tool_use entry = %+v", entries[2])
	}
	if entries[4].Type != "tool_result" || entries[4].ToolID != "toolu_1" || entries[4].Text != "workspace" {
		t.Fatalf("tool_result entry = %+v", entries[4])
	}
}

func TestReadAgentTranscriptParsesQueuedCommandPrompt(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	raw := `{"type":"attachment","timestamp":"2026-06-25T10:00:00Z","attachment":{"type":"queued_command","prompt":[{"type":"text","text":"User: Generate a Solo artifact for this task.\n\nAssistant:"}],"commandMode":"prompt"}}` + "\n"
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	entries, err := ReadAgentTranscript(path, 100)
	if err != nil {
		t.Fatalf("ReadAgentTranscript: %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("entry count = %d, want 1: %+v", len(entries), entries)
	}
	if entries[0].Role != "user" || entries[0].Type != "text" || entries[0].Text == "" {
		t.Fatalf("queued command entry = %+v", entries[0])
	}
}

func TestReadAgentTranscriptReturnsEmptyArrayForMissingPath(t *testing.T) {
	entries, err := ReadAgentTranscript(filepath.Join(t.TempDir(), "missing.jsonl"), 100)
	if err != nil {
		t.Fatalf("ReadAgentTranscript missing path: %v", err)
	}
	if entries == nil {
		t.Fatal("ReadAgentTranscript returned nil, want empty slice")
	}
	if len(entries) != 0 {
		t.Fatalf("entry count = %d, want 0", len(entries))
	}
}

func TestReadHermesTranscriptReadsStateDBDirectly(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dbPath := filepath.Join(home, ".hermes", "state.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	sql := `
CREATE TABLE sessions (id TEXT PRIMARY KEY);
CREATE TABLE messages (
  id INTEGER PRIMARY KEY AUTOINCREMENT,
  session_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT,
  tool_name TEXT,
  tool_calls TEXT,
  reasoning TEXT,
  reasoning_content TEXT,
  timestamp REAL NOT NULL,
  active INTEGER NOT NULL DEFAULT 1
);
INSERT INTO sessions (id) VALUES ('hermes-1');
INSERT INTO messages (session_id, role, content, timestamp) VALUES ('hermes-1', 'user', 'hello', 1760000000.5);
INSERT INTO messages (session_id, role, content, timestamp) VALUES ('hermes-1', 'assistant', 'world', 1760000001.5);
`
	cmd := exec.Command("sqlite3", dbPath, sql)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3: %v\n%s", err, output)
	}

	entries, err := ReadHermesTranscript("hermes-1", 10)
	if err != nil {
		t.Fatalf("ReadHermesTranscript: %v", err)
	}
	if len(entries) != 2 || entries[0].Text != "hello" || entries[1].Text != "world" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestReadAgentTranscriptWindowFiltersSessionJSONLToRun(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	raw := `{"type":"user","timestamp":"2026-06-25T09:00:00Z","message":{"content":"old turn"}}
{"type":"user","timestamp":"2026-06-25T10:00:00Z","message":{"content":"current turn"}}
{"type":"assistant","timestamp":"2026-06-25T10:00:05Z","message":{"content":[{"type":"text","text":"current answer"}]}}
{"type":"user","timestamp":"2026-06-25T11:00:00Z","message":{"content":"next turn"}}
`
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}
	start := time.Date(2026, 6, 25, 9, 59, 59, 0, time.UTC)
	end := time.Date(2026, 6, 25, 10, 0, 10, 0, time.UTC)

	entries, err := ReadAgentTranscriptWindow(path, start, end, 100)
	if err != nil {
		t.Fatalf("ReadAgentTranscriptWindow: %v", err)
	}
	if len(entries) != 2 {
		t.Fatalf("entry count = %d, want 2: %+v", len(entries), entries)
	}
	if entries[0].Text != "current turn" || entries[1].Text != "current answer" {
		t.Fatalf("entries = %+v", entries)
	}
}

func TestReadAgentTranscriptCacheInvalidatesWhenFileChanges(t *testing.T) {
	path := filepath.Join(t.TempDir(), "session.jsonl")
	first := `{"type":"user","timestamp":"2026-06-25T09:00:00Z","message":{"content":"first"}}` + "\n"
	if err := os.WriteFile(path, []byte(first), 0600); err != nil {
		t.Fatal(err)
	}
	firstTime := time.Date(2026, 6, 25, 9, 0, 0, 0, time.UTC)
	if err := os.Chtimes(path, firstTime, firstTime); err != nil {
		t.Fatal(err)
	}
	entries, err := ReadAgentTranscript(path, 100)
	if err != nil {
		t.Fatalf("ReadAgentTranscript first: %v", err)
	}
	if len(entries) != 1 || entries[0].Text != "first" {
		t.Fatalf("first entries = %+v", entries)
	}

	second := `{"type":"user","timestamp":"2026-06-25T09:00:01Z","message":{"content":"second"}}` + "\n"
	if err := os.WriteFile(path, []byte(second), 0600); err != nil {
		t.Fatal(err)
	}
	secondTime := firstTime.Add(time.Second)
	if err := os.Chtimes(path, secondTime, secondTime); err != nil {
		t.Fatal(err)
	}
	entries, err = ReadAgentTranscript(path, 100)
	if err != nil {
		t.Fatalf("ReadAgentTranscript second: %v", err)
	}
	if len(entries) != 1 || entries[0].Text != "second" {
		t.Fatalf("second entries = %+v", entries)
	}
}
