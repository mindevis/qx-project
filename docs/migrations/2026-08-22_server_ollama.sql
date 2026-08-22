-- Host-level Ollama runtime on a dedicated server (install/start via QXAgent).
-- GORM AutoMigrate also creates this table on API boot.

CREATE TABLE IF NOT EXISTS server_ollama (
    server_id      CHAR(36) PRIMARY KEY,
    status         VARCHAR(32) NOT NULL DEFAULT 'not_installed',
    version        VARCHAR(64) NOT NULL DEFAULT '',
    bin_path       VARCHAR(512) NOT NULL DEFAULT '',
    root_dir       VARCHAR(512) NOT NULL DEFAULT '',
    models_dir     VARCHAR(512) NOT NULL DEFAULT '',
    listen_addr    VARCHAR(128) NOT NULL DEFAULT '127.0.0.1:11434',
    pulling_model  VARCHAR(256) NOT NULL DEFAULT '',
    last_error     TEXT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_server_ollama_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
