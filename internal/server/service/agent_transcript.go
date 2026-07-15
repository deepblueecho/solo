package service

import (
	"bufio"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type AgentTranscriptEntry struct {
	Seq       int              `json:"seq"`
	Timestamp string           `json:"timestamp,omitempty"`
	Role      string           `json:"role"`
	Type      string           `json:"type"`
	Text      string           `json:"text,omitempty"`
	ToolName  string           `json:"tool_name,omitempty"`
	ToolID    string           `json:"tool_id,omitempty"`
	Input     json.RawMessage  `json:"input,omitempty"`
	Usage     *TranscriptUsage `json:"usage,omitempty"`
	Raw       json.RawMessage  `json:"raw,omitempty"`
}

type TranscriptUsage struct {
	InputTokens              int64 `json:"input_tokens"`
	OutputTokens             int64 `json:"output_tokens"`
	CacheCreationInputTokens int64 `json:"cache_creation_input_tokens,omitempty"`
	CacheReadInputTokens     int64 `json:"cache_read_input_tokens,omitempty"`
}

type cachedAgentTranscript struct {
	size    int64
	modTime time.Time
	entries []AgentTranscriptEntry
}

var agentTranscriptCache = struct {
	sync.Mutex
	byPath map[string]cachedAgentTranscript
}{
	byPath: map[string]cachedAgentTranscript{},
}

func ReadAgentTranscript(path string, limit int) ([]AgentTranscriptEntry, error) {
	return readAgentTranscript(path, time.Time{}, time.Time{}, limit)
}

func ReadAgentTranscriptWindow(path string, start, end time.Time, limit int) ([]AgentTranscriptEntry, error) {
	return readAgentTranscript(path, start, end, limit)
}

func ReadHermesTranscript(sessionID string, limit int) ([]AgentTranscriptEntry, error) {
	return ReadHermesTranscriptWindow(sessionID, time.Time{}, time.Time{}, limit)
}

func ReadHermesTranscriptWindow(sessionID string, start, end time.Time, limit int) ([]AgentTranscriptEntry, error) {
	if sessionID == "" {
		return []AgentTranscriptEntry{}, nil
	}
	if limit <= 0 {
		limit = 1000
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return []AgentTranscriptEntry{}, nil
	}
	dbPath := filepath.Join(home, ".hermes", "state.db")
	if _, err := os.Stat(dbPath); err != nil {
		if os.IsNotExist(err) {
			return []AgentTranscriptEntry{}, nil
		}
		return nil, err
	}
	query := `SELECT role, content, tool_name, tool_calls, reasoning, reasoning_content, timestamp FROM messages WHERE session_id = ` + sqliteQuote(sessionID) + ` AND active = 1 ORDER BY id`
	raw, err := exec.Command("sqlite3", "-json", dbPath, query).Output()
	if err != nil {
		return nil, err
	}
	var rows []hermesTranscriptRow
	if len(strings.TrimSpace(string(raw))) > 0 {
		if err := json.Unmarshal(raw, &rows); err != nil {
			return nil, err
		}
	}
	entries := make([]AgentTranscriptEntry, 0, len(rows))
	for _, row := range rows {
		entries = append(entries, hermesTranscriptEntry(row))
	}
	return filterTranscriptEntries(entries, start, end, limit), nil
}

func readAgentTranscript(path string, start, end time.Time, limit int) ([]AgentTranscriptEntry, error) {
	if path == "" {
		return []AgentTranscriptEntry{}, nil
	}
	if limit <= 0 {
		limit = 1000
	}

	entries, err := cachedTranscriptEntries(path)
	if err != nil {
		return nil, err
	}
	return filterTranscriptEntries(entries, start, end, limit), nil
}

type hermesTranscriptRow struct {
	Role             string   `json:"role"`
	Content          *string  `json:"content"`
	ToolName         *string  `json:"tool_name"`
	ToolCalls        *string  `json:"tool_calls"`
	Reasoning        *string  `json:"reasoning"`
	ReasoningContent *string  `json:"reasoning_content"`
	Timestamp        *float64 `json:"timestamp"`
}

func hermesTranscriptEntry(row hermesTranscriptRow) AgentTranscriptEntry {
	role := row.Role
	if role != "user" && role != "assistant" && role != "tool" {
		role = "assistant"
	}
	entry := AgentTranscriptEntry{
		Role: role,
		Type: "text",
		Text: firstNonEmptyPtr(row.Content, row.ReasoningContent, row.Reasoning, row.ToolCalls, row.ToolName),
	}
	if row.ToolName != nil && *row.ToolName != "" {
		entry.Type = "tool_use"
		entry.ToolName = *row.ToolName
	}
	if row.Role == "tool" {
		entry.Type = "tool_result"
	}
	if row.Timestamp != nil && *row.Timestamp > 0 {
		sec := int64(*row.Timestamp)
		nsec := int64((*row.Timestamp - float64(sec)) * 1e9)
		entry.Timestamp = time.Unix(sec, nsec).UTC().Format(time.RFC3339Nano)
	}
	return entry
}

func firstNonEmptyPtr(values ...*string) string {
	for _, value := range values {
		if value != nil && *value != "" {
			return *value
		}
	}
	return ""
}

func sqliteQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "''") + "'"
}

