-- Persist the current prepare/launch file phase so the web UI can show
-- what QXLauncher is downloading or validating.
-- ADD COLUMN only — do not DROP on API boot.

ALTER TABLE prepare_requests
    ADD COLUMN progress_message VARCHAR(256) NOT NULL DEFAULT '';

ALTER TABLE launch_requests
    ADD COLUMN progress_message VARCHAR(256) NOT NULL DEFAULT '';
