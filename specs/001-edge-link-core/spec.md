# Feature Specification: Edge-Link Core System

**Feature Branch**: `001-edge-link-core`
**Created**: 2025-10-19
**Status**: Draft
**Input**: User description: "End-to-end direct connection system based on WireGuard P2P networking with NAT traversal"

## User Scenarios & Testing *(mandatory)*

### User Story 1 - Device Registration and Network Connection (Priority: P1)

A device owner installs the Edge-Link client on their device (desktop, mobile, or IoT), registers with the control plane using a pre-shared key, and establishes a secure P2P connection with peer devices in their virtual network.

**Why this priority**: This is the foundational user flow - without device registration and connection, no other features can function. This represents the minimum viable product (MVP) that delivers immediate value: secure device-to-device connectivity.

**Independent Test**: Can be fully tested by installing a client, registering it with a server address and pre-shared key, and verifying that the device receives a virtual IP and can ping at least one other registered device. Delivers immediate value: secure connectivity.

**Acceptance Scenarios**:

1. **Given** a fresh device with Edge-Link client installed, **When** the user enters server address and pre-shared key, **Then** the system generates device keypair, registers successfully, receives virtual IP, and stores encrypted configuration locally
2. **Given** a registered device with valid configuration, **When** the device starts the Edge-Link daemon, **Then** it performs NAT detection, establishes WireGuard tunnel to at least one peer, and reports online status to control plane
3. **Given** two registered devices in the same virtual network, **When** both are online, **Then** they can communicate directly via their virtual IPs (ICMP ping succeeds)
4. **Given** devices behind symmetric NAT (worst-case scenario), **When** direct P2P connection fails, **Then** the system automatically falls back to TURN relay and maintains connectivity

---

### User Story 2 - Network Administration and Monitoring (Priority: P2)

A network administrator logs into the management UI to view device status, monitor connection health, review network topology, manage device lifecycle (approve/revoke/rotate keys), and respond to alerts.

**Why this priority**: After establishing basic connectivity (P1), administrators need visibility and control to manage the network at scale. This enables operational management and troubleshooting.

**Independent Test**: Can be tested independently by having an admin user log into the web UI with existing registered devices, view dashboard metrics (online devices, bandwidth, alerts), navigate to device details, and perform one management action (e.g., revoke a device). Delivers value: operational visibility and control.

**Acceptance Scenarios**:

1. **Given** an authenticated administrator on the management UI, **When** they view the dashboard, **Then** they see current metrics (online devices count, total bandwidth, tunnel success rate, active alerts) with real-time updates via WebSocket
2. **Given** an administrator viewing the device list, **When** they select a specific device, **Then** they see detailed information (virtual IP, NAT type, WireGuard configuration, connection history, recent metrics)
3. **Given** a device with expired or compromised credentials, **When** the administrator triggers key rotation or device revocation, **Then** the device is notified, new keys are generated (rotation) or access is denied (revocation), and audit logs record the action
4. **Given** a connection failure between two devices, **When** the administrator reviews the network topology view, **Then** they can identify the failed tunnel, view error logs, and access diagnostic information

---

### User Story 3 - Automated Health Monitoring and Alerting (Priority: P3)

The system continuously monitors device health (connectivity, bandwidth, latency, failed handshakes), detects anomalies, and sends alerts to administrators via configured channels (email, webhook, enterprise messaging).

**Why this priority**: This enhances operational maturity by enabling proactive issue detection and resolution. It builds on P1 (connectivity) and P2 (monitoring UI) to provide automated oversight.

**Independent Test**: Can be tested by simulating failure conditions (disconnect a device, induce high latency, trigger excessive failed authentication attempts), verifying that alerts are generated with correct severity levels, and confirming delivery via configured notification channels. Delivers value: proactive problem detection.

**Acceptance Scenarios**:

1. **Given** a device that loses connectivity to all peers for more than 5 minutes, **When** the heartbeat timeout threshold is exceeded, **Then** the system generates a "Device Offline" alert with HIGH severity and notifies configured administrators
2. **Given** a device reporting latency above 500ms p95 consistently for 10 minutes, **When** the performance degradation threshold is met, **Then** the system generates a "High Latency" alert with MEDIUM severity
3. **Given** a device with 10+ failed authentication attempts in 1 minute, **When** the security threshold is exceeded, **Then** the system generates a "Potential Attack" alert with CRITICAL severity, temporarily locks the device, and notifies security team
4. **Given** upcoming key expiration (30 days before expiry), **When** the scheduled check runs, **Then** the system generates a "Key Rotation Reminder" alert with LOW severity and includes rotation instructions

