-- Lock launcher instances created for a game server so clients cannot
-- change mods/resources or rebind another instance.
-- ADD COLUMN only — do not DROP on API boot.

ALTER TABLE launcher_instances
    ADD COLUMN managed_by_game_server_id CHAR(36) NULL;

CREATE INDEX idx_instances_managed_server
    ON launcher_instances (managed_by_game_server_id);
