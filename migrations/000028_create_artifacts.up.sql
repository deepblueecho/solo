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
