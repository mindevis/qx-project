-- Drop leftover jar/zip LONGTEXT columns. File bytes live in MinIO;
-- AutoMigrate does not drop unused columns, so old rows still crush InnoDB.
--
-- Do NOT run this at API boot: DROP COLUMN rebuilds the table and can
-- block qxapi from listening (nginx 502). Apply in a maintenance window:
--
--   mysql -u qx -p qx < docs/migrations/2026-08-16_drop_resource_content_b64.sql

ALTER TABLE instance_resource_upload_requests DROP COLUMN content_b64;
ALTER TABLE instance_resource_export_requests DROP COLUMN content_b64;
