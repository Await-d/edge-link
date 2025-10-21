# Technical Research: Edge-Link Core System

**Feature**: Edge-Link Core System
**Branch**: 001-edge-link-core
**Date**: 2025-10-19

## Purpose

This document captures technical decisions, architectural patterns, and best practices research for implementing the Edge-Link end-to-end direct connection system. All decisions are made to satisfy functional requirements (FR-001 to FR-046) while adhering to the Edge-Link Constitution principles.

---

## R1: Microservices Architecture vs Monolith

###  Decision

**Chosen**: Microservices architecture with 6 independent services (api-gateway, device-service, topology-service, nat-coordinator, alert-service, background-worker)

### Rationale

1. **Independent Scaling**: NAT coordination and device registration have different load profiles; microservices allow horizontal scaling of high-traffic services (device-service, nat-coordinator) independently
2. **Fault Isolation**: Failure in alert-service doesn't impact device registration or tunnel establishment (Constitution Principle V: Distributed Resilience)
3. **Technology Flexibility**: Future services (e.g., advanced analytics, ML-based NAT prediction) can use different languages/frameworks without affecting core services
4. **Team Autonomy**: Different teams can own services with clear boundaries (device lifecycle, network topology, alerting)

### Alternatives Considered

- **Monolith**: Rejected because single deployment unit creates blast radius for failures; scaling requires duplicating entire application including low-traffic components (alert delivery); violates Constitution Principle V (failure isolation)
- **Serverless Functions**: Rejected because WireGuard tunnel coordination requires stateful connections and predictable latency; cold starts violate performance requirements (FR-005: registration < 30s)

### Implementation Notes

- Use gRPC for inter-service communication (lower latency than REST, Protobuf schema enforcement)
- API Gateway handles external HTTP/REST, translates to gRPC for internal services
- Shared domain models in `backend/internal/domain/` to maintain consistency
- Circuit breakers (e.g., sony/gobreaker) for inter-service calls to prevent cascading failures

---

## R2: WireGuard Integration Approach

### Decision

**Chosen**: Hybrid approach - kernel module where available (Linux), wireguard-go userspace for cross-platform (Windows, macOS, iOS, Android)

### Rationale

1. **Performance**: Kernel module provides ~30% better throughput and lower CPU usage on Linux servers/desktops (critical for Success Criterion SC-003: <10% latency overhead)
2. **Cross-Platform Compatibility**: wireguard-go enables Windows/macOS support without driver signing complexities; mobile platforms use official WireGuardKit/wireguard-android SDKs
3. **Maintenance**: Official WireGuard projects are actively maintained; avoids custom crypto implementations (security risk)

### Alternatives Considered

- **Kernel-Only**: Rejected because Windows/macOS lack stable in-tree kernel modules; third-party drivers (wintun) require complex installation
- **Userspace-Only**: Rejected because Linux performance degradation unacceptable for server deployments; FR-041 metrics show 2x CPU usage vs kernel

### Implementation Notes

- Desktop client detects kernel module availability at runtime (`modprobe wireguard` check on Linux)
- Fallback to wireguard-go if kernel module unavailable or version incompatible
- Configuration format identical (wg-quick compatible) regardless of implementation
- Mobile apps always use SDK (WireGuardKit for iOS, wireguard-android for Android) - no runtime detection needed

---

## R3: NAT Traversal Strategy

### Decision

**Chosen**: Three-tier approach - (1) STUN for NAT detection, (2) ICE-lite for hole punching, (3) TURN relay fallback

### Rationale

1. **Success Rate**: ICE-lite achieves 80-85% P2P success rate for Cone NAT types (meets SC-002: 80%+ direct connections)
2. **Simplicity**: ICE-lite (client-controlled) simpler than full ICE (no server-side candidate generation); sufficient for WireGuard UDP (single port)
3. **Cost Efficiency**: TURN relay bandwidth expensive; minimize fallback to symmetric NAT scenarios only

### Alternatives Considered

