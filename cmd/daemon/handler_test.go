package main

import "testing"

func TestRefreshTranscriptPathForProvider(t *testing.T) {
	existing := "/tmp/existing.jsonl"
	if got := refreshTranscriptPathForProvider("claude", "/tmp/workspace", "session-1", existing); got != existing {
		t.Fatalf("existing transcript path = %q, want %q", got, existing)
	}

	got := refreshTranscriptPathForProvider("claude", "/Users/me/.solo/agents/a1/workspace", "session-1", "")
	want := "/Users/me/.claude/projects/-Users-me--solo-agents-a1-workspace/session-1.jsonl"
	if got != want {
		t.Fatalf("refreshed transcript path = %q, want %q", got, want)
	}

	if got := refreshTranscriptPathForProvider("claude", "/tmp/workspace", "", ""); got != "" {
		t.Fatalf("empty session transcript path = %q, want empty", got)
	}
}
