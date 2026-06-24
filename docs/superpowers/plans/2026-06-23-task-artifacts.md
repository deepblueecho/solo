# Task Artifacts Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the static HTML artifact MVP for one Solo task and its thread.

**Architecture:** Add a small artifact persistence table, a backend `ArtifactService` that gathers task/thread/attachment data and writes self-contained HTML files, protected HTTP routes to generate/serve artifacts, and a small frontend action to trigger generation. No AI summary or live editor is included.

**Tech Stack:** Go 1.x, Chi, pgx/pgxpool, existing Solo REST conventions, Next.js/React, existing `apiClient`, no new dependencies.

## Global Constraints

- First version is a static, local work canvas.
- `latest.html` is overwritten in place so the link stays stable.
- `final.html` is written only by explicit finalize action.
- `work-canvas-skill` is a template/reference source, not a runtime dependency.
- Require existing channel or DM membership before generation or viewing.
- Escape all message content before rendering into HTML.
- Do not execute message content as raw HTML.
- Do not inline private attachment bytes in v1; show authorized attachment links and metadata only.
- Include a provenance footer and "review before external sharing" note.
- Skip AI summaries, version history, realtime updates, editable/commentable artifacts, and runnable React artifacts.

---

## File Structure

- Create `migrations/000028_create_artifacts.up.sql`: artifacts table and indexes.
- Create `migrations/000028_create_artifacts.down.sql`: drop artifacts table.
- Create `internal/server/service/artifact.go`: artifact data model, membership check, data gathering, deterministic HTML rendering, file writes, metadata upsert.
- Create `internal/server/service/artifact_test.go`: unit tests for HTML escaping and latest/final path selection.
- Create `internal/server/handler/artifact.go`: authenticated HTTP handlers for generate, finalize, latest metadata, and serve.
- Create `internal/server/handler/artifact_test.go`: handler validation tests for missing auth and invalid route input.
- Modify `internal/server/router.go`: wire `ArtifactService` and `ArtifactHandler`, add protected routes.
- Modify `frontend/lib/types.ts`: add artifact metadata type.
- Create `frontend/lib/hooks/use-task-artifact.ts`: tiny hook around artifact API calls.
- Modify `frontend/components/tasks/task-card.tsx`: add optional artifact action button.
- Modify `frontend/components/tasks/task-column.tsx`: pass the artifact action to `TaskCard`.
- Modify `frontend/components/tasks/task-board.tsx`: pass the artifact action to `TaskColumn`.
- Modify `frontend/components/dashboard/channel-view.tsx`: wire artifact generation for channel tasks and task threads.
- Modify `frontend/components/dashboard/dm-view.tsx`: wire artifact generation for DM tasks and task threads.
- Modify `frontend/components/dashboard/thread-panel.tsx`: add optional artifact action near task controls.
- Add one source-check script under `frontend/scripts/assert-task-artifact-entrypoints.mjs`.

---

### Task 1: Database Migration

**Files:**
- Create: `migrations/000028_create_artifacts.up.sql`
- Create: `migrations/000028_create_artifacts.down.sql`

**Interfaces:**
- Produces: table `artifacts` with columns `id`, `task_id`, `channel_id`, `kind`, `title`, `html_path`, `summary`, `source_snapshot`, `created_by`, `created_at`, `updated_at`.
- Consumes: existing `tasks(id)` and `channels(id)`.

- [ ] **Step 1: Create the up migration**

Add `migrations/000028_create_artifacts.up.sql`:

```sql
CREATE TABLE artifacts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    task_id UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    channel_id UUID NOT NULL REFERENCES channels(id) ON DELETE CASCADE,
    kind TEXT NOT NULL DEFAULT 'task_snapshot',
    title TEXT NOT NULL,
    html_path TEXT NOT NULL,
    summary TEXT,
    source_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_by UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX idx_artifacts_task_kind_path ON artifacts(task_id, kind, html_path);
CREATE INDEX idx_artifacts_task ON artifacts(task_id, updated_at DESC);
```

- [ ] **Step 2: Create the down migration**

Add `migrations/000028_create_artifacts.down.sql`:

```sql
DROP TABLE IF EXISTS artifacts;
```

- [ ] **Step 3: Run migration check**

Run:

```bash
go test ./cmd/migrate ./internal/db
```

Expected: `ok` for packages with tests, or `? ... [no test files]` for packages without tests.

