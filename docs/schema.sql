-- QXProject initial schema
-- MySQL 8.0+

SET NAMES utf8mb4;
SET FOREIGN_KEY_CHECKS = 0;

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------

CREATE TABLE users (
    id              CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    username        VARCHAR(32) NULL,
    tier            ENUM('free', 'premium') NOT NULL DEFAULT 'free',
    skin_url        TEXT NULL,
    cape_url        TEXT NULL,
    totp_secret_enc BLOB NULL,
    totp_enabled    TINYINT(1) NOT NULL DEFAULT 0,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Guest sessions
-- ---------------------------------------------------------------------------

CREATE TABLE guest_sessions (
    id               CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    device_id        VARCHAR(64) NOT NULL UNIQUE,
    guest_token_hash VARCHAR(255) NOT NULL,
    expires_at       TIMESTAMP NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Launcher devices (mandatory site linking)
-- ---------------------------------------------------------------------------

CREATE TABLE launcher_devices (
    id                CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    device_id         CHAR(36) NOT NULL UNIQUE,
    user_id           CHAR(36) NULL,
    guest_session_id  CHAR(36) NULL,
    status            ENUM('pending_link', 'linked', 'expired', 'revoked') NOT NULL DEFAULT 'pending_link',
    user_code         VARCHAR(16) NULL,  -- legacy; unused (link by HWID device_id in URL)
    device_token_hash VARCHAR(255) NULL,
    hostname          VARCHAR(255) NULL,
    os                VARCHAR(64) NULL,
    launcher_version  VARCHAR(32) NULL,
    link_expires_at   TIMESTAMP NULL,
    linked_at         TIMESTAMP NULL,
    last_seen_at      TIMESTAMP NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_launcher_devices_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE SET NULL,
    CONSTRAINT fk_launcher_devices_guest FOREIGN KEY (guest_session_id) REFERENCES guest_sessions (id) ON DELETE SET NULL,
    CONSTRAINT device_owner CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL) OR
        (user_id IS NULL AND guest_session_id IS NOT NULL) OR
        (user_id IS NULL AND guest_session_id IS NULL AND status = 'pending_link')
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_launcher_devices_status ON launcher_devices (status);
CREATE INDEX idx_launcher_devices_user_code ON launcher_devices (user_code, status);

-- ---------------------------------------------------------------------------
-- Mojang link (post-MVP)
-- ---------------------------------------------------------------------------

CREATE TABLE mojang_links (
    user_id           CHAR(36) PRIMARY KEY,
    minecraft_uuid    CHAR(36) NOT NULL,
    username          VARCHAR(16) NOT NULL,
    access_token_enc  BLOB NULL,
    refresh_token_enc BLOB NULL,
    linked_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_mojang_links_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Offline profiles (Local accounts in launcher)
-- ---------------------------------------------------------------------------

CREATE TABLE offline_profiles (
    id               CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id          CHAR(36) NULL,
    guest_session_id CHAR(36) NULL,
    username         VARCHAR(16) NOT NULL,
    offline_uuid     CHAR(36) NOT NULL,
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_offline_profiles_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_offline_profiles_guest FOREIGN KEY (guest_session_id) REFERENCES guest_sessions (id) ON DELETE CASCADE,
    CONSTRAINT offline_profile_owner CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL) OR
        (user_id IS NULL AND guest_session_id IS NOT NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Modpacks
-- ---------------------------------------------------------------------------

CREATE TABLE modpacks (
    id              CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    name            VARCHAR(255) NOT NULL,
    source          ENUM('curseforge', 'modrinth', 'qx_custom') NOT NULL,
    external_id     VARCHAR(64) NOT NULL,
    mc_version      VARCHAR(16) NOT NULL,
    loader          ENUM('vanilla', 'forge', 'neoforge', 'fabric', 'quilt') NOT NULL,
    loader_version  VARCHAR(32) NULL,
    manifest        JSON NOT NULL DEFAULT (JSON_OBJECT()),
    manifest_sha256 CHAR(64) NOT NULL,
    author_id       CHAR(36) NULL,
    visibility      VARCHAR(16) NOT NULL DEFAULT 'public',
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_modpacks_author FOREIGN KEY (author_id) REFERENCES users (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_modpacks_source_external ON modpacks (source, external_id);

-- ---------------------------------------------------------------------------
-- Launcher instances (client)
-- ---------------------------------------------------------------------------

CREATE TABLE launcher_instances (
    id               CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    user_id          CHAR(36) NULL,
    guest_session_id CHAR(36) NULL,
    name             VARCHAR(128) NOT NULL,
    mc_version       VARCHAR(16) NOT NULL,
    loader           ENUM('vanilla', 'forge', 'neoforge', 'fabric', 'quilt') NOT NULL DEFAULT 'vanilla',
    loader_version   VARCHAR(32) NULL,
    modpack_id       CHAR(36) NULL,
    java_path        TEXT NULL,
    jvm_args         JSON NOT NULL DEFAULT (JSON_ARRAY()),
    mods             JSON NOT NULL DEFAULT (JSON_ARRAY()),
    resource_packs   JSON NOT NULL DEFAULT (JSON_ARRAY()),
    shaders          JSON NOT NULL DEFAULT (JSON_ARRAY()),
    created_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at       TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_instances_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_instances_guest FOREIGN KEY (guest_session_id) REFERENCES guest_sessions (id) ON DELETE CASCADE,
    CONSTRAINT fk_instances_modpack FOREIGN KEY (modpack_id) REFERENCES modpacks (id) ON DELETE SET NULL,
    CONSTRAINT instance_owner CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL) OR
        (user_id IS NULL AND guest_session_id IS NOT NULL)
    )
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_instances_user ON launcher_instances (user_id);
CREATE INDEX idx_instances_guest ON launcher_instances (guest_session_id);
CREATE INDEX idx_instances_modpack ON launcher_instances (modpack_id);

-- ---------------------------------------------------------------------------
-- Servers (BYOS)
-- ---------------------------------------------------------------------------

CREATE TABLE servers (
    id                 CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    owner_id           CHAR(36) NOT NULL,
    name               VARCHAR(128) NOT NULL,
    slug               VARCHAR(64) NOT NULL,
    server_type        ENUM('vanilla', 'paper', 'spigot', 'purpur', 'forge', 'neoforge', 'fabric', 'quilt', 'hybrid') NOT NULL DEFAULT 'vanilla',
    status             ENUM('pending', 'deploying', 'offline', 'starting', 'online', 'stopping', 'error') NOT NULL DEFAULT 'pending',
    online_mode        TINYINT(1) NOT NULL DEFAULT 0,
    modpack_id         CHAR(36) NULL,
    mc_version         VARCHAR(16) NULL,
    loader             ENUM('vanilla', 'forge', 'neoforge', 'fabric', 'quilt') NULL,
    is_public          TINYINT(1) NOT NULL DEFAULT 0,
    public_address     VARCHAR(255) NULL,
    public_description TEXT NULL,
    config             JSON NOT NULL DEFAULT (JSON_OBJECT()),
    hybrid_platform    VARCHAR(32) NULL,
    agent_token_hash   VARCHAR(255) NULL,
    last_seen_at       TIMESTAMP NULL,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY uq_servers_owner_slug (owner_id, slug),
    CONSTRAINT fk_servers_owner FOREIGN KEY (owner_id) REFERENCES users (id) ON DELETE CASCADE,
    CONSTRAINT fk_servers_modpack FOREIGN KEY (modpack_id) REFERENCES modpacks (id) ON DELETE SET NULL
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_servers_owner ON servers (owner_id);
CREATE INDEX idx_servers_public ON servers (is_public);
CREATE INDEX idx_servers_modpack ON servers (modpack_id);

-- ---------------------------------------------------------------------------
-- Server members (multi-admin)
-- ---------------------------------------------------------------------------

CREATE TABLE server_members (
    server_id  CHAR(36) NOT NULL,
    user_id    CHAR(36) NOT NULL,
    role       ENUM('owner', 'admin', 'viewer') NOT NULL DEFAULT 'viewer',
    invited_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (server_id, user_id),
    CONSTRAINT fk_server_members_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE,
    CONSTRAINT fk_server_members_user FOREIGN KEY (user_id) REFERENCES users (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_server_members_user ON server_members (user_id);

-- ---------------------------------------------------------------------------
-- SSH credentials (encrypted private key)
-- ---------------------------------------------------------------------------

CREATE TABLE ssh_credentials (
    server_id       CHAR(36) PRIMARY KEY,
    host            VARCHAR(255) NOT NULL,
    port            INT NOT NULL DEFAULT 22,
    username        VARCHAR(64) NOT NULL,
    private_key_enc BLOB NOT NULL,
    created_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at      TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_ssh_credentials_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Agents
-- ---------------------------------------------------------------------------

CREATE TABLE agents (
    id            CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    server_id     CHAR(36) NOT NULL UNIQUE,
    hostname      VARCHAR(255) NULL,
    os            VARCHAR(64) NOT NULL DEFAULT 'linux',
    agent_version VARCHAR(32) NULL,
    connected_at  TIMESTAMP NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_agents_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Agent command idempotency log
-- ---------------------------------------------------------------------------

CREATE TABLE agent_command_log (
    request_id   CHAR(36) PRIMARY KEY,
    server_id    CHAR(36) NOT NULL,
    command_type VARCHAR(64) NOT NULL,
    response     JSON NULL,
    created_at   TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_agent_cmd_log_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_agent_cmd_log_server ON agent_command_log (server_id, created_at);

-- ---------------------------------------------------------------------------
-- Backups
-- ---------------------------------------------------------------------------

CREATE TABLE server_backups (
    id          CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    server_id   CHAR(36) NOT NULL,
    storage_key VARCHAR(512) NOT NULL,
    size_bytes  BIGINT NOT NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_server_backups_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Deploy jobs
-- ---------------------------------------------------------------------------

CREATE TABLE deploy_jobs (
    id          CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    server_id   CHAR(36) NOT NULL,
    status      ENUM('queued', 'running', 'success', 'failed') NOT NULL DEFAULT 'queued',
    log         TEXT NULL,
    started_at  TIMESTAMP NULL,
    finished_at TIMESTAMP NULL,
    created_at  TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_deploy_jobs_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

-- ---------------------------------------------------------------------------
-- Launch requests (site → tray bridge)
-- ---------------------------------------------------------------------------

CREATE TABLE launch_requests (
    id                 CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    device_id          CHAR(36) NOT NULL,
    instance_id        CHAR(36) NOT NULL,
    offline_profile_id CHAR(36) NULL,
    status             ENUM('queued', 'dispatched', 'running', 'completed', 'failed', 'expired') NOT NULL DEFAULT 'queued',
    pid                INT NULL,
    exit_code          INT NULL,
    error_code         VARCHAR(64) NULL,
    expires_at         TIMESTAMP NOT NULL,
    dispatched_at      TIMESTAMP NULL,
    completed_at       TIMESTAMP NULL,
    created_at         TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    CONSTRAINT fk_launch_requests_device FOREIGN KEY (device_id) REFERENCES launcher_devices (device_id),
    CONSTRAINT fk_launch_requests_instance FOREIGN KEY (instance_id) REFERENCES launcher_instances (id) ON DELETE CASCADE,
    CONSTRAINT fk_launch_requests_profile FOREIGN KEY (offline_profile_id) REFERENCES offline_profiles (id)
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_launch_pending ON launch_requests (device_id, status);

-- ---------------------------------------------------------------------------
-- Audit log (append-only)
-- ---------------------------------------------------------------------------

CREATE TABLE audit_logs (
    id            BIGINT UNSIGNED NOT NULL AUTO_INCREMENT PRIMARY KEY,
    action        VARCHAR(64) NOT NULL,
    actor_id      CHAR(36) NULL,
    actor_type    VARCHAR(16) NULL,
    resource_type VARCHAR(32) NULL,
    resource_id   CHAR(36) NULL,
    metadata      JSON NOT NULL DEFAULT (JSON_OBJECT()),
    ip            VARCHAR(45) NULL,
    user_agent    TEXT NULL,
    created_at    TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_audit_actor ON audit_logs (actor_id, created_at);
CREATE INDEX idx_audit_resource ON audit_logs (resource_type, resource_id);

SET FOREIGN_KEY_CHECKS = 1;
