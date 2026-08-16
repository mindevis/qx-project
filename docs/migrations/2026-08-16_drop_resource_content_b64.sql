-- Drop leftover jar/zip LONGTEXT columns. File bytes live in MinIO;
-- AutoMigrate does not drop unused columns, so old rows still crush InnoDB.
--
-- Applied automatically on qxapi startup (dropResourceBlobColumns).
-- Safe to run manually if the API has not restarted yet:
--
--   mysql -u qx -p qx < docs/migrations/2026-08-16_drop_resource_content_b64.sql

ALTER TABLE instance_resource_upload_requests DROP COLUMN content_b64;
ALTER TABLE instance_resource_export_requests DROP COLUMN content_b64;