- [ ] **Step 4: Commit**

```bash
git add migrations/000028_create_artifacts.up.sql migrations/000028_create_artifacts.down.sql
git commit -m "Add artifacts migration"
```

---

### Task 2: Artifact Service And Renderer

**Files:**
- Create: `internal/server/service/artifact.go`
- Create: `internal/server/service/artifact_test.go`

**Interfaces:**
- Consumes: `TaskService.GetTaskGlobal(ctx, taskID, userID) (*Task, error)`.
- Produces: `NewArtifactService(pool *pgxpool.Pool, rootDir string) *ArtifactService`.
- Produces: `func (s *ArtifactService) GenerateLatest(ctx context.Context, taskID, userID string) (*Artifact, error)`.
- Produces: `func (s *ArtifactService) Finalize(ctx context.Context, taskID, userID string) (*Artifact, error)`.
- Produces: `func (s *ArtifactService) Latest(ctx context.Context, taskID, userID string) (*Artifact, error)`.
- Produces: `func (s *ArtifactService) Get(ctx context.Context, artifactID, userID string) (*Artifact, error)`.

- [ ] **Step 1: Write unit tests for pure rendering helpers**

Add `internal/server/service/artifact_test.go`:

```go
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
		Mode: "latest",
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
```

- [ ] **Step 2: Run tests to verify they fail**

Run:

```bash
go test ./internal/server/service -run 'TestRenderArtifactHTML|TestArtifactFilenameForMode'
```

Expected: FAIL because `artifactRenderData`, `ArtifactTask`, `ArtifactMessage`, `renderArtifactHTML`, and `artifactFilename` are undefined.

- [ ] **Step 3: Add service types and renderer**

Create `internal/server/service/artifact.go` with these exported types and pure helpers first:

```go
package service

import (
	"html"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Artifact struct {
	ID             string    `json:"id"`
	TaskID         string    `json:"task_id"`
	ChannelID      string    `json:"channel_id"`
	Kind           string    `json:"kind"`
	Title          string    `json:"title"`
	HTMLPath       string    `json:"html_path"`
	URL            string    `json:"url"`
	Summary        string    `json:"summary,omitempty"`
	SourceSnapshot []byte    `json:"-"`
	CreatedBy      string    `json:"created_by"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type ArtifactService struct {
	pool    *pgxpool.Pool
	rootDir string
}

type ArtifactTask struct {
	ID, ChannelID, Title, Description, Status, Priority string
	Number                                             int
	CreatorName, ClaimerName                            string
	CreatedAt, UpdatedAt                                time.Time
}

type ArtifactMessage struct {
	ID, SenderType, SenderName, Content string
	CreatedAt                          time.Time
	Attachments                        []ArtifactAttachment
}

type ArtifactAttachment struct {
	ID, Filename, MIMEType, URL string
	Size                       int64
}

type artifactRenderData struct {
	Task        ArtifactTask
	RootMessage ArtifactMessage
	Thread      []ArtifactMessage
	GeneratedAt time.Time
	Mode        string
}

func NewArtifactService(pool *pgxpool.Pool, rootDir string) *ArtifactService {
	if rootDir == "" {
		rootDir = filepath.Join(".", ".solo", "artifacts")
	}
	return &ArtifactService{pool: pool, rootDir: rootDir}
}

func artifactFilename(mode string) string {
	if mode == "final" {
		return "final.html"
	}
	return "latest.html"
}

