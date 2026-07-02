-- Fix collation mismatch on existing production databases.
--
-- Root cause: game_servers and game_server_monitoring_feedback were created by
-- GORM AutoMigrate using MySQL 8 default utf8mb4_0900_ai_ci, while tables from
-- docs/schema.sql use utf8mb4_unicode_ci. JOINs in monitoring queries then fail
-- with: Illegal mix of collations (utf8mb4_unicode_ci,IMPLICIT) and
-- (utf8mb4_0900_ai_ci,IMPLICIT).
--
-- Run on the production MySQL instance (safe online; brief metadata lock per table):
--
--   mysql -u qx -p qx < docs/migrations/2026-06-30_game_servers_collation.sql

ALTER TABLE game_servers
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;

ALTER TABLE game_server_monitoring_feedback
    CONVERT TO CHARACTER SET utf8mb4 COLLATE utf8mb4_unicode_ci;
