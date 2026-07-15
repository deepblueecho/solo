package agent

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestClaudeTranscriptPath(t *testing.T) {
	got := ClaudeTranscriptPath("/Users/me/.solo/agents/a1/workspace", "session-123")
	want := "/Users/me/.claude/projects/-Users-me--solo-agents-a1-workspace/session-123.jsonl"
	if got != want {
		t.Fatalf("ClaudeTranscriptPath() = %q, want %q", got, want)
	}
}

func TestTranscriptPath(t *testing.T) {
	got := TranscriptPath("claude", "/Users/me/.solo/agents/a1/workspace", "session-123")
	want := "/Users/me/.claude/projects/-Users-me--solo-agents-a1-workspace/session-123.jsonl"
	if got != want {
		t.Fatalf("TranscriptPath() = %q, want %q", got, want)
	}
	codexHome := filepath.Join(t.TempDir(), ".codex")
	path := filepath.Join(codexHome, "sessions", "2026", "06", "28", "rollout-2026-06-28T12-00-00-session-123.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)
	if got := TranscriptPath("codex", "/tmp/workspace", "session-123"); got != path {
		t.Fatalf("TranscriptPath() for codex = %q, want %q", got, path)
	}
}

func TestOpenCodeTranscriptPath(t *testing.T) {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		t.Skip("sqlite3 not installed")
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	dbPath := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if err := os.MkdirAll(filepath.Dir(dbPath), 0755); err != nil {
		t.Fatal(err)
	}
	sql := `
CREATE TABLE message (id TEXT PRIMARY KEY, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, data TEXT NOT NULL);
CREATE TABLE part (id TEXT PRIMARY KEY, message_id TEXT NOT NULL, session_id TEXT NOT NULL, time_created INTEGER NOT NULL, data TEXT NOT NULL);
INSERT INTO message (id, session_id, time_created, data) VALUES ('msg_1', 'ses_123', 1760000000000, '{"role":"user"}');
INSERT INTO part (id, message_id, session_id, time_created, data) VALUES ('part_1', 'msg_1', 'ses_123', 1760000000001, '{"type":"text","text":"hello"}');
`
	cmd := exec.Command("sqlite3", dbPath, sql)
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("sqlite3: %v\n%s", err, output)
	}

	want := filepath.Join(home, ".solo", "opencode-transcripts", "ses_123.jsonl")
	if got := TranscriptPath("opencode", "", "ses_123"); got != want {
		t.Fatalf("TranscriptPath() for opencode = %q, want %q", got, want)
	}
	data, err := os.ReadFile(want)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"type":"user"`) || !strings.Contains(string(data), `"content":"hello"`) {
		t.Fatalf("exported jsonl = %s", data)
	}
}

func TestTranscriptPathSkipsHermesExport(t *testing.T) {
	if got := TranscriptPath("hermes", "", "hermes-123"); got != "" {
		t.Fatalf("TranscriptPath() for hermes = %q, want empty", got)
	}
}
