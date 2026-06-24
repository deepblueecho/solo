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

func TestRenderArtifactHTML_RendersImageAttachmentsInline(t *testing.T) {
	attachments := attachmentPlaceholders([]string{"550e8400-e29b-41d4-a716-446655440000"})
	attachments[0].Filename = `diagram <draft>.png`
	attachments[0].MIMEType = `image/png`
	attachments[0].Size = 4321

	data := artifactRenderData{
		Task: ArtifactTask{
			ID: "task-1", ChannelID: "channel-1", Number: 7,
			Title: "Attachment task", Status: TaskStatusTodo,
			CreatedAt: time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC),
		},
		RootMessage: ArtifactMessage{SenderName: "Alice", Content: "see attached", CreatedAt: time.Date(2026, 6, 23, 10, 5, 0, 0, time.UTC), Attachments: attachments},
		GeneratedAt: time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC),
		Mode:        "latest",
	}

	html := renderArtifactHTML(data)
	for _, want := range []string{
		`<img loading="lazy" src="/api/v1/attachments/550e8400-e29b-41d4-a716-446655440000" alt="diagram &lt;draft&gt;.png">`,
		"diagram &lt;draft&gt;.png",
		"image/png",
		"4321 bytes",
	} {
		if !strings.Contains(html, want) {
			t.Fatalf("expected rendered HTML to contain %q, got:\n%s", want, html)
		}
	}
}

func TestRenderArtifactHTML_RendersNonImageAttachmentsAsLinks(t *testing.T) {
	attachments := attachmentPlaceholders([]string{"550e8400-e29b-41d4-a716-446655440000"})
	attachments[0].Filename = `report <final>.pdf`
	attachments[0].MIMEType = `application/pdf`
	attachments[0].Size = 1234

	data := artifactRenderData{
		Task: ArtifactTask{
			ID: "task-1", ChannelID: "channel-1", Number: 7,
			Title: "Attachment task", Status: TaskStatusTodo,
			CreatedAt: time.Date(2026, 6, 23, 10, 0, 0, 0, time.UTC),
			UpdatedAt: time.Date(2026, 6, 23, 11, 0, 0, 0, time.UTC),
		},
		RootMessage: ArtifactMessage{SenderName: "Alice", Content: "see attached", CreatedAt: time.Date(2026, 6, 23, 10, 5, 0, 0, time.UTC), Attachments: attachments},
		GeneratedAt: time.Date(2026, 6, 23, 12, 0, 0, 0, time.UTC),
		Mode:        "latest",
	}

	html := renderArtifactHTML(data)
	for _, want := range []string{
		`<a href="/api/v1/attachments/550e8400-e29b-41d4-a716-446655440000">report &lt;final&gt;.pdf</a>`,
		"application/pdf",
		"1234 bytes",
	} {
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
