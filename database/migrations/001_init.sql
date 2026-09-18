-- Scanläufe bilden die zentrale Einheit der gespeicherten Netzwerkanalyse.
CREATE TABLE scans (
    id BIGSERIAL PRIMARY KEY,
    scan_id VARCHAR(32) NOT NULL UNIQUE,
    target VARCHAR(100) NOT NULL,
    status VARCHAR(20) NOT NULL,
    raw_output TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT chk_scan_status
        CHECK (status IN ('pending', 'running', 'completed', 'failed'))
);

-- Ein Scan kann einen oder mehrere erreichbare Hosts enthalten.
CREATE TABLE hosts (
    id BIGSERIAL PRIMARY KEY,
    scan_id BIGINT NOT NULL REFERENCES scans(id) ON DELETE CASCADE,
    ip_address INET NOT NULL,
    hostname VARCHAR(255),
    state VARCHAR(20) NOT NULL
);

-- Zu jedem erkannten Host können mehrere Ports gespeichert werden.
CREATE TABLE ports (
    id BIGSERIAL PRIMARY KEY,
    host_id BIGINT NOT NULL REFERENCES hosts(id) ON DELETE CASCADE,
    port_number INTEGER NOT NULL,
    protocol VARCHAR(10) NOT NULL,
    state VARCHAR(20) NOT NULL,

    CONSTRAINT chk_port_number
        CHECK (port_number BETWEEN 1 AND 65535)
);

-- Erkannte Dienste werden getrennt von den Portinformationen gespeichert.
CREATE TABLE services (
    id BIGSERIAL PRIMARY KEY,
    port_id BIGINT NOT NULL UNIQUE REFERENCES ports(id) ON DELETE CASCADE,
    name VARCHAR(100),
    product VARCHAR(255),
    version VARCHAR(100)
);

-- Indizes unterstützen spätere Abfragen der Scan-Historie.
CREATE INDEX idx_hosts_scan_id ON hosts(scan_id);
CREATE INDEX idx_ports_host_id ON ports(host_id);