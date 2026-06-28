package agent

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// TranscriptPath returns a provider-specific jsonl transcript path when Solo
// can derive one from the runtime workspace and provider session id.
func TranscriptPath(provider, workspaceDir, sessionID string) string {
	switch provider {
	case "claude":
		return ClaudeTranscriptPath(workspaceDir, sessionID)
	case "codex":
		return CodexTranscriptPath(sessionID)
	case "opencode":
		return OpenCodeTranscriptPath(sessionID)
	case "openclaw":
		return OpenClawTranscriptPath(workspaceDir, sessionID)
	default:
		return ""
	}
}

// ClaudeTranscriptPath returns Claude Code's jsonl path for a workspace/session.
func ClaudeTranscriptPath(workspaceDir, sessionID string) string {
	if workspaceDir == "" || sessionID == "" {
		return ""
	}
	return filepath.Join(claudeHomeForWorkspace(workspaceDir), "projects", encodeClaudeProjectPath(workspaceDir), sessionID+".jsonl")
}

func claudeHomeForWorkspace(workspaceDir string) string {
	clean := filepath.Clean(workspaceDir)
	parts := strings.Split(clean, string(filepath.Separator))
	if len(parts) >= 3 && parts[1] == "Users" {
		return filepath.Join(string(filepath.Separator), parts[1], parts[2], ".claude")
	}
	if len(parts) >= 3 && parts[1] == "home" {
		return filepath.Join(string(filepath.Separator), parts[1], parts[2], ".claude")
	}
	if home, err := os.UserHomeDir(); err == nil && home != "" {
		return filepath.Join(home, ".claude")
	}
	return filepath.Join(".claude")
}

func encodeClaudeProjectPath(path string) string {
	return strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			return r
		}
		return '-'
	}, path)
}

// CodexTranscriptPath finds Codex's rollout jsonl file for a thread id.
func CodexTranscriptPath(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	root := codexSessionRoot()
	if root == "" {
		return ""
	}
	for _, dir := range []string{root, filepath.Join(filepath.Dir(root), "archived_sessions")} {
		if path := findCodexTranscript(dir, sessionID); path != "" {
			return path
		}
	}
	return ""
}

func findCodexTranscript(root, sessionID string) string {
	var found string
	_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil || found != "" {
			return nil
		}
		if d.IsDir() || !strings.HasSuffix(d.Name(), sessionID+".jsonl") {
			return nil
		}
		found = path
		return nil
	})
	return found
}

func OpenCodeTranscriptPath(sessionID string) string {
	if sessionID == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dbPath := filepath.Join(home, ".local", "share", "opencode", "opencode.db")
	if !fileExists(dbPath) {
		return ""
	}
	outPath := filepath.Join(home, ".solo", "opencode-transcripts", sessionID+".jsonl")
	if err := exportOpenCodeTranscript(dbPath, outPath, sessionID); err != nil {
		return ""
	}
	return outPath
}

