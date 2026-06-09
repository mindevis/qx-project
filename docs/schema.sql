-- QXProject initial schema
-- PostgreSQL 16+

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- ---------------------------------------------------------------------------
-- Users
-- ---------------------------------------------------------------------------

CREATE TYPE user_tier AS ENUM ('free', 'premium');

CREATE TABLE users (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email           VARCHAR(255) NOT NULL UNIQUE,
    password_hash   VARCHAR(255) NOT NULL,
    username        VARCHAR(32),
    tier            user_tier NOT NULL DEFAULT 'free',
    skin_url        TEXT,
    cape_url        TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_users_email ON users (email);

-- ---------------------------------------------------------------------------
-- Launcher devices (mandatory site linking)
-- ---------------------------------------------------------------------------

CREATE TYPE device_link_status AS ENUM ('pending_link', 'linked', 'expired', 'revoked');

CREATE TABLE launcher_devices (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL UNIQUE,
    user_id         UUID REFERENCES users (id) ON DELETE SET NULL,
    guest_session_id UUID REFERENCES guest_sessions (id) ON DELETE SET NULL,
    status          device_link_status NOT NULL DEFAULT 'pending_link',
    user_code       VARCHAR(16),
    device_token_hash VARCHAR(255),
    hostname        VARCHAR(255),
    os              VARCHAR(64),
    launcher_version VARCHAR(32),
    link_expires_at TIMESTAMPTZ,
    linked_at       TIMESTAMPTZ,
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT device_owner CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL) OR
        (user_id IS NULL AND guest_session_id IS NOT NULL) OR
        (user_id IS NULL AND guest_session_id IS NULL AND status = 'pending_link')
    )
);

CREATE INDEX idx_launcher_devices_status ON launcher_devices (status);
CREATE INDEX idx_launcher_devices_user_code ON launcher_devices (user_code) WHERE status = 'pending_link';

-- ---------------------------------------------------------------------------
-- Guest sessions
-- ---------------------------------------------------------------------------

CREATE TABLE guest_sessions (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       VARCHAR(64) NOT NULL UNIQUE,
    guest_token_hash VARCHAR(255) NOT NULL,
    expires_at      TIMESTAMPTZ NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Mojang link (post-MVP)
-- ---------------------------------------------------------------------------

CREATE TABLE mojang_links (
    user_id         UUID PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    minecraft_uuid  UUID NOT NULL,
    username        VARCHAR(16) NOT NULL,
    access_token_enc BYTEA,
    refresh_token_enc BYTEA,
    linked_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Offline profiles (Local accounts in launcher)
-- ---------------------------------------------------------------------------

CREATE TABLE offline_profiles (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users (id) ON DELETE CASCADE,
    guest_session_id UUID REFERENCES guest_sessions (id) ON DELETE CASCADE,
    username        VARCHAR(16) NOT NULL,
    offline_uuid    UUID NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT offline_profile_owner CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL) OR
        (user_id IS NULL AND guest_session_id IS NOT NULL)
    )
);

-- ---------------------------------------------------------------------------
-- Modpacks
-- ---------------------------------------------------------------------------

CREATE TYPE modpack_source AS ENUM ('curseforge', 'modrinth', 'qx_custom');
CREATE TYPE mc_loader AS ENUM ('vanilla', 'forge', 'neoforge', 'fabric', 'quilt');

