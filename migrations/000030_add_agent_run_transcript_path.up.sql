ALTER TABLE agent_runs
    ADD COLUMN IF NOT EXISTS transcript_path TEXT;
