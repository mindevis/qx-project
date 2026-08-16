-- Catalog metadata for mods/plugins/datapacks installed on a game server.
-- Files stay on the VPS; this column stores project/version info only.
-- AutoMigrate also adds the column on API boot.

ALTER TABLE game_servers
    ADD COLUMN content_resources MEDIUMTEXT NULL;