---

### User Story 4 - Cross-Platform Client Support (Priority: P4)

Users can install and use Edge-Link clients on diverse platforms (Linux, Windows, macOS desktop; Android, iOS mobile; ARM-based IoT/container environments) with consistent configuration and behavior across all platforms.

**Why this priority**: This extends device coverage to support heterogeneous environments. While important for adoption, it builds on the foundational connectivity (P1) which can initially be demonstrated on a single platform.

**Independent Test**: Can be tested by installing platform-specific clients on 3+ different platforms (e.g., Windows desktop, Android mobile, Raspberry Pi IoT), completing registration using the same virtual network, and verifying all devices can ping each other. Delivers value: multi-platform support.

**Acceptance Scenarios**:

1. **Given** a user on Windows/macOS/Linux desktop, **When** they install the GUI/CLI client and complete registration, **Then** they successfully establish tunnels using either wireguard-go (userspace) or kernel WireGuard module
2. **Given** a user on Android/iOS mobile, **When** they install the app and register, **Then** they connect using the official WireGuard SDK and maintain connectivity during network transitions (WiFi to cellular)
3. **Given** an IoT device (ARM-based, limited resources) running Linux, **When** the lightweight CLI daemon is deployed, **Then** it registers, establishes tunnels, and operates within resource constraints (<50MB memory, <5% CPU idle)
4. **Given** a containerized workload (Docker/Kubernetes), **When** the Edge-Link sidecar container is added, **Then** the workload gains secure P2P connectivity to other network members without host network privileges

---

### Edge Cases

- **What happens when a device switches networks** (e.g., laptop moves from office WiFi to home WiFi)? The client must re-perform NAT detection, update peer with new endpoint information, and re-establish tunnels without manual intervention.

- **What happens when the control plane is unreachable** (temporary outage, network partition)? Existing P2P tunnels must remain active (data plane resilience), devices cache last-known peer configurations, and reconnect to control plane when available to sync state.

- **What happens when both devices are behind symmetric NAT with no STUN/TURN available**? The system must fail gracefully, log the connection attempt with clear diagnostics, notify administrators of the limitation, and provide fallback guidance (manual port forwarding or TURN deployment).

- **What happens when a device's key is compromised**? The administrator must be able to immediately revoke the device (blacklist public key), force re-registration with new keypair, and audit all sessions initiated with the compromised key.

- **What happens when concurrent key rotation occurs** (multiple devices rotating simultaneously)? The system must handle race conditions gracefully using pessimistic locking or versioned key updates, ensure no device is left with mismatched keys, and roll back failed rotations.

- **What happens when a device runs out of virtual IPs in the subnet**? The system must detect subnet exhaustion, alert administrators, provide clear guidance on expanding the subnet or reclaiming unused IPs, and block new registrations until resolved.

- **What happens during WireGuard handshake failures** (mismatched keys, replay attacks, clock skew)? The client must log detailed handshake failure reasons, increment failure counters (for alerting), and retry with exponential backoff up to a maximum retry count before escalating to support.

## Requirements *(mandatory)*

### Functional Requirements

#### Client Layer

- **FR-001**: System MUST provide native clients for desktop platforms (Linux, Windows, macOS) supporting both GUI and CLI modes
- **FR-002**: System MUST provide mobile apps for Android and iOS using official WireGuard SDKs
- **FR-003**: System MUST provide lightweight CLI daemon for IoT and containerized environments (ARM and x86_64 architectures)
- **FR-004**: Clients MUST generate cryptographically secure device keypairs (Ed25519 or Curve25519) on first run
- **FR-005**: Clients MUST register with control plane via REST API (`/api/v1/device/register`) providing public key, device fingerprint, and network capabilities
- **FR-006**: Clients MUST receive and locally store virtual IP address, subnet configuration, peer list, and STUN/TURN server addresses in encrypted format
- **FR-007**: Clients MUST support one-click configuration import/export for backup and migration
- **FR-008**: Clients MUST run a daemon process that monitors WireGuard interface health and auto-reconnects on failure
- **FR-009**: Clients MUST perform NAT type detection using STUN protocol and attempt UDP hole punching for P2P connections
- **FR-010**: Clients MUST fall back to TURN relay when direct P2P connection fails (symmetric NAT, firewall restrictions)
- **FR-011**: Clients MUST send periodic heartbeats (every 30 seconds default) and metrics (bandwidth, latency, failure counts) to control plane
- **FR-012**: Clients MUST collect structured logs and support log export or push to control plane for diagnostics