func renderArtifactHTML(data artifactRenderData) string {
	var b strings.Builder
	b.WriteString("<!doctype html><html><head><meta charset=\"utf-8\"><meta name=\"viewport\" content=\"width=device-width,initial-scale=1\">")
	b.WriteString("<title>")
	b.WriteString(html.EscapeString(data.Task.Title))
	b.WriteString("</title><style>")
	b.WriteString("body{font-family:ui-sans-serif,system-ui;margin:0;background:#f8fafc;color:#0f172a}main{max-width:960px;margin:0 auto;padding:32px}.card{background:white;border:2px solid #0f172a;border-radius:8px;padding:18px;margin:16px 0;box-shadow:6px 6px 0 #2563eb}.badge{display:inline-block;border:1px solid #0f172a;border-radius:999px;padding:2px 8px;font-size:12px;font-weight:700}pre{white-space:pre-wrap}footer{margin-top:32px;color:#64748b;font-size:12px}.msg{border-left:4px solid #2563eb;padding-left:12px;margin:14px 0}.meta{display:flex;gap:8px;flex-wrap:wrap}")
	b.WriteString("</style></head><body><main>")
	b.WriteString("<h1>")
	b.WriteString(html.EscapeString(data.Task.Title))
	b.WriteString("</h1><div class=\"meta\"><span class=\"badge\">#")
	b.WriteString(html.EscapeString(stringInt(data.Task.Number)))
	b.WriteString("</span><span class=\"badge\">")
	b.WriteString(html.EscapeString(data.Task.Status))
	b.WriteString("</span><span class=\"badge\">")
	b.WriteString(html.EscapeString(data.Mode))
	b.WriteString("</span></div>")
	if data.Task.Status == TaskStatusInReview {
		b.WriteString("<section class=\"card\"><h2>Needs input</h2><p>This task is in review. Review the artifact before accepting or rejecting the task.</p></section>")
	}
	b.WriteString("<section class=\"card\"><h2>Task</h2><p>")
	b.WriteString(html.EscapeString(data.Task.Description))
	b.WriteString("</p></section>")
	b.WriteString("<section class=\"card\"><h2>Root message</h2>")
	writeArtifactMessage(&b, data.RootMessage)
	b.WriteString("</section><section class=\"card\"><h2>Thread timeline</h2>")
	if len(data.Thread) == 0 {
		b.WriteString("<p>No thread replies yet.</p>")
	}
	for _, msg := range data.Thread {
		writeArtifactMessage(&b, msg)
	}
	b.WriteString("</section><footer>Generated by Solo artifact renderer at ")
	b.WriteString(html.EscapeString(data.GeneratedAt.Format(time.RFC3339)))
	b.WriteString(". Review before external sharing.</footer></main></body></html>")
	return b.String()
}

func writeArtifactMessage(b *strings.Builder, msg ArtifactMessage) {
	b.WriteString("<div class=\"msg\"><strong>")
	b.WriteString(html.EscapeString(msg.SenderName))
	b.WriteString("</strong> <small>")
	b.WriteString(html.EscapeString(msg.CreatedAt.Format(time.RFC3339)))
	b.WriteString("</small><pre>")
	b.WriteString(html.EscapeString(msg.Content))
	b.WriteString("</pre>")
	if len(msg.Attachments) > 0 {
		b.WriteString("<ul>")
		for _, a := range msg.Attachments {
			b.WriteString("<li><a href=\"")
			b.WriteString(html.EscapeString(a.URL))
			b.WriteString("\">")
			b.WriteString(html.EscapeString(a.Filename))
			b.WriteString("</a> ")
			b.WriteString(html.EscapeString(a.MIMEType))
			b.WriteString("</li>")
		}
		b.WriteString("</ul>")
	}
	b.WriteString("</div>")
}

func stringInt(n int) string {
	return strconv.Itoa(n)
}
```

- [ ] **Step 4: Run pure helper tests**

Run:

```bash
gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go
go test ./internal/server/service -run 'TestRenderArtifactHTML|TestArtifactFilenameForMode'
```

Expected: PASS.

- [ ] **Step 5: Add database-backed methods**

Extend `internal/server/service/artifact.go` with:

```go
func (s *ArtifactService) GenerateLatest(ctx context.Context, taskID, userID string) (*Artifact, error) {
	return s.generate(ctx, taskID, userID, "latest")
}

func (s *ArtifactService) Finalize(ctx context.Context, taskID, userID string) (*Artifact, error) {
	return s.generate(ctx, taskID, userID, "final")
}

func (s *ArtifactService) Latest(ctx context.Context, taskID, userID string) (*Artifact, error) {
	task, err := NewTaskService(s.pool).GetTaskGlobal(ctx, taskID, userID)
	if err != nil {
		return nil, err
	}
	return s.getByTaskPath(ctx, task.ID, userID, artifactFilename("latest"))
}

