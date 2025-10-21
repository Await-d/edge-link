# Data Model: Edge-Link Core System

**Feature**: Edge-Link Core System
**Branch**: 001-edge-link-core
**Date**: 2025-10-19

## Purpose

This document defines the entity-relationship model and database schema for the Edge-Link system. The schema supports all functional requirements (FR-001 to FR-046) with normalized design for data integrity, indexed for query performance, and partitioned for scale.

---

## Entity-Relationship Diagram

```
┌────────────────┐       ┌──────────────────┐       ┌─────────────┐
│ Organization   │──────<│ VirtualNetwork   │──────<│  Device     │
│ (Tenant)       │ 1:N   │ (Subnet)         │ 1:N   │  (Endpoint) │
└────────────────┘       └──────────────────┘       └─────────────┘
        │                                                    │
        │ 1:N                                                │ 1:N
        ├─────────────────────────┐                         ├──────────────┐
        │                         │                         │              │
        ▼                         ▼                         ▼              ▼
┌────────────────┐       ┌─────────────────┐       ┌─────────────┐ ┌─────────────┐
│ PreSharedKey   │       │  AdminUser      │       │ DeviceKey   │ │  Session    │
│ (Auth Token)   │       │  (UI Access)    │       │ (Keypair)   │ │  (Tunnel)   │
└────────────────┘       └─────────────────┘       └─────────────┘ └─────────────┘
                                                             │              │
                                                             │              │ N:M
                                                             ▼              ▼
                                            ┌────────────────────────────────────┐
                                            │      PeerConfiguration             │
                                            │  (Routing Policy: Device A ↔ B)    │
                                            └────────────────────────────────────┘

┌─────────────┐       ┌──────────────┐       ┌──────────────────┐
│   Alert     │       │  AuditLog    │       │ DiagnosticBundle │
│ (Event)     │       │ (Immutable)  │       │ (Logs Archive)   │
└─────────────┘       └──────────────┘       └──────────────────┘
```

---

## Core Entities

### 1. Organization

**Purpose**: Multi-tenant isolation; each organization owns virtual networks and devices

**Attributes**:
- `id` (UUID, PK): Unique organization identifier
- `name` (VARCHAR(255), UNIQUE, NOT NULL): Organization display name
- `slug` (VARCHAR(100), UNIQUE, NOT NULL): URL-safe identifier (e.g., "acme-corp")
- `max_devices` (INTEGER, DEFAULT 100): Device quota limit
- `max_virtual_networks` (INTEGER, DEFAULT 10): Virtual network quota
- `created_at` (TIMESTAMP, NOT NULL): Creation timestamp
- `updated_at` (TIMESTAMP, NOT NULL): Last modification timestamp

**Indexes**:
- Primary key: `id`
- Unique: `slug`

**Validation Rules**:
- `name` must be unique across all organizations
- `slug` must match regex `^[a-z0-9-]+$`
- `max_devices` >= current active device count

**Relationships**:
- 1:N with `VirtualNetwork` (one organization has many virtual networks)
- 1:N with `PreSharedKey` (one organization has many pre-shared keys)
- 1:N with `AdminUser` (one organization has many admin users)

---

### 2. VirtualNetwork

**Purpose**: Logical subnet for device grouping; defines IP address space

**Attributes**:
- `id` (UUID, PK): Unique network identifier
- `organization_id` (UUID, FK → Organization.id, NOT NULL): Owner organization
- `name` (VARCHAR(255), NOT NULL): Network display name
- `cidr` (CIDR, NOT NULL): IP range (e.g., "10.100.0.0/16")
- `gateway_ip` (INET, NOT NULL): Gateway address (e.g., "10.100.0.1")
- `dns_servers` (INET[], NULLABLE): Custom DNS servers for network
- `created_at` (TIMESTAMP, NOT NULL): Creation timestamp
- `updated_at` (TIMESTAMP, NOT NULL): Last modification timestamp

**Indexes**:
- Primary key: `id`
- Foreign key: `organization_id`
- Composite unique: (`organization_id`, `name`)

