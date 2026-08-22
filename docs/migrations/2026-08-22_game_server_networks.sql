-- Game server projects (Velocity + lobby + backends) grouped per dedicated host.
-- GORM AutoMigrate also creates these tables on API boot.

CREATE TABLE IF NOT EXISTS game_server_networks (
    id                CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    server_id         CHAR(36) NOT NULL,
    name              VARCHAR(128) NOT NULL,
    forwarding_secret VARCHAR(64) NOT NULL,
    created_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at        TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_game_server_networks_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_game_server_networks_server_id ON game_server_networks (server_id);

CREATE TABLE IF NOT EXISTS game_server_network_members (
    id             CHAR(36) PRIMARY KEY DEFAULT (UUID()),
    network_id     CHAR(36) NOT NULL,
    game_server_id CHAR(36) NOT NULL,
    role           VARCHAR(32) NOT NULL,
    alias          VARCHAR(64) NOT NULL,
    sort_order     INT NOT NULL DEFAULT 0,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY idx_network_members_game_server (game_server_id),
    CONSTRAINT fk_network_members_network FOREIGN KEY (network_id) REFERENCES game_server_networks (id) ON DELETE CASCADE,
    CONSTRAINT fk_network_members_game_server FOREIGN KEY (game_server_id) REFERENCES game_servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE INDEX idx_game_server_network_members_network_id ON game_server_network_members (network_id);
