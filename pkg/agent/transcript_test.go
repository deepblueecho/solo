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

func TestOpenClawTranscriptPath(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	path := filepath.Join(home, ".openclaw", "agents", "main", "sessions", "solo-123.trajectory.jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := TranscriptPath("openclaw", "", "solo-123"); got != path {
		t.Fatalf("TranscriptPath() for openclaw = %q, want %q", got, path)
	}
}

func TestOpenClawTranscriptPathPrefersExactJSONL(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".openclaw", "agents", "main", "sessions")
	exact := filepath.Join(dir, "solo-123.jsonl")
	trajectory := filepath.Join(dir, "solo-123.trajectory.jsonl")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(exact, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trajectory, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := OpenClawTranscriptPath("", "solo-123"); got != exact {
		t.Fatalf("OpenClawTranscriptPath() = %q, want %q", got, exact)
	}
}

func TestOpenClawTranscriptPathUsesPointer(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".openclaw", "agents", "main", "sessions")
	runtimeFile := filepath.Join(dir, "runtime", "solo-123.trajectory.jsonl")
	pointer := filepath.Join(dir, "solo-123.trajectory-path.json")
	if err := os.MkdirAll(filepath.Dir(runtimeFile), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(runtimeFile, []byte("{}\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(pointer, []byte(`{"runtimeFile":"`+runtimeFile+`"}`), 0644); err != nil {
		t.Fatal(err)
	}
	if got := OpenClawTranscriptPath("", "solo-123"); got != runtimeFile {
		t.Fatalf("OpenClawTranscriptPath() = %q, want %q", got, runtimeFile)
	}
}

func TestOpenClawTranscriptPathFallsBackToWorkspace(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".openclaw", "agents", "main", "sessions")
	workspaceDir := filepath.Join(home, ".solo", "agents", "agent-1", "workspace")
	path := filepath.Join(dir, "runtime-session.jsonl")
	trajectory := filepath.Join(dir, "runtime-session.trajectory.jsonl")
	if err := os.MkdirAll(dir, 0755); err != nil {
		t.Fatal(err)
	}
	raw := `{"type":"message","message":{"role":"user","content":"Workspace: ` + workspaceDir + `"}}` + "\n"
	if err := os.WriteFile(path, []byte(raw), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(trajectory, []byte(`{"type":"prompt.submitted","data":{"prompt":"`+workspaceDir+`"}}`+"\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if got := OpenClawTranscriptPath(workspaceDir, "provider-session"); got != path {
		t.Fatalf("OpenClawTranscriptPath() = %q, want %q", got, path)
	}
}

func TestTranscriptPathSkipsHermesExport(t *testing.T) {
	if got := TranscriptPath("hermes", "", "hermes-123"); got != "" {
		t.Fatalf("TranscriptPath() for hermes = %q, want empty", got)
	}
}