func cachedTranscriptEntries(path string) ([]AgentTranscriptEntry, error) {
	stat, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []AgentTranscriptEntry{}, nil
		}
		return nil, err
	}

	agentTranscriptCache.Lock()
	cached, ok := agentTranscriptCache.byPath[path]
	if ok && cached.size == stat.Size() && cached.modTime.Equal(stat.ModTime()) {
		entries := append([]AgentTranscriptEntry(nil), cached.entries...)
		agentTranscriptCache.Unlock()
		return entries, nil
	}
	agentTranscriptCache.Unlock()

	entries, err := parseTranscriptFile(path)
	if err != nil {
		return nil, err
	}

	agentTranscriptCache.Lock()
	agentTranscriptCache.byPath[path] = cachedAgentTranscript{
		size:    stat.Size(),
		modTime: stat.ModTime(),
		entries: append([]AgentTranscriptEntry(nil), entries...),
	}
	agentTranscriptCache.Unlock()
	return entries, nil
}

func parseTranscriptFile(path string) ([]AgentTranscriptEntry, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return []AgentTranscriptEntry{}, nil
		}
		return nil, err
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	scanner.Buffer(make([]byte, 64*1024), 10*1024*1024)

	var entries []AgentTranscriptEntry
	for scanner.Scan() {
		line := append([]byte(nil), scanner.Bytes()...)
		for _, entry := range parseTranscriptLine(line) {
			entries = append(entries, entry)
		}
	}
	return entries, scanner.Err()
}

func filterTranscriptEntries(entries []AgentTranscriptEntry, start, end time.Time, limit int) []AgentTranscriptEntry {
	result := make([]AgentTranscriptEntry, 0, len(entries))
	seq := 0
	for _, entry := range entries {
		if !entryInWindow(entry, start, end) {
			continue
		}
		seq++
		entry.Seq = seq
		result = append(result, entry)
		if len(result) > limit {
			result = result[1:]
		}
	}
	if result == nil {
		return []AgentTranscriptEntry{}
	}
	return result
}

func entryInWindow(entry AgentTranscriptEntry, start, end time.Time) bool {
	if start.IsZero() && end.IsZero() {
		return true
	}
	if entry.Timestamp == "" {
		return false
	}
	ts, err := time.Parse(time.RFC3339, entry.Timestamp)
	if err != nil {
		return false
	}
	if !start.IsZero() && ts.Before(start) {
		return false
	}
	if !end.IsZero() && ts.After(end) {
		return false
	}
	return true
}