func OpenClawTranscriptPath(workspaceDir, sessionID string) string {
	if sessionID == "" {
		return ""
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	dir := filepath.Join(home, ".openclaw", "agents", "main", "sessions")
	if path := filepath.Join(dir, sessionID+".jsonl"); fileExists(path) {
		return path
	}
	if path := openClawPointerRuntimeFile(filepath.Join(dir, sessionID+".trajectory-path.json")); path != "" {
		return path
	}
	if path := filepath.Join(dir, sessionID+".trajectory.jsonl"); fileExists(path) {
		return path
	}
	if path := firstExistingGlob(filepath.Join(dir, sessionID+"*.jsonl")); path != "" {
		return path
	}
	if path := firstExistingGlob(filepath.Join(dir, sessionID+"*.json")); path != "" {
		return path
	}
	return newestOpenClawTranscriptForWorkspace(dir, workspaceDir)
}

func newestOpenClawTranscriptForWorkspace(dir, workspaceDir string) string {
	if path := newestOpenClawTranscriptForWorkspaceKind(dir, workspaceDir, false); path != "" {
		return path
	}
	return newestOpenClawTranscriptForWorkspaceKind(dir, workspaceDir, true)
}

func newestOpenClawTranscriptForWorkspaceKind(dir, workspaceDir string, includeTrajectory bool) string {
	if workspaceDir == "" {
		return ""
	}
	matches, _ := filepath.Glob(filepath.Join(dir, "*.jsonl"))
	var newest string
	var newestTime time.Time
	for _, path := range matches {
		isTrajectory := strings.Contains(filepath.Base(path), ".trajectory.")
		if isTrajectory != includeTrajectory {
			continue
		}
		info, err := os.Stat(path)
		if err != nil || info.ModTime().Before(newestTime) {
			continue
		}
		data, err := os.ReadFile(path)
		if err != nil || !strings.Contains(string(data), workspaceDir) {
			continue
		}
		newest = path
		newestTime = info.ModTime()
	}
	return newest
}

type openCodePartRow struct {
	Role        string `json:"role"`
	Data        string `json:"data"`
	TimeCreated int64  `json:"time_created"`
}

func exportOpenCodeTranscript(dbPath, outPath, sessionID string) error {
	if _, err := exec.LookPath("sqlite3"); err != nil {
		return err
	}
	query := `SELECT m.data ->> '$.role' AS role, p.data, p.time_created
FROM part p JOIN message m ON m.id = p.message_id
WHERE p.session_id = ` + sqliteQuote(sessionID) + ` ORDER BY p.time_created, p.id`
	raw, err := exec.Command("sqlite3", "-json", dbPath, query).Output()
	if err != nil {
		return err
	}
	var rows []openCodePartRow
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return err
		}
	}
	if len(rows) == 0 {
		return os.ErrNotExist
	}
	if err := os.MkdirAll(filepath.Dir(outPath), 0755); err != nil {
		return err
	}
	var b strings.Builder
	for _, row := range rows {
		line := openCodeTranscriptLine(row)
		if line == nil {
			continue
		}
		encoded, err := json.Marshal(line)
		if err != nil {
			return err
		}
		b.Write(encoded)
		b.WriteByte('\n')
	}
	return os.WriteFile(outPath, []byte(b.String()), 0644)
}

func openCodeTranscriptLine(row openCodePartRow) map[string]any {
	role := row.Role
	if role != "user" && role != "assistant" {
		role = "assistant"
	}
	var part struct {
		Type   string          `json:"type"`
		Text   string          `json:"text"`
		Tool   string          `json:"tool"`
		CallID string          `json:"callID"`
		State  json.RawMessage `json:"state"`
		Time   struct {
			Start int64 `json:"start"`
			End   int64 `json:"end"`
		} `json:"time"`
	}
	if err := json.Unmarshal([]byte(row.Data), &part); err != nil {
		return nil
	}
	content := strings.TrimSpace(part.Text)
	if content == "" && len(part.State) > 0 {
		content = string(part.State)
	}
	if content == "" && part.Tool != "" {
		content = part.Tool
	}
	if content == "" {
		return nil
	}
	line := map[string]any{
		"type": role,
		"message": map[string]any{
			"content": content,
		},
	}
	ts := row.TimeCreated
	if part.Time.Start > 0 {
		ts = part.Time.Start
	}
	if ts > 0 {
		line["timestamp"] = time.UnixMilli(ts).UTC().Format(time.RFC3339Nano)
	}
	return line
}

func sqliteQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func openClawPointerRuntimeFile(path string) string {
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	var pointer struct {
		RuntimeFile string `json:"runtimeFile"`
	}
	if err := json.Unmarshal(data, &pointer); err != nil || pointer.RuntimeFile == "" {
		return ""
	}
	if fileExists(pointer.RuntimeFile) {
		return pointer.RuntimeFile
	}
	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func firstExistingGlob(pattern string) string {
	matches, err := filepath.Glob(pattern)
	if err != nil || len(matches) == 0 {
		return ""
	}
	return matches[0]
}
