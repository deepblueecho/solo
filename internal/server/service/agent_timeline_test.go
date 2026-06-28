package service

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestGetSessionTimelineReadsSessionTranscript(t *testing.T) {
	pool := agentRunTestPool(t)
	ctx := context.Background()
	ownerID := agentRunUser(t, pool)
	agentID := agentRunAgent(t, pool, ownerID)
	var runID string
	t.Cleanup(func() {
		if runID != "" {
			_, _ = pool.Exec(context.Background(), `DELETE FROM agent_runs WHERE id = $1`, runID)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM agent_sessions WHERE agent_id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agents WHERE id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	svc := NewAgentRunService(pool)
	session, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "session-timeline-1",
		TranscriptPath:    agentRunTranscriptFileWithText(t, "session detail"),
	})
	if err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	run, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    session.ID,
		TriggerType:  AgentRunTriggerMessage,
		Status:       AgentRunStatusCompleted,
		ActivityText: "已完成",
		Source:       "claude",
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID = run.ID

	timeline, err := svc.GetSessionTimeline(ctx, session.ID, 100)
	if err != nil {
		t.Fatalf("GetSessionTimeline: %v", err)
	}
	if timeline.Scope != "session" || timeline.Session == nil || timeline.Session.ID != session.ID {
		t.Fatalf("timeline metadata = %+v", timeline)
	}
	if len(timeline.Entries) != 1 || timeline.Entries[0].Text != "session detail" {
		t.Fatalf("timeline entries = %+v", timeline.Entries)
	}
	if len(timeline.Runs) != 1 || timeline.Runs[0].ID != run.ID {
		t.Fatalf("timeline runs = %+v", timeline.Runs)
	}
}

func TestGetSessionTimelineFallsBackToRunTranscriptPath(t *testing.T) {
	pool := agentRunTestPool(t)
	ctx := context.Background()
	ownerID := agentRunUser(t, pool)
	agentID := agentRunAgent(t, pool, ownerID)
	channelID := agentRunChannel(t, pool, ownerID)
	var runID string
	t.Cleanup(func() {
		if runID != "" {
			_, _ = pool.Exec(context.Background(), `DELETE FROM agent_runs WHERE id = $1`, runID)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM agent_sessions WHERE agent_id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM channels WHERE id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agents WHERE id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	svc := NewAgentRunService(pool)
	session, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "session-timeline-run-path",
	})
	if err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	run, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    session.ID,
		TriggerType:  AgentRunTriggerMessage,
		ChannelID:    channelID,
		Status:       AgentRunStatusCompleted,
		ActivityText: "completed",
		Source:       "claude",
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID = run.ID
	runPath := agentRunTranscriptFileWithText(t, "run path transcript")
	if _, err := svc.UpdateRunTranscript(ctx, UpdateRunTranscriptInput{RunID: run.ID, TranscriptPath: runPath}); err != nil {
		t.Fatalf("UpdateRunTranscript: %v", err)
	}

	timeline, err := svc.GetSessionTimeline(ctx, session.ID, 100)
	if err != nil {
		t.Fatalf("GetSessionTimeline: %v", err)
	}
	if timeline.Session.TranscriptPath != runPath {
		t.Fatalf("session transcript path = %q, want %q", timeline.Session.TranscriptPath, runPath)
	}
	if len(timeline.Entries) != 1 || timeline.Entries[0].Text != "run path transcript" {
		t.Fatalf("timeline entries = %+v", timeline.Entries)
	}
}

func TestGetSessionTimelineResolvesProviderPathWhenSessionPathMissing(t *testing.T) {
	pool := agentRunTestPool(t)
	ctx := context.Background()
	ownerID := agentRunUser(t, pool)
	agentID := agentRunAgent(t, pool, ownerID)
	channelID := agentRunChannel(t, pool, ownerID)
	var runID string
	t.Cleanup(func() {
		if runID != "" {
			_, _ = pool.Exec(context.Background(), `DELETE FROM agent_runs WHERE id = $1`, runID)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM agent_sessions WHERE agent_id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM channels WHERE id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agents WHERE id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	codexHome := filepath.Join(t.TempDir(), ".codex")
	externalID := "codex-session-timeline"
	path := filepath.Join(codexHome, "sessions", "2026", "06", "28", "rollout-2026-06-28T12-00-00-"+externalID+".jsonl")
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatal(err)
	}
	line := `{"timestamp":"` + time.Now().UTC().Format(time.RFC3339) + `","type":"response_item","payload":{"type":"message","role":"assistant","content":[{"type":"output_text","text":"timeline transcript"}]}}` + "\n"
	if err := os.WriteFile(path, []byte(line), 0644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CODEX_HOME", codexHome)

	svc := NewAgentRunService(pool)
	session, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "codex",
		ExternalSessionID: externalID,
	})
	if err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	run, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    session.ID,
		TriggerType:  AgentRunTriggerManual,
		ChannelID:    channelID,
		Status:       AgentRunStatusRunning,
		ActivityText: "running",
		Source:       "codex",
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID = run.ID

	timeline, err := svc.GetSessionTimeline(ctx, session.ID, 10)
	if err != nil {
		t.Fatalf("GetSessionTimeline: %v", err)
	}
	if len(timeline.Entries) != 1 || timeline.Entries[0].Text != "timeline transcript" {
		t.Fatalf("timeline entries = %+v", timeline.Entries)
	}
}

func TestGetTaskTimelineStitchesLinkedRunTranscripts(t *testing.T) {
	pool := agentRunTestPool(t)
	ctx := context.Background()
	ownerID := agentRunUser(t, pool)
	agentID := agentRunAgent(t, pool, ownerID)
	channelID := agentRunChannel(t, pool, ownerID)
	taskID := agentRunTask(t, pool, channelID, ownerID)
	var runIDs []string
	t.Cleanup(func() {
		if len(runIDs) > 0 {
			_, _ = pool.Exec(context.Background(), `DELETE FROM agent_runs WHERE id = ANY($1)`, runIDs)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM agent_sessions WHERE agent_id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tasks WHERE channel_id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM channels WHERE id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agents WHERE id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	svc := NewAgentRunService(pool)
	firstSession, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "task-timeline-1",
		TranscriptPath:    agentRunTranscriptFileWithText(t, "first task turn"),
	})
	if err != nil {
		t.Fatalf("UpsertSession first: %v", err)
	}
	firstRun, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    firstSession.ID,
		TriggerType:  AgentRunTriggerTask,
		ChannelID:    channelID,
		Status:       AgentRunStatusCompleted,
		ActivityText: "first",
		Source:       "claude",
	})
	if err != nil {
		t.Fatalf("StartRun first: %v", err)
	}
	runIDs = append(runIDs, firstRun.ID)
	if err := svc.LinkTask(ctx, LinkRunTaskInput{RunID: firstRun.ID, TaskID: taskID}); err != nil {
		t.Fatalf("LinkTask first: %v", err)
	}

	secondSession, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "task-timeline-2",
		TranscriptPath:    agentRunTranscriptFileWithText(t, "second task turn"),
	})
	if err != nil {
		t.Fatalf("UpsertSession second: %v", err)
	}
	secondRun, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    secondSession.ID,
		TriggerType:  AgentRunTriggerTask,
		ChannelID:    channelID,
		Status:       AgentRunStatusCompleted,
		ActivityText: "second",
		Source:       "claude",
	})
	if err != nil {
		t.Fatalf("StartRun second: %v", err)
	}
	runIDs = append(runIDs, secondRun.ID)
	if err := svc.LinkTask(ctx, LinkRunTaskInput{RunID: secondRun.ID, TaskID: taskID}); err != nil {
		t.Fatalf("LinkTask second: %v", err)
	}
	timeoutSession, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "task-timeline-timeout",
		TranscriptPath:    agentRunTranscriptFileWithText(t, "timeout task turn"),
	})
	if err != nil {
		t.Fatalf("UpsertSession timeout: %v", err)
	}
	timeoutRun, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    timeoutSession.ID,
		TriggerType:  AgentRunTriggerTask,
		ChannelID:    channelID,
		Status:       AgentRunStatusTimeout,
		ActivityText: "timeout",
		Source:       "claude",
	})
	if err != nil {
		t.Fatalf("StartRun timeout: %v", err)
	}
	runIDs = append(runIDs, timeoutRun.ID)
	if err := svc.LinkTask(ctx, LinkRunTaskInput{RunID: timeoutRun.ID, TaskID: taskID}); err != nil {
		t.Fatalf("LinkTask timeout: %v", err)
	}

	timeline, err := svc.GetTaskTimeline(ctx, taskID, agentID, 100)
	if err != nil {
		t.Fatalf("GetTaskTimeline: %v", err)
	}
	if timeline.Scope != "task" || timeline.Task == nil || timeline.Task.ID != taskID {
		t.Fatalf("timeline metadata = %+v", timeline)
	}
	if len(timeline.Entries) != 3 {
		t.Fatalf("timeline entry count = %d, want 3: %+v", len(timeline.Entries), timeline.Entries)
	}
	if !timelineTextSet(timeline.Entries, "first task turn", "second task turn", "timeout task turn") {
		t.Fatalf("timeline entries = %+v", timeline.Entries)
	}
	if len(timeline.Runs) != 3 {
		t.Fatalf("timeline runs = %+v", timeline.Runs)
	}
	if timeline.Runs[0].EntryStartSeq == 0 || timeline.Runs[1].EntryStartSeq == 0 {
		t.Fatalf("timeline run segments missing entry seq: %+v", timeline.Runs)
	}
}