- **STUN-Only**: Rejected because symmetric NAT (15-20% of networks) cannot hole-punch without relay; violates FR-010 (TURN fallback required)
- **Full ICE**: Rejected because complexity overhead (STUN/TURN server candidates, trickle ICE) unnecessary for single UDP port; ICE-lite sufficient

### Implementation Notes

- Use pion/stun library (Go) for STUN client in desktop/control plane
- NAT coordinator service maintains TURN relay pool, assigns based on geographic proximity
- Clients attempt P2P first (30s timeout), fallback to TURN automatically (FR-010)
- Monitor relay usage (Prometheus metric: `turn_relay_active_sessions`) to optimize TURN capacity

---

## R4: Database Schema Design

### Decision

**Chosen**: Normalized PostgreSQL schema with 10 core tables (organizations, virtual_networks, devices, device_keys, pre_shared_keys, peer_configurations, sessions, alerts, audit_logs, diagnostic_bundles)

### Rationale

1. **Data Integrity**: Foreign key constraints enforce referential integrity (e.g., device must belong to valid virtual_network); prevents orphaned records
2. **Query Performance**: Indexes on high-traffic queries (devices by organization, sessions by device_id, audit logs by timestamp)
3. **Auditability**: Immutable audit_logs table with composite index (resource_id, timestamp) supports compliance requirements

### Alternatives Considered

- **NoSQL (MongoDB)**: Rejected because transactional consistency critical for device registration (allocate IP, create keys, update topology atomically); NoSQL weak ACID guarantees risk IP conflicts
- **Event Sourcing**: Rejected because complexity overhead unnecessary for MVP; current state queries (device status, online count) would require complex projections

### Implementation Notes

- Use GORM for ORM with auto-migration disabled (explicit migrations via golang-migrate)
- Partition `audit_logs` by month (PostgreSQL declarative partitioning) once exceeds 10M rows
- Connection pooling: max 100 connections per service instance (PostgreSQL default: 100 total)
- Read replicas for dashboard queries (Grafana, management UI) to offload primary

---

## R5: Authentication and Authorization

### Decision

**Chosen**: Dual authentication - (1) Pre-shared key (HMAC-SHA256) for initial device registration, (2) Ed25519 device signature for ongoing API calls

### Rationale

1. **Bootstrap Security**: Pre-shared key distributed out-of-band (admin portal, email) prevents unauthorized device registration
2. **Device Identity**: Ed25519 keypair generated on device; public key serves as immutable device ID (cannot be stolen without private key access)
3. **Performance**: Ed25519 signature verification <1ms (vs RSA 2048-bit ~3ms); critical for 10,000+ concurrent connections (FR-005)

### Alternatives Considered

- **OAuth2/JWT**: Rejected because devices aren't user-facing (no browser flow); JWT rotation complexity adds failure points (violates Constitution Principle V: simplicity)
- **mTLS**: Rejected because certificate distribution/rotation overhead; pre-shared key + device signature simpler operationally

### Implementation Notes

- Pre-shared key stored hashed (bcrypt) in PostgreSQL `pre_shared_keys` table
- Device public key stored in `devices.public_key` (base64-encoded Ed25519 32 bytes)
- API Gateway validates signature using `crypto/ed25519` standard library
- Management UI uses OIDC/SAML for admin authentication (separate from device auth)

---

## R6: Real-Time Communication (WebSocket)

### Decision

**Chosen**: WebSocket for bidirectional real-time updates (device status changes, alerts, topology updates) with JSON message framing

### Rationale

1. **Low Latency**: WebSocket push <100ms vs polling (5-30s latency); critical for admin responsiveness (FR-014)
2. **Bandwidth Efficiency**: Single persistent connection vs repeated HTTP requests; reduces control plane load
3. **Browser Compatibility**: WebSocket widely supported (IE11+, all modern browsers); no fallback needed

### Alternatives Considered

- **Server-Sent Events (SSE)**: Rejected because unidirectional (server→client only); cannot support client-initiated actions (e.g., trigger diagnostic)
- **HTTP Long Polling**: Rejected because higher latency and server overhead (connection per poll); doesn't scale to 10,000+ devices

