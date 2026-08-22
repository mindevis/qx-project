-- Host-level MariaDB / Percona MySQL for game servers (install via QXAgent).

CREATE TABLE IF NOT EXISTS server_mysql (
    server_id      CHAR(36) PRIMARY KEY,
    status         VARCHAR(32) NOT NULL DEFAULT 'not_installed',
    engine         VARCHAR(32) NOT NULL DEFAULT '',
    version        VARCHAR(32) NOT NULL DEFAULT '',
    method         VARCHAR(16) NOT NULL DEFAULT '',
    bind_addr      VARCHAR(64) NOT NULL DEFAULT '127.0.0.1',
    port           INT NOT NULL DEFAULT 3306,
    image          VARCHAR(128) NOT NULL DEFAULT '',
    root_password  VARCHAR(128) NOT NULL DEFAULT '',
    last_error     TEXT NULL,
    created_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at     TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    CONSTRAINT fk_server_mysql_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS server_mysql_database (
    id         CHAR(36) PRIMARY KEY,
    server_id  CHAR(36) NOT NULL,
    name       VARCHAR(64) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    UNIQUE KEY ux_mysql_db_name (server_id, name),
    CONSTRAINT fk_mysql_db_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS server_mysql_user (
    id         CHAR(36) PRIMARY KEY,
    server_id  CHAR(36) NOT NULL,
    username   VARCHAR(32) NOT NULL,
    host       VARCHAR(255) NOT NULL DEFAULT '%',
    password   VARCHAR(128) NOT NULL DEFAULT '',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    UNIQUE KEY ux_mysql_user_host (server_id, username, host),
    CONSTRAINT fk_mysql_user_server FOREIGN KEY (server_id) REFERENCES servers (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;

CREATE TABLE IF NOT EXISTS server_mysql_grant (
    id          CHAR(36) PRIMARY KEY,
    user_id     CHAR(36) NOT NULL,
    database_id CHAR(36) NOT NULL,
    privileges  TEXT NULL,
    UNIQUE KEY ux_mysql_grant (user_id, database_id),
    CONSTRAINT fk_mysql_grant_user FOREIGN KEY (user_id) REFERENCES server_mysql_user (id) ON DELETE CASCADE,
    CONSTRAINT fk_mysql_grant_db FOREIGN KEY (database_id) REFERENCES server_mysql_database (id) ON DELETE CASCADE
) ENGINE=InnoDB DEFAULT CHARSET=utf8mb4 COLLATE=utf8mb4_unicode_ci;
