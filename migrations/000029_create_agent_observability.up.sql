CREATE TABLE IF NOT EXISTS agent_sessions (
    id                  UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id            UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    provider            TEXT NOT NULL,
    external_session_id TEXT,
    transcript_path     TEXT,
    title               TEXT,
    status              TEXT NOT NULL DEFAULT 'active',
    started_at          TIMESTAMPTZ NOT NULL DEFAULT now(),
    last_active_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_agent_sessions_external
    ON agent_sessions(agent_id, provider, external_session_id)
    WHERE external_session_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_agent_sessions_agent_last_active
    ON agent_sessions(agent_id, last_active_at DESC);

CREATE TABLE IF NOT EXISTS agent_runs (
    id                 UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    agent_id           UUID NOT NULL REFERENCES agents(id) ON DELETE CASCADE,
    session_id         UUID REFERENCES agent_sessions(id) ON DELETE SET NULL,
    trigger_type       TEXT NOT NULL,
    trigger_message_id UUID REFERENCES messages(id) ON DELETE SET NULL,
    channel_id         UUID REFERENCES channels(id) ON DELETE SET NULL,
    thread_id          UUID REFERENCES threads(id) ON DELETE SET NULL,
    status             TEXT NOT NULL,
    activity_text      TEXT NOT NULL DEFAULT '',
    tool_name          TEXT,
    tool_input_summary TEXT,
    source             TEXT,
    usage_json         JSONB NOT NULL DEFAULT '{}'::jsonb,
    started_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT now(),
    finished_at        TIMESTAMPTZ
);

CREATE INDEX IF NOT EXISTS idx_agent_runs_agent_updated
    ON agent_runs(agent_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_runs_status_updated
    ON agent_runs(status, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_agent_runs_session
    ON agent_runs(session_id, started_at DESC);

CREATE TABLE IF NOT EXISTS agent_run_task_links (
    run_id     UUID NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    task_id    UUID NOT NULL REFERENCES tasks(id) ON DELETE CASCADE,
    role       TEXT NOT NULL DEFAULT 'primary',
    confidence DOUBLE PRECISION NOT NULL DEFAULT 1.0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (run_id, task_id)
);

CREATE INDEX IF NOT EXISTS idx_agent_run_task_links_task
    ON agent_run_task_links(task_id);

CREATE TABLE IF NOT EXISTS agent_run_events (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    run_id     UUID NOT NULL REFERENCES agent_runs(id) ON DELETE CASCADE,
    seq        INTEGER NOT NULL,
    type       TEXT NOT NULL,
    message    TEXT NOT NULL DEFAULT '',
    tool_name  TEXT,
    payload    JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (run_id, seq)
);

CREATE INDEX IF NOT EXISTS idx_agent_run_events_run_seq
    ON agent_run_events(run_id, seq);
