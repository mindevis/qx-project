-- Launch memory and extra arguments for VPS game servers.
--   mysql -u qx -p qx < docs/migrations/2026-08-22_game_server_launch_args.sql

ALTER TABLE game_servers
    ADD COLUMN min_memory_mb INT NULL,
    ADD COLUMN max_memory_mb INT NULL,
    ADD COLUMN extra_jvm_args TEXT NULL,
    ADD COLUMN extra_args TEXT NULL;