**Validation Rules**:
- `cidr` must not overlap with other virtual networks in same organization
- `gateway_ip` must be within `cidr` range
- `cidr` must be private IP range (10.0.0.0/8, 172.16.0.0/12, 192.168.0.0/16)

**Relationships**:
- N:1 with `Organization` (many networks belong to one organization)
- 1:N with `Device` (one network contains many devices)

---

### 3. Device

**Purpose**: Registered endpoint (desktop, mobile, IoT); holds virtual IP and connection state

**Attributes**:
- `id` (UUID, PK): Unique device identifier
- `virtual_network_id` (UUID, FK → VirtualNetwork.id, NOT NULL): Assigned network
- `name` (VARCHAR(255), NOT NULL): User-assigned device name
- `virtual_ip` (INET, NOT NULL): Assigned IP within virtual network CIDR
- `public_key` (TEXT, NOT NULL): Ed25519 public key (base64-encoded, 32 bytes)
- `platform` (ENUM: desktop_linux, desktop_windows, desktop_macos, mobile_ios, mobile_android, iot, container): Platform type
- `nat_type` (ENUM: full_cone, restricted_cone, port_restricted_cone, symmetric, unknown): Detected NAT type
- `last_seen_at` (TIMESTAMP, NULLABLE): Last heartbeat timestamp
- `online` (BOOLEAN, DEFAULT FALSE): Current online status
- `created_at` (TIMESTAMP, NOT NULL): Registration timestamp
- `updated_at` (TIMESTAMP, NOT NULL): Last modification timestamp

**Indexes**:
- Primary key: `id`
- Foreign key: `virtual_network_id`
- Unique: `virtual_ip` (per virtual network)
- Unique: `public_key`
- Index: `virtual_network_id, online` (for dashboard queries: "online devices per network")
- Index: `last_seen_at DESC` (for detecting stale devices)

**Validation Rules**:
- `virtual_ip` must be within `virtual_network_id.cidr` range
- `virtual_ip` cannot be `gateway_ip`
- `public_key` must be valid Ed25519 public key (32 bytes base64)
- `online` = TRUE only if `last_seen_at` within last 5 minutes

**Relationships**:
- N:1 with `VirtualNetwork` (many devices belong to one network)
- 1:N with `DeviceKey` (one device has many keys over lifetime, due to rotation)
- 1:N with `Session` (one device participates in many sessions)

**State Transitions**:
1. **Registered** (`created_at` set, `online` = FALSE): Device registered but not connected
2. **Online** (`online` = TRUE, `last_seen_at` recent): Device sending heartbeats
3. **Offline** (`online` = FALSE, `last_seen_at` stale): Heartbeat timeout exceeded
4. **Revoked** (soft delete via `deleted_at` column): Admin-initiated removal

---

### 4. DeviceKey

**Purpose**: Track device keypair lifecycle for audit and rotation

**Attributes**:
- `id` (UUID, PK): Unique key identifier
- `device_id` (UUID, FK → Device.id, NOT NULL): Owner device
- `public_key` (TEXT, NOT NULL): Ed25519 public key (base64-encoded)
- `private_key_fingerprint` (TEXT, NOT NULL): SHA256 hash of private key (never stored)
- `created_at` (TIMESTAMP, NOT NULL): Key generation timestamp
- `expires_at` (TIMESTAMP, NULLABLE): Expiration timestamp (NULL = no expiration)
- `revoked_at` (TIMESTAMP, NULLABLE): Revocation timestamp (NULL = active)
- `rotation_reason` (ENUM: scheduled, compromised, admin_requested, NULLABLE): Reason for rotation

**Indexes**:
- Primary key: `id`
- Foreign key: `device_id`
- Index: `device_id, created_at DESC` (for key history queries)
- Index: `expires_at` (for scheduled rotation tasks)

**Validation Rules**:
- Only one key per device with `revoked_at` = NULL (active key)
- `expires_at` must be > `created_at`
- `public_key` must match `device.public_key` for active key

**Relationships**:
- N:1 with `Device` (many keys belong to one device over time)