#### Control Plane

- **FR-013**: System MUST expose REST/gRPC APIs for device management (registration, configuration, status updates)
- **FR-014**: System MUST provide WebSocket endpoints for real-time state push and alert delivery to management UI
- **FR-015**: System MUST authenticate devices using pre-shared key (initial registration) and device keypair signature (ongoing operations)
- **FR-016**: System MUST manage device lifecycle: registration, virtual IP allocation, peer list distribution, key rotation scheduling, device revocation
- **FR-017**: System MUST maintain virtual network topology, assign IPs from configurable subnets, and generate peer routing policies
- **FR-018**: System MUST coordinate NAT traversal by facilitating STUN-based capability exchange and TURN relay assignment
- **FR-019**: System MUST record all administrative actions (device registration, revocation, key rotation, configuration changes) in audit logs with timestamps and actor identity
- **FR-020**: System MUST trigger alerts based on configurable thresholds (device offline >5min, latency >500ms p95, failed auth >10/min, key expiration <30days)
- **FR-021**: System MUST deliver alerts via multiple channels (email, webhook, enterprise messaging integration)
- **FR-022**: System MUST persist entity data (organizations, devices, keys, virtual networks, sessions, alerts, audit logs) in relational database (PostgreSQL)
- **FR-023**: System MUST cache online device state, session tokens, heartbeat information, and rate limits in Redis with TTL management
- **FR-024**: System MUST store diagnostic bundles (client logs, packet captures) in S3-compatible object storage
- **FR-025**: System MUST run scheduled background tasks: cleanup of stale devices/keys, key rotation planning, NAT capability testing, TURN node health checks
- **FR-026**: System MUST support horizontal scaling with stateless API services, Redis Sentinel for cache HA, and PostgreSQL replication for database HA

#### Management UI

- **FR-027**: System MUST provide web-based management UI built with modern framework (React 19 minimum) and responsive design (Ant Design 5 or equivalent)
- **FR-028**: UI MUST display dashboard with real-time metrics (online device count, aggregate bandwidth, tunnel success rate, active alerts) using charts (ECharts or equivalent)
- **FR-029**: UI MUST provide device detail view showing status, virtual IP, NAT type, WireGuard configuration, session history, and log download link
- **FR-030**: UI MUST visualize network topology with device interconnections and tunnel health indicators
- **FR-031**: UI MUST allow administrators to manage pre-shared keys, view device key lifecycle, and schedule/execute key rotation
- **FR-032**: UI MUST display alerts and audit logs with filtering (by device, severity, time range, action type) and timeline visualization
- **FR-033**: UI MUST provide administrative tools: remote configuration push, diagnostic trigger, device suspension/revocation
- **FR-034**: UI MUST implement role-based access control (RBAC) with predefined roles (system admin, network operator, auditor) and custom role support
- **FR-035**: UI MUST support authentication via OIDC/SAML identity providers and optional 2FA for sensitive operations

#### Data Plane

- **FR-036**: System MUST establish WireGuard tunnels using kernel module (when available) or wireguard-go userspace implementation
- **FR-037**: System MUST prefer direct P2P connections with STUN-assisted NAT traversal over TURN relay
- **FR-038**: System MUST encrypt all tunnel traffic using WireGuard's ChaCha20-Poly1305 cipher suite
- **FR-039**: System MUST implement cryptokey routing where each device's virtual IP is bound to its public key
- **FR-040**: Data plane connections MUST remain active during temporary control plane outages (data/control plane independence)

#### Monitoring and Operations

- **FR-041**: System MUST expose Prometheus-compatible metrics endpoints for control plane services, STUN/TURN servers, and WireGuard tunnel statistics
- **FR-042**: System MUST support centralized logging with structured format (JSON) including correlation IDs for distributed tracing
- **FR-043**: System MUST integrate distributed tracing (OpenTelemetry or Jaeger) for critical flows (device registration, NAT traversal, tunnel establishment)
- **FR-044**: System MUST provide health check endpoints (`/health`, `/ready`) for orchestration platforms (Kubernetes)
- **FR-045**: System MUST support blue-green deployment for control plane services with zero-downtime upgrades
- **FR-046**: System MUST support gradual client rollout with version channels (stable, beta, canary) and rollback capability

