# Task Artifacts Design

## Goal

Create a lightweight Solo artifact feature that turns one task and its thread into a self-contained HTML snapshot. The first version is a static, local work canvas: useful during a task as `latest.html`, and frozen at review or completion as `final.html`.

## Assumptions

- A Solo task is already tied to a canonical message through `tasks.message_id`.
- The task thread is the source of work discussion and outcome evidence.
- The first version does not need live editing, collaboration, version history, or a React sandbox.
- `work-canvas-skill` is a template/reference source, not a runtime dependency.

## Recommended Approach

Copy and localize the useful parts of `work-canvas-skill`: single-file HTML structure, restrained CSS, optional vanilla JS modules, and the progress/review/comparison report patterns. Keep the generator native to Solo so it reads from Solo tables and respects Solo permissions.

The first artifact type is a review/status memo. It renders:

- Task title, number, status, priority, creator, claimer, and timestamps.
- The root task message.
- Thread timeline ordered by time.
- Attachment list with filenames, MIME type, size, and existing attachment URLs; image attachments render inline.
- A small "Needs input" section for `in_review` tasks or thread messages that explicitly ask the user for a decision.
- Provenance footer with task ID, channel ID, generator, and generated time.

## Backend

Add an `artifacts` table:

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

Routes:

- `POST /api/v1/tasks/{taskID}/artifact` regenerates `latest.html`.
- `POST /api/v1/tasks/{taskID}/artifact/finalize` writes `final.html`.
- `GET /api/v1/tasks/{taskID}/artifact/latest` returns metadata and URL.
- `GET /api/v1/artifacts/{artifactID}` serves the HTML file after membership checks.

The service should fetch the task, verify channel or DM membership, load the root message, load thread replies, collect attachment metadata, HTML-escape user content, render one template, write the file, and upsert artifact metadata.

## Storage

Store files under:

```text
.solo/artifacts/{taskID}/latest.html
.solo/artifacts/{taskID}/final.html
```

`latest.html` is overwritten in place so the link stays stable. `final.html` is overwritten only when the user explicitly finalizes again.

## Frontend

Add a small Artifact action in two places:

- Task card or task detail: `Generate Artifact` / `Refresh Artifact`.
- Thread panel: `Generate Artifact` near the task controls.

After generation, show the artifact inside Solo in an iframe-backed panel or dialog. `Open in new tab` can exist as a secondary action.

## Security

- Require existing channel or DM membership before generation or viewing.
- Escape all message content before rendering into HTML.
- Do not execute message content as raw HTML.
- Do not create a separate artifact attachment authorization layer in v1. Reuse existing attachment URLs for convenient in-product viewing.
- Render image attachments inline in the artifact HTML; render other attachments as links plus metadata.
- Include a provenance footer and "review before external sharing" note.

## Testing

Backend tests:

- Unauthorized user cannot generate or view an artifact HTML page.
- A task with no thread still generates a valid HTML file.
- Thread messages are ordered chronologically.
- Message content containing HTML/script is escaped.
- Refresh updates `latest.html` and artifact metadata.

Frontend smoke check:

- Task card or thread panel can trigger generation and show the returned artifact URL inside Solo.

## Explicitly Skipped

- AI summaries.
- Version history beyond `latest.html` and `final.html`.
- Realtime artifact updates.
- Editable/commentable artifacts.
- Sandboxed runnable React artifacts.