---

### 5. PreSharedKey

**Purpose**: Initial authentication secret for device registration (out-of-band distribution)

**Attributes**:
- `id` (UUID, PK): Unique PSK identifier
- `organization_id` (UUID, FK → Organization.id, NOT NULL): Owner organization
- `key_hash` (TEXT, NOT NULL): bcrypt hash of PSK (never store plaintext)
- `name` (VARCHAR(255), NOT NULL): Admin-assigned label (e.g., "Office Devices")
- `max_uses` (INTEGER, DEFAULT NULL): Maximum registration count (NULL = unlimited)
- `used_count` (INTEGER, DEFAULT 0): Current registration count
- `expires_at` (TIMESTAMP, NULLABLE): Expiration timestamp
- `created_at` (TIMESTAMP, NOT NULL): Creation timestamp
- `revoked_at` (TIMESTAMP, NULLABLE): Revocation timestamp

**Indexes**:
- Primary key: `id`
- Foreign key: `organization_id`
- Index: `organization_id, revoked_at` (for active PSK queries)

**Validation Rules**:
- `used_count` <= `max_uses` (if `max_uses` is not NULL)
- Cannot be used if `revoked_at` is not NULL or `expires_at` < NOW()

**Relationships**:
- N:1 with `Organization` (many PSKs belong to one organization)

---

### 6. PeerConfiguration

**Purpose**: Defines routing policy between two devices (which devices can communicate)

