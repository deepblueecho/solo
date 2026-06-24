package service

import (
	"strings"
	"testing"
	"time"
)

func TestRenderArtifactHTML_EscapesMessageContent(t *testing.T) {
	data := artifactRenderData{
		Task: ArtifactTask{
			ID: "task-1", ChannelID: "channel-1", Number: 7,
			Title: "<script>alert(1)</script>", Status: TaskStatusInReview,
			CreatedAt: time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC),
		},
		RootMessage: ArtifactMessage{SenderName: "Alice", Content: `<img src=x onerror=alert(1)>`, CreatedAt: time.Date(2026, 6, 23, 10, 5, 0, 0, time.UTC)},
		GeneratedAt: time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC),
		Mode:        "latest",
	}

	html := renderArtifactHTML(data)
	if strings.Contains(html, "<script>") || strings.Contains(html, "<img src=x") {
		t.Fatalf("expected unsafe content to be escaped, got:\n%s", html)
	}
	for _, want := range []string{"&lt;script&gt;alert(1)&lt;/script&gt;", "&lt;img src=x onerror=alert(1)&gt;", "review before external sharing"} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered HTML to contain %q", want)
		}
	}
}

func TestArtifactFilenameForMode(t *testing.T) {
	if got := artifactFilename("latest"); got != "latest.html" {
		t.Fatalf("latest filename = %q", got)
	}
	if got := artifactFilename("final"); got != "final.html" {
		t.Fatalf("final filename = %q", got)
	}
}