### Implementation Notes

- Use gorilla/websocket library (mature, well-tested)
- Message format: `{"type": "DEVICE_STATUS_CHANGE", "payload": {...}, "timestamp": "2025-10-19T10:00:00Z"}`
- Heartbeat ping/pong every 30s to detect stale connections
- Reconnection logic with exponential backoff (1s, 2s, 4s, 8s, max 60s)

---

## R7: Metrics and Observability

### Decision

**Chosen**: Prometheus (metrics), Jaeger (tracing), Loki (logs) stack with OpenTelemetry SDK instrumentation

### Rationale

1. **Industry Standard**: Prometheus de facto standard for Kubernetes metrics; Grafana dashboards ecosystem rich
2. **Distributed Tracing**: Jaeger provides end-to-end visibility for device registration flow (API Gateway → Device Service → Topology Service); critical for diagnosing latency issues
3. **Log Aggregation**: Loki's label-based indexing (vs full-text) reduces storage costs while maintaining queryability

### Alternatives Considered

- **ELK Stack**: Rejected because Elasticsearch resource-intensive (RAM, CPU); Loki achieves 70% storage reduction for structured logs
- **Datadog/New Relic**: Rejected because SaaS vendor lock-in; open-source stack maintains operational independence

### Implementation Notes

- Use prometheus/client_golang for metric instrumentation (counter, histogram, gauge)
- OpenTelemetry Go SDK for trace context propagation (gRPC metadata, HTTP headers)
- Structured logging with zap library (JSON output, correlation IDs)
- Standard metrics: `http_request_duration_seconds` (histogram), `active_devices` (gauge), `tunnel_failures_total` (counter)

---

## R8: Frontend State Management

### Decision

**Chosen**: TanStack Query (server state) + Zustand (client state) split

### Rationale

1. **Clear Separation**: Server state (device list, metrics) managed by TanStack Query (caching, invalidation); client state (UI preferences, selected device) by Zustand
2. **Automatic Refetching**: TanStack Query handles background refetch, stale-while-revalidate; reduces manual useEffect logic
3. **TypeScript Integration**: Both libraries first-class TypeScript support; type-safe API calls

### Alternatives Considered

- **Redux Toolkit**: Rejected because boilerplate overhead for server state (actions, reducers, thunks); TanStack Query declarative queries simpler
- **Recoil**: Rejected because smaller ecosystem vs Zustand; Zustand simpler API for global UI state

### Implementation Notes

- TanStack Query for all API calls (devices, virtual networks, alerts, topology)
- Query invalidation on mutations (e.g., device revocation invalidates device list query)
- Zustand for: selected device ID, dashboard time range, UI theme, sidebar collapsed state
- WebSocket messages trigger TanStack Query invalidation (real-time updates)

---

## R9: Mobile App Architecture

### Decision

**Chosen**: MVVM (Model-View-ViewModel) pattern with platform-specific UI (SwiftUI for iOS, Jetpack Compose for Android)

### Rationale

1. **Testability**: ViewModels contain business logic (API calls, WireGuard config generation), unit-testable without UI
2. **Platform-Native**: SwiftUI/Compose provide best UX (animations, gestures); cross-platform frameworks (Flutter, React Native) compromise on native feel
3. **Maintainability**: Separation of concerns (View = presentation, ViewModel = logic, Model = data) reduces coupling

### Alternatives Considered

- **Flutter**: Rejected because adds Dart language to tech stack; WireGuard SDK integration via platform channels adds complexity
- **React Native**: Rejected because JavaScript bridge performance overhead; native WireGuard SDK integration fragile

### Implementation Notes

- iOS: Combine framework for reactive bindings (ViewModel → View)
- Android: LiveData/StateFlow for reactive bindings
- Shared data models (JSON codecs) across platforms via OpenAPI-generated schemas
- Network layer: URLSession (iOS), Retrofit+OkHttp (Android)

---

## R10: Deployment Strategy

### Decision

**Chosen**: Kubernetes with Helm charts, blue-green deployment for control plane, gradual rollout for clients

### Rationale