func parseTranscriptLine(line json.RawMessage) []AgentTranscriptEntry {
	var obj struct {
		Type      string          `json:"type"`
		TS        string          `json:"ts"`
		Timestamp string          `json:"timestamp"`
		Payload   json.RawMessage `json:"payload"`
		Data      json.RawMessage `json:"data"`
		Message   struct {
			Role    string          `json:"role"`
			Content json.RawMessage `json:"content"`
			Usage   json.RawMessage `json:"usage"`
		} `json:"message"`
		Attachment struct {
			Type   string `json:"type"`
			Prompt []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"prompt"`
		} `json:"attachment"`
	}
	if err := json.Unmarshal(line, &obj); err != nil {
		return nil
	}

	if entries := parseCodexTranscriptLine(obj.Type, obj.Timestamp, obj.Payload, line); len(entries) > 0 {
		return entries
	}

	if obj.Type == "attachment" && obj.Attachment.Type == "queued_command" {
		entries := make([]AgentTranscriptEntry, 0, len(obj.Attachment.Prompt))
		for _, prompt := range obj.Attachment.Prompt {
			if prompt.Text == "" {
				continue
			}
			entries = append(entries, AgentTranscriptEntry{
				Timestamp: obj.Timestamp,
				Role:      "user",
				Type:      "text",
				Text:      prompt.Text,
				Raw:       line,
			})
		}
		return entries
	}

	role := obj.Type
	if obj.Type == "message" {
		role = obj.Message.Role
	}
	if role != "user" && role != "assistant" {
		return nil
	}
	usage := transcriptUsageFromRaw(obj.Message.Usage)

	if text := rawJSONString(obj.Message.Content); text != "" {
		return attachTranscriptUsage([]AgentTranscriptEntry{{
			Timestamp: obj.Timestamp,
			Role:      role,
			Type:      "text",
			Text:      text,
			Raw:       line,
		}}, usage)
	}

	var blocks []json.RawMessage
	if err := json.Unmarshal(obj.Message.Content, &blocks); err != nil {
		return nil
	}

	entries := make([]AgentTranscriptEntry, 0, len(blocks))
	for _, blockRaw := range blocks {
		var block struct {
			Type      string          `json:"type"`
			Text      string          `json:"text"`
			Thinking  string          `json:"thinking"`
			Name      string          `json:"name"`
			ID        string          `json:"id"`
			ToolUseID string          `json:"tool_use_id"`
			Content   json.RawMessage `json:"content"`
			Input     json.RawMessage `json:"input"`
		}
		if err := json.Unmarshal(blockRaw, &block); err != nil {
			continue
		}
		entry := AgentTranscriptEntry{
			Timestamp: obj.Timestamp,
			Role:      role,
			Type:      block.Type,
			Raw:       blockRaw,
		}
		switch block.Type {
		case "text", "output_text":
			entry.Type = "text"
			entry.Text = block.Text
		case "thinking", "reasoning":
			entry.Type = "thinking"
			entry.Text = firstNonEmptyString(block.Thinking, block.Text)
		case "tool_use":
			entry.ToolName = block.Name
			entry.ToolID = block.ID
			entry.Input = block.Input
		case "tool_result":
			entry.ToolID = block.ToolUseID
			entry.Text = rawJSONString(block.Content)
		default:
			entry.Text = firstNonEmptyString(block.Text, rawJSONString(block.Content))
		}
		entries = append(entries, entry)
	}
	return attachTranscriptUsage(entries, usage)
}

func parseCodexTranscriptLine(kind, timestamp string, payload, raw json.RawMessage) []AgentTranscriptEntry {
	if kind != "response_item" && kind != "event_msg" {
		return nil
	}
	var p struct {
		Type      string          `json:"type"`
		Role      string          `json:"role"`
		Content   json.RawMessage `json:"content"`
		Name      string          `json:"name"`
		CallID    string          `json:"call_id"`
		Arguments string          `json:"arguments"`
		Input     string          `json:"input"`
		Output    string          `json:"output"`
		Message   string          `json:"message"`
		Text      string          `json:"text"`
		Summary   json.RawMessage `json:"summary"`
	}
	if len(payload) == 0 || json.Unmarshal(payload, &p) != nil {
		return nil
	}
	switch p.Type {
	case "message":
		if p.Role != "user" && p.Role != "assistant" {
			return nil
		}
		return contentEntries(timestamp, p.Role, p.Content, raw)
	case "reasoning":
		text := firstNonEmptyString(p.Text, codexReasoningSummaryText(p.Summary))
		if text == "" {
			return nil
		}
		return []AgentTranscriptEntry{{Timestamp: timestamp, Role: "assistant", Type: "thinking", Text: text, Raw: raw}}
	case "function_call", "custom_tool_call":
		return []AgentTranscriptEntry{{
			Timestamp: timestamp,
			Role:      "assistant",
			Type:      "tool_use",
			ToolName:  p.Name,
			ToolID:    p.CallID,
			Text:      firstNonEmptyString(p.Arguments, p.Input),
			Raw:       raw,
		}}
	case "function_call_output", "custom_tool_call_output":
		return []AgentTranscriptEntry{{Timestamp: timestamp, Role: "tool", Type: "tool_result", ToolID: p.CallID, Text: firstNonEmptyString(p.Output, p.Message), Raw: raw}}
	case "user_message":
		return []AgentTranscriptEntry{{Timestamp: timestamp, Role: "user", Type: "text", Text: p.Message, Raw: raw}}
	}
	return nil
}

func contentEntries(timestamp, role string, content, raw json.RawMessage) []AgentTranscriptEntry {
	if text := rawJSONString(content); text != "" {
		return []AgentTranscriptEntry{{Timestamp: timestamp, Role: role, Type: "text", Text: text, Raw: raw}}
	}
	var blocks []struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}
	if json.Unmarshal(content, &blocks) != nil {
		return nil
	}
	entries := make([]AgentTranscriptEntry, 0, len(blocks))
	for _, block := range blocks {
		text := block.Text
		if text == "" {
			continue
		}
		entries = append(entries, AgentTranscriptEntry{Timestamp: timestamp, Role: role, Type: "text", Text: text, Raw: raw})
	}
	return entries
}

func codexReasoningSummaryText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	if text := rawJSONString(raw); text != "" {
		return text
	}
	var blocks []struct {
		Text string `json:"text"`
	}
	if json.Unmarshal(raw, &blocks) != nil {
		return ""
	}
	parts := make([]string, 0, len(blocks))
	for _, block := range blocks {
		if block.Text != "" {
			parts = append(parts, block.Text)
		}
	}
	return strings.Join(parts, "\n")
}

func rawJSONString(raw json.RawMessage) string {
	var text string
	if len(raw) > 0 && json.Unmarshal(raw, &text) == nil {
		return text
	}
	return ""
}

func attachTranscriptUsage(entries []AgentTranscriptEntry, usage *TranscriptUsage) []AgentTranscriptEntry {
	if usage == nil || len(entries) == 0 {
		return entries
	}
	entries[0].Usage = usage
	return entries
}

func transcriptUsageFromRaw(raw json.RawMessage) *TranscriptUsage {
	if len(raw) == 0 {
		return nil
	}
	var data struct {
		InputTokens              int64 `json:"input_tokens"`
		OutputTokens             int64 `json:"output_tokens"`
		CacheCreationInputTokens int64 `json:"cache_creation_input_tokens"`
		CacheReadInputTokens     int64 `json:"cache_read_input_tokens"`
		CacheCreation            struct {
			Ephemeral1hInputTokens int64 `json:"ephemeral_1h_input_tokens"`
			Ephemeral5mInputTokens int64 `json:"ephemeral_5m_input_tokens"`
		} `json:"cache_creation"`
	}
	if err := json.Unmarshal(raw, &data); err != nil {
		return nil
	}
	usage := &TranscriptUsage{
		InputTokens:              data.InputTokens,
		OutputTokens:             data.OutputTokens,
		CacheCreationInputTokens: data.CacheCreationInputTokens + data.CacheCreation.Ephemeral1hInputTokens + data.CacheCreation.Ephemeral5mInputTokens,
		CacheReadInputTokens:     data.CacheReadInputTokens,
	}
	if usage.InputTokens == 0 && usage.OutputTokens == 0 && usage.CacheCreationInputTokens == 0 && usage.CacheReadInputTokens == 0 {
		return nil
	}
	return usage
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if value != "" {
			return value
		}
	}
	return ""
}