### Key Entities

- **Organization**: Represents a tenant or customer account; owns virtual networks and devices; has billing and quota limits
- **Virtual Network**: A logical subnet with CIDR range (e.g., 10.100.0.0/16); belongs to one organization; contains multiple devices
- **Device**: A registered endpoint (desktop, mobile, IoT); has unique ID, keypair, virtual IP, NAT type, online/offline status; belongs to one virtual network
- **Device Key**: Public/private keypair for WireGuard authentication; has creation date, expiration date, rotation schedule, revocation status
- **Pre-Shared Key**: Initial authentication secret for device registration; has expiration, usage count limit, revocation status; belongs to one organization
- **Peer Configuration**: Routing policy defining which devices can communicate; includes allowed IPs, endpoint addresses, persistent keepalive settings
- **Session**: Represents an active WireGuard tunnel between two devices; tracks start time, last handshake, bandwidth counters, latency metrics
- **Alert**: Event notification triggered by threshold violation; has severity (CRITICAL/HIGH/MEDIUM/LOW), type (offline/latency/security), status (active/acknowledged/resolved), notification recipients
- **Audit Log**: Immutable record of administrative action; includes timestamp, actor (user/system), action type, affected resource, before/after state
- **Diagnostic Bundle**: Compressed archive of client logs, system info, network diagnostics; uploaded to control plane; has expiration TTL

## Success Criteria *(mandatory)*

### Measurable Outcomes

- **SC-001**: Users can complete device registration (from install to first successful ping) in under 5 minutes without technical assistance
- **SC-002**: System establishes direct P2P connections for 80%+ device pairs (only 20% requiring TURN relay fallback)
- **SC-003**: Tunnel connection latency is within 10% of direct IP-to-IP baseline latency (minimal WireGuard overhead)
- **SC-004**: System maintains 99.9% data plane availability (existing tunnels remain active during control plane disruptions)
- **SC-005**: Control plane handles 10,000+ concurrent device connections with API response time p95 < 200ms
- **SC-006**: Device registration to first tunnel establishment completes in under 30 seconds in optimal network conditions
- **SC-007**: System successfully recovers from control plane outage within 60 seconds (devices reconnect and resync state)
- **SC-008**: 95%+ of connectivity issues are diagnosed through UI-accessible logs without requiring manual log collection
- **SC-009**: Administrators can identify root cause of tunnel failures in under 2 minutes using topology view and diagnostic tools
- **SC-010**: Zero security vulnerabilities rated HIGH or CRITICAL in annual penetration testing
- **SC-011**: Device key rotation completes for 1000+ devices in under 10 minutes with zero dropped connections during rotation
- **SC-012**: System scales horizontally to support 100,000+ registered devices across 1,000+ virtual networks with linear resource growth

## Assumptions

- **Network Infrastructure**: Assumes IPv4 networks with standard NAT configurations (Full Cone, Restricted Cone, Port-Restricted Cone, Symmetric); IPv6 support is out of scope for MVP
- **Trust Model**: Assumes organizations trust their own devices (insider threat prevention via device revocation, not runtime behavior analysis)
- **Certificate Authority**: Assumes organizations use pre-shared keys for initial device authentication; integration with external PKI/CA is future enhancement
- **STUN/TURN Infrastructure**: Assumes deployment includes at least one STUN server and one TURN server; public STUN services (e.g., Google STUN) can be used for testing but not production
- **Client Installation**: Assumes users have sufficient privileges to install software and create virtual network interfaces (requires admin/root on desktop, standard app install on mobile)
- **Database Scaling**: Assumes PostgreSQL handles up to 1M device records and 100M audit log entries without partitioning; sharding is future optimization
- **Compliance**: Assumes general data protection practices (encryption at rest/in transit); industry-specific compliance (HIPAA, PCI-DSS) requires additional controls not in MVP
- **Monitoring Infrastructure**: Assumes Prometheus/Grafana deployment exists or will be provisioned; metrics export is provided but dashboards are user-configured