**Attributes**:
- `id` (UUID, PK): Unique config identifier
- `device_a_id` (UUID, FK → Device.id, NOT NULL): First peer
- `device_b_id` (UUID, FK → Device.id, NOT NULL): Second peer
- `allowed_ips` (CIDR[], NOT NULL): IP ranges allowed through tunnel (default: peer's virtual_ip/32)
- `persistent_keepalive` (INTEGER, DEFAULT 25): WireGuard keepalive interval (seconds)
- `created_at` (TIMESTAMP, NOT NULL): Configuration timestamp

**Indexes**:
- Primary key: `id`
- Foreign key: `device_a_id`, `device_b_id`
- Unique: (`device_a_id`, `device_b_id`) with CHECK (`device_a_id` < `device_b_id`) to prevent duplicates

**Validation Rules**:
- `device_a_id` and `device_b_id` must be in same `virtual_network_id`
- `device_a_id` != `device_b_id`
- `allowed_ips` must be within virtual network CIDR

**Relationships**:
- N:2 with `Device` (one config involves two devices)

---

### 7. Session

**Purpose**: Represents active WireGuard tunnel between two devices; tracks metrics

**Attributes**:
- `id` (UUID, PK): Unique session identifier
- `device_a_id` (UUID, FK → Device.id, NOT NULL): First peer
- `device_b_id` (UUID, FK → Device.id, NOT NULL): Second peer
- `started_at` (TIMESTAMP, NOT NULL): Tunnel establishment timestamp
- `last_handshake_at` (TIMESTAMP, NOT NULL): Most recent WireGuard handshake
- `endpoint_a` (TEXT, NULLABLE): Device A's public endpoint (IP:port)
- `endpoint_b` (TEXT, NULLABLE): Device B's public endpoint (IP:port)
- `connection_type` (ENUM: p2p_direct, turn_relay): Connection method
- `bytes_sent_a` (BIGINT, DEFAULT 0): Bytes sent by device A
- `bytes_received_a` (BIGINT, DEFAULT 0): Bytes received by device A
- `bytes_sent_b` (BIGINT, DEFAULT 0): Bytes sent by device B
- `bytes_received_b` (BIGINT, DEFAULT 0): Bytes received by device B
- `latency_ms` (INTEGER, NULLABLE): Round-trip latency (milliseconds)
- `ended_at` (TIMESTAMP, NULLABLE): Tunnel termination timestamp (NULL = active)

**Indexes**:
- Primary key: `id`
- Foreign key: `device_a_id`, `device_b_id`
- Index: `device_a_id, ended_at` (for active session queries)
- Index: `started_at DESC` (for recent session history)

**Validation Rules**:
- `last_handshake_at` >= `started_at`
- `ended_at` >= `started_at` (if not NULL)
- `device_a_id` != `device_b_id`

**Relationships**:
- N:2 with `Device` (one session involves two devices)

---

### 8. Alert

**Purpose**: Event notification triggered by threshold violations or anomalies

**Attributes**:
- `id` (UUID, PK): Unique alert identifier
- `device_id` (UUID, FK → Device.id, NULLABLE): Related device (NULL for system-wide alerts)
- `severity` (ENUM: critical, high, medium, low): Alert severity
- `type` (ENUM: device_offline, high_latency, failed_auth, key_expiration, tunnel_failure): Alert category
- `message` (TEXT, NOT NULL): Human-readable alert message
- `metadata` (JSONB, NULLABLE): Additional context (e.g., {"failed_attempts": 12})
- `status` (ENUM: active, acknowledged, resolved): Alert status
- `created_at` (TIMESTAMP, NOT NULL): Alert generation timestamp
- `acknowledged_at` (TIMESTAMP, NULLABLE): Admin acknowledgment timestamp
- `resolved_at` (TIMESTAMP, NULLABLE): Resolution timestamp

**Indexes**:
- Primary key: `id`
- Foreign key: `device_id`
- Index: `status, severity, created_at DESC` (for active alert queries)
- Index: `device_id, created_at DESC` (for device-specific alerts)

**Validation Rules**:
- `acknowledged_at` >= `created_at`
- `resolved_at` >= `created_at`
- Cannot transition from `resolved` back to `active`

**Relationships**:
- N:1 with `Device` (many alerts for one device)

---

### 9. AuditLog

**Purpose**: Immutable record of all administrative actions for compliance

**Attributes**:
- `id` (UUID, PK): Unique log entry identifier
- `organization_id` (UUID, FK → Organization.id, NOT NULL): Organization context
- `actor_id` (UUID, FK → AdminUser.id, NULLABLE): Acting user (NULL for system actions)
- `action` (ENUM: device_registered, device_revoked, key_rotated, psk_created, alert_acknowledged, ...): Action type
- `resource_type` (ENUM: device, virtual_network, pre_shared_key, alert, ...): Affected resource type
- `resource_id` (UUID, NOT NULL): Affected resource identifier
- `before_state` (JSONB, NULLABLE): State before action
- `after_state` (JSONB, NULLABLE): State after action
- `ip_address` (INET, NULLABLE): Actor's IP address
- `user_agent` (TEXT, NULLABLE): Actor's user agent string
- `created_at` (TIMESTAMP, NOT NULL): Action timestamp (immutable)

**Indexes**:
- Primary key: `id`
- Foreign key: `organization_id`, `actor_id`
- Composite index: (`resource_type`, `resource_id`, `created_at DESC`) for resource history
- Index: `created_at DESC` for timeline queries
- Partitioned by month (PostgreSQL declarative partitioning) when exceeds 10M rows

**Validation Rules**:
- `created_at` is immutable (no updates allowed)
- `before_state` and `after_state` must be valid JSON

**Relationships**:
- N:1 with `Organization` (many logs for one organization)
- N:1 with `AdminUser` (many logs by one admin)

---

### 10. DiagnosticBundle

**Purpose**: Compressed archive of client logs and system diagnostics for troubleshooting

**Attributes**:
- `id` (UUID, PK): Unique bundle identifier
- `device_id` (UUID, FK → Device.id, NOT NULL): Source device
- `file_path` (TEXT, NOT NULL): S3 object key (e.g., "diagnostics/2025-10-19/device-123.tar.gz")
- `file_size_bytes` (BIGINT, NOT NULL): Bundle size
- `uploaded_at` (TIMESTAMP, NOT NULL): Upload timestamp
- `expires_at` (TIMESTAMP, NOT NULL): TTL for automatic deletion (default: 30 days)
- `metadata` (JSONB, NULLABLE): Bundle contents summary (log lines, timespan, etc.)

**Indexes**:
- Primary key: `id`
- Foreign key: `device_id`
- Index: `expires_at` for cleanup task
- Index: `device_id, uploaded_at DESC` for device history

**Validation Rules**:
- `expires_at` > `uploaded_at`
- `file_path` must match regex `^diagnostics/\d{4}-\d{2}-\d{2}/.+\.tar\.gz$`

**Relationships**:
- N:1 with `Device` (many bundles from one device)

---

## Supporting Entities

### 11. AdminUser

**Purpose**: Management UI user accounts with RBAC

**Attributes**:
- `id` (UUID, PK)
- `organization_id` (UUID, FK → Organization.id, NOT NULL)
- `email` (VARCHAR(255), UNIQUE, NOT NULL)
- `full_name` (VARCHAR(255), NOT NULL)
- `role` (ENUM: system_admin, network_operator, auditor): RBAC role
- `oidc_subject` (TEXT, NULLABLE): External identity provider subject ID
- `created_at` (TIMESTAMP, NOT NULL)
- `last_login_at` (TIMESTAMP, NULLABLE)

**Relationships**:
- N:1 with `Organization`

---

## Database Constraints

### Foreign Key Cascades

- `Device.virtual_network_id` ON DELETE CASCADE (deleting network deletes devices)
- `DeviceKey.device_id` ON DELETE CASCADE
- `Session.device_a_id`, `Session.device_b_id` ON DELETE CASCADE
- `Alert.device_id` ON DELETE SET NULL (preserve alerts after device deletion)
- `AuditLog.*` ON DELETE RESTRICT (audit logs are immutable, never delete)

### Check Constraints

- `Organization.max_devices > 0`
- `VirtualNetwork.cidr` in private IP ranges (10/8, 172.16/12, 192.168/16)
- `Device.virtual_ip` within `VirtualNetwork.cidr`
- `PreSharedKey.used_count <= max_uses`
- `Session.device_a_id < device_b_id` (prevent duplicate pairs)

---

## Indexing Strategy

**Query Patterns**:

1. **Dashboard**: Fetch online device count per organization
   - Index: `(virtual_network_id, online)` on `Device`

2. **Device Details**: Load device + active sessions + recent alerts
   - Indexes: `device_id` on `Session`, `Alert`

3. **Audit Timeline**: Fetch recent actions for resource
   - Index: `(resource_type, resource_id, created_at DESC)` on `AuditLog`

4. **Background Tasks**: Find expired keys/PSKs for cleanup
   - Indexes: `expires_at` on `DeviceKey`, `PreSharedKey`, `DiagnosticBundle`

**Composite Index Justification**:
- `(virtual_network_id, online)` supports `WHERE virtual_network_id = ? AND online = true` with index-only scan
- `(resource_type, resource_id, created_at DESC)` supports audit history queries with ORDER BY optimization

---

## Schema Evolution

**Migration Strategy**:
- Use golang-migrate for version-controlled migrations
- Backward-compatible changes only (add columns, never drop)
- Multi-step migrations for breaking changes (add column → backfill → drop old column)

**Partitioning Plan**:
- `AuditLog` partitioned by month when exceeds 10M rows (declarative partitioning)
- `Session` partitioned by quarter when exceeds 50M rows

---

## Data Retention

- `AuditLog`: Retain 2 years, then archive to cold storage (S3 Glacier)
- `Session`: Retain 90 days after `ended_at`
- `DiagnosticBundle`: Retain per `expires_at` (default 30 days), then delete from S3
- `Alert`: Retain 1 year after `resolved_at`

---

## Summary

The data model supports all functional requirements (FR-001 to FR-046) with:
- **Normalization**: 3NF to prevent anomalies (device-network-organization hierarchy)
- **Integrity**: Foreign keys, check constraints enforce business rules
- **Performance**: Indexes on high-traffic queries (dashboard, device details, audit timeline)
- **Scalability**: Partitioning plan for high-volume tables (audit logs, sessions)
- **Auditability**: Immutable `AuditLog` with before/after state snapshots

Ready for API contract generation (Phase 1 next step).