func TestGetTaskTimelineStartsAtPromptBeforeTriggerMessage(t *testing.T) {
	pool := agentRunTestPool(t)
	ctx := context.Background()
	ownerID := agentRunUser(t, pool)
	agentID := agentRunAgent(t, pool, ownerID)
	channelID := agentRunChannel(t, pool, ownerID)
	taskID := agentRunTask(t, pool, channelID, ownerID)
	messageID := agentRunMessage(t, pool, channelID, ownerID)
	var runID string
	t.Cleanup(func() {
		if runID != "" {
			_, _ = pool.Exec(context.Background(), `DELETE FROM agent_runs WHERE id = $1`, runID)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM messages WHERE channel_id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agent_sessions WHERE agent_id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tasks WHERE channel_id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM channels WHERE id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agents WHERE id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	promptAt := time.Date(2026, 6, 26, 8, 0, 0, 0, time.UTC)
	triggerAt := promptAt.Add(time.Minute)
	runAt := triggerAt.Add(10 * time.Second)
	_, err := pool.Exec(ctx, `UPDATE messages SET created_at = $2, updated_at = $2 WHERE id = $1`, messageID, triggerAt)
	if err != nil {
		t.Fatalf("update message time: %v", err)
	}
	transcriptPath := agentRunTranscriptFileWithTimedText(t,
		agentTranscriptTextAt{At: promptAt, Text: "original user question"},
		agentTranscriptTextAt{At: runAt.Add(time.Second), Text: "assistant work"},
	)

	svc := NewAgentRunService(pool)
	session, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "task-timeline-trigger",
		TranscriptPath:    transcriptPath,
	})
	if err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	run, err := svc.StartRun(ctx, StartRunInput{
		AgentID:          agentID,
		SessionID:        session.ID,
		TriggerType:      AgentRunTriggerTask,
		TriggerMessageID: messageID,
		ChannelID:        channelID,
		Status:           AgentRunStatusCompleted,
		ActivityText:     "completed",
		Source:           "claude",
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID = run.ID
	_, err = pool.Exec(ctx, `UPDATE agent_runs SET started_at = $2, updated_at = $2, finished_at = $3 WHERE id = $1`, run.ID, runAt, runAt.Add(5*time.Second))
	if err != nil {
		t.Fatalf("update run time: %v", err)
	}
	if err := svc.LinkTask(ctx, LinkRunTaskInput{RunID: run.ID, TaskID: taskID}); err != nil {
		t.Fatalf("LinkTask: %v", err)
	}

	timeline, err := svc.GetTaskTimeline(ctx, taskID, agentID, 100)
	if err != nil {
		t.Fatalf("GetTaskTimeline: %v", err)
	}
	if len(timeline.Entries) != 2 {
		t.Fatalf("timeline entries = %+v", timeline.Entries)
	}
	if timeline.Entries[0].Text != "original user question" {
		t.Fatalf("first timeline entry = %+v", timeline.Entries[0])
	}
}

func TestGetTaskTimelinePrefersRunTranscriptPath(t *testing.T) {
	pool := agentRunTestPool(t)
	ctx := context.Background()
	ownerID := agentRunUser(t, pool)
	agentID := agentRunAgent(t, pool, ownerID)
	channelID := agentRunChannel(t, pool, ownerID)
	taskID := agentRunTask(t, pool, channelID, ownerID)
	var runID string
	t.Cleanup(func() {
		if runID != "" {
			_, _ = pool.Exec(context.Background(), `DELETE FROM agent_runs WHERE id = $1`, runID)
		}
		_, _ = pool.Exec(context.Background(), `DELETE FROM agent_sessions WHERE agent_id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM tasks WHERE channel_id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM channels WHERE id = $1`, channelID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM agents WHERE id = $1`, agentID)
		_, _ = pool.Exec(context.Background(), `DELETE FROM users WHERE id = $1`, ownerID)
	})

	sessionPath := agentRunTranscriptFileWithTimedText(t, agentTranscriptTextAt{At: time.Date(2026, 6, 26, 8, 0, 0, 0, time.UTC), Text: "new session text"})
	runPath := agentRunTranscriptFileWithTimedText(t, agentTranscriptTextAt{At: time.Date(2026, 6, 25, 8, 0, 0, 0, time.UTC), Text: "old run text"})
	svc := NewAgentRunService(pool)
	session, err := svc.UpsertSession(ctx, UpsertSessionInput{
		AgentID:           agentID,
		Provider:          "claude",
		ExternalSessionID: "shared-session",
		TranscriptPath:    sessionPath,
	})
	if err != nil {
		t.Fatalf("UpsertSession: %v", err)
	}
	run, err := svc.StartRun(ctx, StartRunInput{
		AgentID:      agentID,
		SessionID:    session.ID,
		TriggerType:  AgentRunTriggerTask,
		ChannelID:    channelID,
		Status:       AgentRunStatusCompleted,
		ActivityText: "completed",
		Source:       "claude",
	})
	if err != nil {
		t.Fatalf("StartRun: %v", err)
	}
	runID = run.ID
	if _, err := svc.UpdateRunTranscript(ctx, UpdateRunTranscriptInput{RunID: run.ID, TranscriptPath: runPath}); err != nil {
		t.Fatalf("UpdateRunTranscript: %v", err)
	}
	runAt := time.Date(2026, 6, 25, 8, 0, 1, 0, time.UTC)
	_, err = pool.Exec(ctx, `UPDATE agent_runs SET started_at = $2, updated_at = $2, finished_at = $3 WHERE id = $1`, run.ID, runAt, runAt.Add(time.Second))
	if err != nil {
		t.Fatalf("update run time: %v", err)
	}
	if err := svc.LinkTask(ctx, LinkRunTaskInput{RunID: run.ID, TaskID: taskID}); err != nil {
		t.Fatalf("LinkTask: %v", err)
	}

	timeline, err := svc.GetTaskTimeline(ctx, taskID, agentID, 100)
	if err != nil {
		t.Fatalf("GetTaskTimeline: %v", err)
	}
	if len(timeline.Entries) != 1 || timeline.Entries[0].Text != "old run text" {
		t.Fatalf("timeline entries = %+v", timeline.Entries)
	}
}

func TestTranscriptPromptWindowStartUsesPreviousUserPrompt(t *testing.T) {
	promptAt := time.Date(2026, 6, 26, 8, 0, 0, 0, time.UTC)
	anchor := promptAt.Add(time.Minute)
	got := transcriptPromptWindowStart([]AgentTranscriptEntry{
		{Timestamp: promptAt.Add(-time.Hour).Format(time.RFC3339), Role: "user", Type: "text", Text: "old"},
		{Timestamp: promptAt.Format(time.RFC3339), Role: "user", Type: "text", Text: "current"},
		{Timestamp: anchor.Add(time.Second).Format(time.RFC3339), Role: "user", Type: "text", Text: "next"},
	}, anchor)
	if !got.Equal(promptAt) {
		t.Fatalf("window start = %s, want %s", got, promptAt)
	}
}

func TestTranscriptWindowEntriesFallsBackToSiblingJSONL(t *testing.T) {
	dir := t.TempDir()
	promptAt := time.Date(2026, 6, 25, 9, 43, 53, 0, time.UTC)
	wrongPath := filepath.Join(dir, "wrong.jsonl")
	rightPath := filepath.Join(dir, "right.jsonl")
	if err := os.WriteFile(wrongPath, []byte(`{"type":"user","timestamp":"2026-06-26T09:43:53Z","message":{"content":"wrong day"}}`+"\n"), 0600); err != nil {
		t.Fatal(err)
	}
	raw := fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"content":"right task prompt"}}`+"\n", promptAt.Format(time.RFC3339))
	if err := os.WriteFile(rightPath, []byte(raw), 0600); err != nil {
		t.Fatal(err)
	}

	resolvedPath, entries, err := transcriptWindowEntries(wrongPath, promptAt.Add(time.Minute), promptAt.Add(5*time.Minute), 100)
	if err != nil {
		t.Fatalf("transcriptWindowEntries: %v", err)
	}
	if resolvedPath != rightPath {
		t.Fatalf("resolved path = %s, want %s", resolvedPath, rightPath)
	}
	if len(entries) != 1 || entries[0].Text != "right task prompt" {
		t.Fatalf("entries = %+v", entries)
	}
}

type agentTranscriptTextAt struct {
	At   time.Time
	Text string
}

func agentRunTranscriptFileWithTimedText(t *testing.T, entries ...agentTranscriptTextAt) string {
	t.Helper()
	path := t.TempDir() + "/session.jsonl"
	raw := ""
	for _, entry := range entries {
		raw += fmt.Sprintf(`{"type":"user","timestamp":%q,"message":{"content":%q}}`+"\n", entry.At.UTC().Format(time.RFC3339), entry.Text)
	}
	if err := os.WriteFile(path, []byte(raw), 0600); err != nil {
		t.Fatalf("write transcript: %v", err)
	}
	return path
}

func timelineTextSet(entries []AgentTranscriptEntry, wants ...string) bool {
	seen := map[string]bool{}
	for _, entry := range entries {
		seen[entry.Text] = true
	}
	for _, want := range wants {
		if !seen[want] {
			return false
		}
	}
	return true
}