func (s *ArtifactService) Get(ctx context.Context, artifactID, userID string) (*Artifact, error) {
	var a Artifact
	err := s.pool.QueryRow(ctx, `SELECT id, task_id, channel_id, kind, title, html_path, COALESCE(summary, ''), source_snapshot, created_by, created_at, updated_at FROM artifacts WHERE id = $1`, artifactID).
		Scan(&a.ID, &a.TaskID, &a.ChannelID, &a.Kind, &a.Title, &a.HTMLPath, &a.Summary, &a.SourceSnapshot, &a.CreatedBy, &a.CreatedAt, &a.UpdatedAt)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrTaskNotFound
		}
		return nil, err
	}
	if _, err := NewTaskService(s.pool).GetTask(ctx, a.ChannelID, a.TaskID, userID); err != nil {
		return nil, err
	}
	a.URL = "/api/v1/artifacts/" + a.ID
	return &a, nil
}
```

Implement private helpers:

```go
func (s *ArtifactService) generate(ctx context.Context, taskID, userID, mode string) (*Artifact, error)
func (s *ArtifactService) loadRenderData(ctx context.Context, task *Task) (artifactRenderData, error)
func (s *ArtifactService) artifactPath(taskID, mode string) string
func (s *ArtifactService) upsertArtifact(ctx context.Context, task *Task, userID, mode, path string, snapshot []byte) (*Artifact, error)
func (s *ArtifactService) getByTaskPath(ctx context.Context, taskID, userID, filename string) (*Artifact, error)
```

Use these queries:

```sql
SELECT m.id, m.sender_type, COALESCE(u.display_name, ag.name, ''), m.content, m.created_at, COALESCE(m.attachment_ids, '{}')
FROM messages m
LEFT JOIN users u ON m.sender_type = 'user' AND m.sender_id = u.id
LEFT JOIN agents ag ON m.sender_type = 'agent' AND m.sender_id = ag.id
WHERE m.id = $1
```

```sql
SELECT m.id, m.sender_type, COALESCE(u.display_name, ag.name, ''), m.content, m.created_at, COALESCE(m.attachment_ids, '{}')
FROM threads t
JOIN messages m ON m.thread_id = t.id
LEFT JOIN users u ON m.sender_type = 'user' AND m.sender_id = u.id
LEFT JOIN agents ag ON m.sender_type = 'agent' AND m.sender_id = ag.id
WHERE t.root_message_id = $1 AND t.channel_id = $2
ORDER BY m.created_at ASC, m.id ASC
```

For attachments, query by collected IDs:

```sql
SELECT id, filename, mime_type, size FROM attachments WHERE id = ANY($1::uuid[])
```

Build URLs as `/api/v1/attachments/{id}`. Write HTML with `os.MkdirAll(filepath.Dir(path), 0o755)` and `os.WriteFile(path, []byte(html), 0o644)`. Store `source_snapshot` as compact JSON with task ID, message ID, thread message IDs, attachment IDs, and mode.

- [ ] **Step 6: Run service package tests**

Run:

```bash
gofmt -w internal/server/service/artifact.go internal/server/service/artifact_test.go
go test ./internal/server/service
```

Expected: PASS.

- [ ] **Step 7: Commit**

```bash
git add internal/server/service/artifact.go internal/server/service/artifact_test.go
git commit -m "Add task artifact renderer"
```

---

### Task 3: Artifact HTTP Routes

**Files:**
- Create: `internal/server/handler/artifact.go`
- Create: `internal/server/handler/artifact_test.go`
- Modify: `internal/server/router.go`

**Interfaces:**
- Consumes: `ArtifactService.GenerateLatest`, `Finalize`, `Latest`, and `Get`.
- Produces route: `POST /api/v1/tasks/{taskID}/artifact`.
- Produces route: `POST /api/v1/tasks/{taskID}/artifact/finalize`.
- Produces route: `GET /api/v1/tasks/{taskID}/artifact/latest`.
- Produces route: `GET /api/v1/artifacts/{artifactID}`.

- [ ] **Step 1: Write handler validation tests**

Add `internal/server/handler/artifact_test.go`:

```go
package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/solo-ai/solo/internal/server/service"
)

func setupArtifactRouter(h *ArtifactHandler) chi.Router {
	r := chi.NewRouter()
	r.Post("/api/v1/tasks/{taskID}/artifact", h.GenerateLatest)
	r.Post("/api/v1/tasks/{taskID}/artifact/finalize", h.Finalize)
	r.Get("/api/v1/tasks/{taskID}/artifact/latest", h.Latest)
	r.Get("/api/v1/artifacts/{artifactID}", h.Serve)
	return r
}