1. **Zero-Downtime**: Blue-green deployment maintains old version during migration; rollback instant if issues detected (FR-045)
2. **Scalability**: Kubernetes HorizontalPodAutoscaler scales services based on CPU/memory/custom metrics (active devices)
3. **Client Safety**: Gradual rollout (stable → beta → canary channels) limits blast radius of client bugs

### Alternatives Considered

- **Docker Compose**: Rejected because no orchestration for multi-node HA; suitable for dev only
- **Rolling Deployment**: Rejected because brief downtime during pod termination; blue-green eliminates this

### Implementation Notes

- Helm chart per service (api-gateway, device-service, etc.) with shared values (database URL, Redis endpoint)
- Kubernetes Ingress for external traffic (NGINX or Traefik)
- Client version channels: `stable` (99% of users), `beta` (0.5%), `canary` (0.5%); controlled via control plane config
- Health checks: liveness (process alive), readiness (database connected, ready for traffic)

---

## R11: Error Handling and Resilience Patterns

### Decision

**Chosen**: Circuit breaker (inter-service), retry with exponential backoff (external APIs), timeout policies (all network calls)

### Rationale

1. **Cascading Failure Prevention**: Circuit breaker stops calling failing service after threshold (e.g., 5 failures in 10s); prevents thread pool exhaustion
2. **Transient Failures**: Exponential backoff (1s, 2s, 4s, 8s) handles temporary network issues without overwhelming downstream
3. **Predictability**: Explicit timeouts (30s for STUN, 5s for database, 10s for gRPC) prevent indefinite hangs

### Alternatives Considered

- **No Resilience Patterns**: Rejected because violates Constitution Principle V (Distributed Resilience); single timeout can cascade
- **Retries Only**: Rejected because retry storms amplify failures; circuit breaker required for protection

### Implementation Notes

- Use sony/gobreaker for circuit breaker (configurable thresholds, half-open state)
- Retry logic: 3 attempts with exponential backoff, jitter to avoid thundering herd
- Timeout context propagation: `context.WithTimeout` in all gRPC/HTTP calls
- Metrics: `circuit_breaker_state{service="device-service"}` (open/closed/half-open)

---

## R12: Testing Strategy

### Decision

**Chosen**: Test pyramid - 70% unit, 20% integration, 10% E2E; NAT simulation via Docker Compose networks

### Rationale

1. **Speed**: Unit tests run in <5s (Go test), provide fast feedback; integration tests <30s (Testcontainers), E2E <5min (Playwright)
2. **Real Dependencies**: Testcontainers PostgreSQL/Redis ensures tests match production (Constitution Principle I: Real Data Only)
3. **NAT Coverage**: Docker networks with iptables rules simulate Cone/Symmetric NAT; validates hole-punching logic

### Alternatives Considered

- **Mocks Only**: Rejected because violates Constitution Principle I; mocked NAT traversal cannot catch real-world edge cases
- **Manual Testing**: Rejected because insufficient coverage for 46 functional requirements; automation required

### Implementation Notes

- Unit tests: Go `testing` package + testify assertions
- Integration tests: Testcontainers for PostgreSQL, Redis; docker-compose for multi-service tests
- E2E tests: Playwright for management UI, custom Go clients for device registration flow
- NAT simulation: `docker network create --driver bridge --subnet 10.200.0.0/16 nat-test`

---

## Summary

All technical decisions satisfy functional requirements (FR-001 to FR-046) and comply with Edge-Link Constitution principles:

- **Real Data Only** (I): Testcontainers, real PostgreSQL/Redis, actual WireGuard tunnels in integration tests
- **Complete Implementation** (II): Circuit breakers, retries, timeouts, comprehensive error handling
- **Production-Ready** (III): Microservices (SOLID), Helm charts (multi-environment), structured logging
- **Clean Project** (IV): No temporary tools; automated cleanup in CI/CD
- **Distributed Resilience** (V): Circuit breakers, idempotent operations, explicit timeouts, observability stack

No further research required - ready for Phase 1 (data-model.md, contracts/, quickstart.md).