CREATE TABLE modpacks (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name            VARCHAR(255) NOT NULL,
    source          modpack_source NOT NULL,
    external_id     VARCHAR(64) NOT NULL,
    mc_version      VARCHAR(16) NOT NULL,
    loader          mc_loader NOT NULL,
    loader_version  VARCHAR(32),
    manifest        JSONB NOT NULL DEFAULT '{}',
    manifest_sha256 CHAR(64) NOT NULL,
    author_id       UUID REFERENCES users (id) ON DELETE SET NULL,
    visibility      VARCHAR(16) NOT NULL DEFAULT 'public',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_modpacks_source_external ON modpacks (source, external_id);

-- ---------------------------------------------------------------------------
-- Launcher instances (client)
-- ---------------------------------------------------------------------------

CREATE TABLE launcher_instances (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID REFERENCES users (id) ON DELETE CASCADE,
    guest_session_id UUID REFERENCES guest_sessions (id) ON DELETE CASCADE,
    name            VARCHAR(128) NOT NULL,
    mc_version      VARCHAR(16) NOT NULL,
    loader          mc_loader NOT NULL DEFAULT 'vanilla',
    loader_version  VARCHAR(32),
    modpack_id      UUID REFERENCES modpacks (id) ON DELETE SET NULL,
    java_path       TEXT,
    jvm_args        JSONB NOT NULL DEFAULT '[]',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT instance_owner CHECK (
        (user_id IS NOT NULL AND guest_session_id IS NULL) OR
        (user_id IS NULL AND guest_session_id IS NOT NULL)
    )
);

CREATE INDEX idx_instances_user ON launcher_instances (user_id);
CREATE INDEX idx_instances_guest ON launcher_instances (guest_session_id);
CREATE INDEX idx_instances_modpack ON launcher_instances (modpack_id);

-- ---------------------------------------------------------------------------
-- Servers (BYOS)
-- ---------------------------------------------------------------------------

CREATE TYPE server_type AS ENUM (
    'vanilla', 'paper', 'spigot', 'purpur',
    'forge', 'neoforge', 'fabric', 'quilt', 'hybrid'
);

CREATE TYPE server_status AS ENUM (
    'pending', 'deploying', 'offline', 'starting', 'online', 'stopping', 'error'
);

CREATE TABLE servers (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id        UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    name            VARCHAR(128) NOT NULL,
    slug            VARCHAR(64) NOT NULL,
    server_type     server_type NOT NULL DEFAULT 'vanilla',
    status          server_status NOT NULL DEFAULT 'pending',
    online_mode     BOOLEAN NOT NULL DEFAULT FALSE,
    modpack_id      UUID REFERENCES modpacks (id) ON DELETE SET NULL,
    mc_version      VARCHAR(16),
    loader          mc_loader,
    is_public       BOOLEAN NOT NULL DEFAULT FALSE,
    public_address  VARCHAR(255),
    public_description TEXT,
    config          JSONB NOT NULL DEFAULT '{}',
    agent_token_hash VARCHAR(255),
    last_seen_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (owner_id, slug)
);

CREATE INDEX idx_servers_owner ON servers (owner_id);
CREATE INDEX idx_servers_public ON servers (is_public) WHERE is_public = TRUE;
CREATE INDEX idx_servers_modpack ON servers (modpack_id);

-- ---------------------------------------------------------------------------
-- Server members (multi-admin)
-- ---------------------------------------------------------------------------

CREATE TYPE server_role AS ENUM ('owner', 'admin', 'viewer');

CREATE TABLE server_members (
    server_id       UUID NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
    user_id         UUID NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            server_role NOT NULL DEFAULT 'viewer',
    invited_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (server_id, user_id)
);

CREATE INDEX idx_server_members_user ON server_members (user_id);

-- ---------------------------------------------------------------------------
-- SSH credentials (encrypted private key)
-- ---------------------------------------------------------------------------

CREATE TABLE ssh_credentials (
    server_id       UUID PRIMARY KEY REFERENCES servers (id) ON DELETE CASCADE,
    host            VARCHAR(255) NOT NULL,
    port            INT NOT NULL DEFAULT 22,
    username        VARCHAR(64) NOT NULL,
    private_key_enc BYTEA NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Agents
-- ---------------------------------------------------------------------------

CREATE TABLE agents (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL UNIQUE REFERENCES servers (id) ON DELETE CASCADE,
    hostname        VARCHAR(255),
    os              VARCHAR(64) NOT NULL DEFAULT 'linux',
    agent_version   VARCHAR(32),
    connected_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Agent command idempotency log (optional server-side mirror)
-- ---------------------------------------------------------------------------

CREATE TABLE agent_command_log (
    request_id      UUID PRIMARY KEY,
    server_id       UUID NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
    command_type    VARCHAR(64) NOT NULL,
    response        JSONB,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agent_cmd_log_server ON agent_command_log (server_id, created_at DESC);

-- ---------------------------------------------------------------------------
-- Backups
-- ---------------------------------------------------------------------------

CREATE TABLE server_backups (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
    storage_key     VARCHAR(512) NOT NULL,
    size_bytes      BIGINT NOT NULL,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Deploy jobs
-- ---------------------------------------------------------------------------

CREATE TYPE deploy_status AS ENUM ('queued', 'running', 'success', 'failed');

CREATE TABLE deploy_jobs (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    server_id       UUID NOT NULL REFERENCES servers (id) ON DELETE CASCADE,
    status          deploy_status NOT NULL DEFAULT 'queued',
    log             TEXT,
    started_at      TIMESTAMPTZ,
    finished_at     TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- ---------------------------------------------------------------------------
-- Launch requests (site → tray bridge)
-- ---------------------------------------------------------------------------

CREATE TYPE launch_status AS ENUM (
    'queued', 'dispatched', 'running', 'completed', 'failed', 'expired'
);

CREATE TABLE launch_requests (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id       UUID NOT NULL REFERENCES launcher_devices (device_id),
    instance_id     UUID NOT NULL REFERENCES launcher_instances (id) ON DELETE CASCADE,
    offline_profile_id UUID REFERENCES offline_profiles (id),
    status          launch_status NOT NULL DEFAULT 'queued',
    pid             INT,
    exit_code       INT,
    error_code      VARCHAR(64),
    expires_at      TIMESTAMPTZ NOT NULL,
    dispatched_at   TIMESTAMPTZ,
    completed_at    TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_launch_pending ON launch_requests (device_id, status)
    WHERE status = 'queued';

-- ---------------------------------------------------------------------------
-- Audit log (append-only)
-- ---------------------------------------------------------------------------

CREATE TABLE audit_logs (
    id              BIGSERIAL PRIMARY KEY,
    action          VARCHAR(64) NOT NULL,
    actor_id        UUID,
    actor_type      VARCHAR(16),
    resource_type   VARCHAR(32),
    resource_id     UUID,
    metadata        JSONB NOT NULL DEFAULT '{}',
    ip              INET,
    user_agent      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_audit_actor ON audit_logs (actor_id, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_logs (resource_type, resource_id);

-- ---------------------------------------------------------------------------
-- User 2FA (post-MVP)
-- ---------------------------------------------------------------------------

ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_secret_enc BYTEA;
ALTER TABLE users ADD COLUMN IF NOT EXISTS totp_enabled BOOLEAN NOT NULL DEFAULT FALSE;

-- ---------------------------------------------------------------------------
-- Instance attachments (mods, shaders, resource packs — registered users)
-- ---------------------------------------------------------------------------

ALTER TABLE launcher_instances ADD COLUMN IF NOT EXISTS mods JSONB NOT NULL DEFAULT '[]';
ALTER TABLE launcher_instances ADD COLUMN IF NOT EXISTS resource_packs JSONB NOT NULL DEFAULT '[]';
ALTER TABLE launcher_instances ADD COLUMN IF NOT EXISTS shaders JSONB NOT NULL DEFAULT '[]';

-- ---------------------------------------------------------------------------
-- Trigger: updated_at
-- ---------------------------------------------------------------------------

CREATE OR REPLACE FUNCTION set_updated_at()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER tr_users_updated BEFORE UPDATE ON users
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER tr_instances_updated BEFORE UPDATE ON launcher_instances
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER tr_servers_updated BEFORE UPDATE ON servers
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();
CREATE TRIGGER tr_modpacks_updated BEFORE UPDATE ON modpacks
    FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- ---------------------------------------------------------------------------
-- Seed: auto-add owner as server_members row (app layer or trigger TBD)
-- ---------------------------------------------------------------------------