func TestArtifactHandler_MissingAuth(t *testing.T) {
	h := NewArtifactHandler(service.NewArtifactService(nil, ""))
	r := setupArtifactRouter(h)

	for _, tc := range []struct {
		method string
		path   string
	}{
		{http.MethodPost, "/api/v1/tasks/task-1/artifact"},
		{http.MethodPost, "/api/v1/tasks/task-1/artifact/finalize"},
		{http.MethodGet, "/api/v1/tasks/task-1/artifact/latest"},
		{http.MethodGet, "/api/v1/artifacts/artifact-1"},
	} {
		req := httptest.NewRequest(tc.method, tc.path, nil)
		rr := httptest.NewRecorder()
		r.ServeHTTP(rr, req)
		if rr.Code != http.StatusUnauthorized {
			t.Fatalf("%s %s: expected 401, got %d", tc.method, tc.path, rr.Code)
		}
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run:

```bash
go test ./internal/server/handler -run TestArtifactHandler_MissingAuth
```

Expected: FAIL because `ArtifactHandler` and `NewArtifactHandler` are undefined.

- [ ] **Step 3: Add artifact handler**

Create `internal/server/handler/artifact.go`:

```go
package handler

import (
	"errors"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/solo-ai/solo/internal/server/service"
)

type ArtifactHandler struct {
	svc *service.ArtifactService
}

func NewArtifactHandler(svc *service.ArtifactService) *ArtifactHandler {
	return &ArtifactHandler{svc: svc}
}

func (h *ArtifactHandler) GenerateLatest(w http.ResponseWriter, r *http.Request) {
	h.generate(w, r, false)
}

func (h *ArtifactHandler) Finalize(w http.ResponseWriter, r *http.Request) {
	h.generate(w, r, true)
}

func (h *ArtifactHandler) Latest(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}
	artifact, err := h.svc.Latest(r.Context(), taskID, userID)
	if err != nil {
		writeArtifactError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, artifact)
}

func (h *ArtifactHandler) Serve(w http.ResponseWriter, r *http.Request) {
	userID, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	artifactID := chi.URLParam(r, "artifactID")
	if artifactID == "" {
		writeError(w, http.StatusBadRequest, "artifact ID is required")
		return
	}
	artifact, err := h.svc.Get(r.Context(), artifactID, userID)
	if err != nil {
		writeArtifactError(w, err)
		return
	}
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	http.ServeFile(w, r, artifact.HTMLPath)
}

func (h *ArtifactHandler) generate(w http.ResponseWriter, r *http.Request, final bool) {
	userID, ok := requireUserID(r)
	if !ok {
		writeError(w, http.StatusUnauthorized, "not authenticated")
		return
	}
	taskID := chi.URLParam(r, "taskID")
	if taskID == "" {
		writeError(w, http.StatusBadRequest, "task ID is required")
		return
	}
	var artifact *service.Artifact
	var err error
	if final {
		artifact, err = h.svc.Finalize(r.Context(), taskID, userID)
	} else {
		artifact, err = h.svc.GenerateLatest(r.Context(), taskID, userID)
	}
	if err != nil {
		writeArtifactError(w, err)
		return
	}
	writeJSON(w, http.StatusCreated, artifact)
}

func writeArtifactError(w http.ResponseWriter, err error) {
	switch {
	case errors.Is(err, service.ErrTaskNotFound):
		writeError(w, http.StatusNotFound, "artifact not found")
	case errors.Is(err, service.ErrTaskNotChannelMember):
		writeError(w, http.StatusForbidden, "not a channel member")
	default:
		writeError(w, http.StatusInternalServerError, "failed to handle artifact")
	}
}
```

- [ ] **Step 4: Wire routes**

Modify `internal/server/router.go`:

```go
artifactRoot := os.Getenv("ARTIFACTS_DIR")
if artifactRoot == "" {
	if home, err := os.UserHomeDir(); err == nil {
		artifactRoot = filepath.Join(home, ".solo", "artifacts")
	} else {
		artifactRoot = filepath.Join(".", ".solo", "artifacts")
	}
}
artifactSvc := service.NewArtifactService(pool, artifactRoot)
artifactHandler := handler.NewArtifactHandler(artifactSvc)
```

Add protected task routes near global task routes:

```go
r.Post("/api/v1/tasks/{taskID}/artifact", artifactHandler.GenerateLatest)
r.Post("/api/v1/tasks/{taskID}/artifact/finalize", artifactHandler.Finalize)
r.Get("/api/v1/tasks/{taskID}/artifact/latest", artifactHandler.Latest)
r.Get("/api/v1/artifacts/{artifactID}", artifactHandler.Serve)
```

- [ ] **Step 5: Run handler and full Go tests**

Run:

```bash
gofmt -w internal/server/handler/artifact.go internal/server/handler/artifact_test.go internal/server/router.go
go test ./internal/server/handler ./internal/server/service
```

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/server/handler/artifact.go internal/server/handler/artifact_test.go internal/server/router.go
git commit -m "Add artifact HTTP routes"
```

---

### Task 4: Frontend Artifact Actions

**Files:**
- Modify: `frontend/lib/types.ts`
- Create: `frontend/lib/hooks/use-task-artifact.ts`
- Modify: `frontend/components/tasks/task-card.tsx`
- Modify: `frontend/components/tasks/task-column.tsx`
- Modify: `frontend/components/tasks/task-board.tsx`
- Modify: `frontend/components/dashboard/thread-panel.tsx`
- Modify: `frontend/components/dashboard/channel-view.tsx`
- Modify: `frontend/components/dashboard/dm-view.tsx`
- Create: `frontend/scripts/assert-task-artifact-entrypoints.mjs`

**Interfaces:**
- Consumes: backend `POST /api/v1/tasks/{taskID}/artifact`.
- Produces: `useTaskArtifact()` with `generateArtifact(taskID: string): Promise<TaskArtifact>`.
- Produces: optional `onGenerateArtifact?: (task: Task) => void` props down to `TaskCard`.
- Produces: optional `onGenerateArtifact?: () => void` on `ThreadPanel`.

- [ ] **Step 1: Add frontend type and hook**

Append to `frontend/lib/types.ts`:

```ts
export interface TaskArtifact {
  id: string;
  task_id: string;
  channel_id: string;
  kind: string;
  title: string;
  html_path: string;
  url: string;
  summary?: string;
  created_by: string;
  created_at: string;
  updated_at: string;
}
```

Create `frontend/lib/hooks/use-task-artifact.ts`:

```ts
'use client';

import { useCallback, useState } from 'react';
import { apiClient } from '@/lib/api-client';
import type { TaskArtifact } from '@/lib/types';

export function useTaskArtifact() {
  const [isGenerating, setIsGenerating] = useState(false);

  const generateArtifact = useCallback(async (taskId: string): Promise<TaskArtifact> => {
    setIsGenerating(true);
    try {
      return await apiClient.post<TaskArtifact>(`/api/v1/tasks/${taskId}/artifact`);
    } finally {
      setIsGenerating(false);
    }
  }, []);

  return { generateArtifact, isGenerating };
}
```

- [ ] **Step 2: Add source-check script before UI edits**

Create `frontend/scripts/assert-task-artifact-entrypoints.mjs`:

```js
import { readFileSync } from 'node:fs';

const read = (path) => readFileSync(new URL(`../${path}`, import.meta.url), 'utf8');
const assert = (condition, message) => {
  if (!condition) throw new Error(message);
};

const types = read('lib/types.ts');
const hook = read('lib/hooks/use-task-artifact.ts');
const taskCard = read('components/tasks/task-card.tsx');
const taskBoard = read('components/tasks/task-board.tsx');
const taskColumn = read('components/tasks/task-column.tsx');
const threadPanel = read('components/dashboard/thread-panel.tsx');
const channelView = read('components/dashboard/channel-view.tsx');
const dmView = read('components/dashboard/dm-view.tsx');

assert(types.includes('export interface TaskArtifact'), 'TaskArtifact type should exist');
assert(hook.includes('generateArtifact') && hook.includes('/api/v1/tasks/${taskId}/artifact'), 'useTaskArtifact should call the generate endpoint');
assert(taskCard.includes('onGenerateArtifact?: (task: Task) => void') && taskCard.includes('FileText'), 'TaskCard should expose an artifact action');
assert(taskBoard.includes('onGenerateArtifact?: (task: Task) => void'), 'TaskBoard should accept artifact action');
assert(taskColumn.includes('onGenerateArtifact?: (task: Task) => void'), 'TaskColumn should pass artifact action');
assert(threadPanel.includes('onGenerateArtifact?: () => void') && threadPanel.includes('Generate Artifact'), 'ThreadPanel should expose artifact generation');
assert(channelView.includes('useTaskArtifact') && channelView.includes('handleGenerateArtifact'), 'Channel view should wire artifact generation');
assert(dmView.includes('useTaskArtifact') && dmView.includes('handleGenerateArtifact'), 'DM view should wire artifact generation');

console.log('task artifact entrypoint source checks passed');
```

- [ ] **Step 3: Run source check to verify it fails**

Run:

```bash
node frontend/scripts/assert-task-artifact-entrypoints.mjs
```

Expected: FAIL because UI entrypoints are not wired yet.

- [ ] **Step 4: Add TaskCard action**

Modify `frontend/components/tasks/task-card.tsx`:

```ts
import { Calendar, User, ChevronRight, FileText } from 'lucide-react';
```

Add prop:

```ts
onGenerateArtifact?: (task: Task) => void;
```

Update function signature:

```ts
export function TaskCard({ task, onClick, showChannel = true, parentTaskNumber, onParentClick, onGenerateArtifact }: TaskCardProps) {
```

Add a compact button in the bottom row:

```tsx
{onGenerateArtifact && (
  <button
    type="button"
    onClick={(e) => {
      e.stopPropagation();
      onGenerateArtifact(task);
    }}
    className="inline-flex items-center gap-1 border-2 border-black bg-white px-2 py-1 font-mono text-[10px] font-bold uppercase shadow-brutal-sm hover:bg-brutal-info hover:text-black"
    aria-label={`Generate artifact for ${task.title}`}
  >
    <FileText className="h-3 w-3" />
    Artifact
  </button>
)}
```

- [ ] **Step 5: Thread action through task board components**

In `frontend/components/tasks/task-column.tsx` and `frontend/components/tasks/task-board.tsx`, add `onGenerateArtifact?: (task: Task) => void` to props and pass it down to `TaskCard`.

The `TaskCard` call should include:

```tsx
onGenerateArtifact={onGenerateArtifact}
```

- [ ] **Step 6: Add ThreadPanel action**

Modify `frontend/components/dashboard/thread-panel.tsx` props:

```ts
onGenerateArtifact?: () => void;
```

Import `FileText` from `lucide-react`. Add a button near the task controls:

```tsx
{onGenerateArtifact && (
  <button
    type="button"
    onClick={onGenerateArtifact}
    className="inline-flex items-center gap-1 border-2 border-black bg-white px-2 py-1 font-mono text-[10px] font-bold uppercase shadow-brutal-sm hover:bg-brutal-info"
  >
    <FileText className="h-3 w-3" />
    Generate Artifact
  </button>
)}
```

- [ ] **Step 7: Wire channel and DM views**

In `frontend/components/dashboard/channel-view.tsx` and `frontend/components/dashboard/dm-view.tsx`, import `useTaskArtifact`. Add:

```ts
const { generateArtifact } = useTaskArtifact();

const handleGenerateArtifact = useCallback(async (task: Task) => {
  const artifact = await generateArtifact(task.id);
  window.open(artifact.url, '_blank', 'noopener,noreferrer');
}, [generateArtifact]);
```

Pass `onGenerateArtifact={handleGenerateArtifact}` to `TaskBoard`. For `ThreadPanel`, pass:

```tsx
onGenerateArtifact={task ? () => handleGenerateArtifact(task) : undefined}
```

- [ ] **Step 8: Run frontend checks**

Run:

```bash
node frontend/scripts/assert-task-artifact-entrypoints.mjs
npm --prefix frontend run build
```

Expected: source check prints `task artifact entrypoint source checks passed`; build succeeds.

- [ ] **Step 9: Commit**

```bash
git add frontend/lib/types.ts frontend/lib/hooks/use-task-artifact.ts frontend/components/tasks/task-card.tsx frontend/components/tasks/task-column.tsx frontend/components/tasks/task-board.tsx frontend/components/dashboard/thread-panel.tsx frontend/components/dashboard/channel-view.tsx frontend/components/dashboard/dm-view.tsx frontend/scripts/assert-task-artifact-entrypoints.mjs
git commit -m "Add task artifact frontend actions"
```

---

## Final Verification

- [ ] Run all Go tests:

```bash
go test ./...
```

Expected: PASS.

- [ ] Run frontend build:

```bash
npm --prefix frontend run build
```

Expected: build succeeds.

- [ ] Manual smoke:

```bash
bash scripts/start-services.sh
```

Open the app, create or choose a task, add one thread reply, click `Artifact`, and confirm a new tab opens HTML containing the task title, root message, thread reply, provenance footer, and no raw HTML execution from message content.
