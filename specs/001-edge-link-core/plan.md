# Implementation Plan: Edge-Link Core System

**Branch**: `001-edge-link-core` | **Date**: 2025-10-19 | **Spec**: [spec.md](./spec.md)
**Input**: Feature specification from `/home/await/project/edge-link/specs/001-edge-link-core/spec.md`

**Note**: This plan covers the complete Edge-Link system: client applications, control plane services, management UI, and supporting infrastructure.

## Summary

Edge-Link is an end-to-end direct connection system implementing WireGuard-based P2P networking with NAT traversal capabilities. The system enables devices across heterogeneous platforms (desktop, mobile, IoT) to establish secure, encrypted tunnels through a centralized control plane that coordinates device registration, NAT traversal (STUN/TURN), key management, and network topology. The architecture follows a three-layer model: Client Layer (TUN/virtual network interfaces), Control Plane (REST/WebSocket APIs, coordination services), and Data Plane (WireGuard tunnels with P2P preference and TURN fallback). A React-based management UI provides administrators with real-time visibility, device lifecycle management, and operational tools.

**Technical Approach** (from research): Multi-repository monorepo structure with three primary codebases: (1) Go-based control plane using microservices architecture (Gin HTTP, gRPC inter-service, Fx DI), (2) Platform-specific clients (Go for desktop CLI/daemon, native mobile apps wrapping WireGuard SDKs), (3) React 19 + TypeScript frontend with Ant Design 5. Infrastructure leverages PostgreSQL for persistent state, Redis for session/cache, WireGuard kernel module or userspace implementation, and external STUN/TURN servers for NAT traversal.

## Technical Context

**Language/Version**:
- Backend: Go 1.21+ (control plane services, desktop clients)
- Frontend: TypeScript 5+ with React 19
- Mobile: Swift 5.9+ (iOS), Kotlin 1.9+ (Android)

**Primary Dependencies**:
- Backend: gin-gonic/gin (HTTP router), grpc/grpc-go (RPC), uber-go/fx (DI), gorm (ORM), go-redis/redis, wireguard-go
- Frontend: React 19, Vite 5, Ant Design 5, ECharts 5, TanStack Query (React Query), Zustand (state)
- Mobile: WireGuardKit (iOS), wireguard-android (Android)
- Infrastructure: PostgreSQL 14+, Redis 7+, Prometheus, Grafana, Loki/ELK, Jaeger/OpenTelemetry

**Storage**:
- Primary: PostgreSQL 14+ (organizations, devices, keys, virtual networks, sessions, alerts, audit logs)
- Cache: Redis 7+ (online state, tokens, heartbeats, rate limits) with Sentinel HA
- Object Storage: S3-compatible (MinIO/AWS S3) for diagnostic bundles and client logs

**Testing**:
- Backend: Go's built-in testing framework + testify assertions, gomock for mocks
- Frontend: Vitest (unit), Playwright (E2E), React Testing Library
- Integration: Testcontainers for PostgreSQL/Redis, Docker Compose for NAT simulation scenarios
- Load: k6 (API), custom scripts for WireGuard tunnel throughput

**Target Platform**:
- Control Plane: Linux servers (Docker containers on Kubernetes)
- Desktop Clients: Linux (kernel WireGuard), Windows 10+ (wintun/wireguard-go), macOS 11+ (wireguard-go)
- Mobile Clients: iOS 15+, Android 8.0+ (API level 26+)
- IoT/Container: ARM64/AMD64 Linux with minimal dependencies

**Project Type**: Multi-component web application (backend services + frontend SPA + multi-platform clients)

**Performance Goals**:
- Control Plane API: p95 latency < 200ms, 10,000+ concurrent device connections per instance
- Device Registration: Complete flow (registration to first tunnel) < 30 seconds (optimal network)
- WireGuard Tunnel: Latency overhead < 10% vs baseline IP-to-IP
- Data Plane: 99.9% availability (independent of control plane)
- P2P Success Rate: 80%+ direct connections (20% TURN fallback acceptable)

**Constraints**:
- Network: IPv4 only for MVP (IPv6 future enhancement), standard NAT types (Full Cone, Restricted Cone, Port-Restricted Cone, Symmetric)
- Client Privileges: Requires admin/root on desktop for TUN interface creation, standard app install on mobile
- Database: PostgreSQL handles up to 1M device records + 100M audit logs without sharding
- Security: Pre-shared key authentication (external PKI/CA integration is future enhancement)
- Deployment: Kubernetes with Helm charts, minimum 3-node cluster for HA
- Alert Thresholds: MVP (v1.0) uses hardcoded defaults with environment variable overrides; UI-based threshold configuration planned for v2.0 (see spec.md FR-020)
- Background Task Scheduling: Uses robfig/cron (Go library) for in-process task scheduling; see research.md R13 for rationale
- Mobile Clients: iOS/Android apps deferred to v2.0 (spec.md FR-002); v1.0 focuses on desktop platforms

**Scale/Scope**:
- Devices: 100,000+ registered devices across 1,000+ virtual networks
- Concurrent Connections: 10,000+ active sessions per control plane instance
- Geographic Distribution: Multi-region deployment support (control plane HA)
- Codebase: Estimated 50K-100K LOC (backend 40K, frontend 20K, clients 30K, infrastructure 10K)

## Constitution Check

*GATE: Must pass before Phase 0 research. Re-check after Phase 1 design.*

Verify compliance with Edge-Link Constitution (`.specify/memory/constitution.md`):

- [x] **Real Data Only** (Principle I): ✅ All testing uses real PostgreSQL/Redis, actual WireGuard tunnels, live STUN/TURN servers; integration tests run in Testcontainers with real dependencies; no mocked NAT traversal or network simulation
- [x] **Complete Implementation** (Principle II): ✅ Spec FR-001 to FR-046 cover comprehensive error handling (network failures, timeouts, handshake failures), edge cases (symmetric NAT, control plane outages, key compromise, IP exhaustion), security (authentication, authorization, key rotation), and performance (connection pooling, async I/O, caching)
- [x] **Production-Ready** (Principle III): ✅ Design uses SOLID principles (microservices with single responsibility), loose coupling (gRPC for inter-service communication, event-driven architecture for alerts), multi-environment config (dev/staging/prod via Helm values), structured logging with correlation IDs
- [x] **Clean Project** (Principle IV): ✅ No temporary files; documentation (quickstart.md, API docs) will be created as part of Phase 1 deliverables with user approval; post-implementation cleanup of test artifacts automated via CI/CD
- [x] **Distributed Resilience** (Principle V): ✅ Explicit timeout policies (FR-011: 30s heartbeat, FR-020: 5min offline threshold), idempotency (device registration retries safe), circuit breakers (TURN fallback on P2P failure), observability (Prometheus metrics, Jaeger tracing, structured logging), data/control plane independence (FR-040)

**Constitution Check Result**: ✅ ALL GATES PASSED - Proceeding to Phase 0

*No violations requiring justification*

## Project Structure

### Documentation (this feature)

```
specs/001-edge-link-core/
├── plan.md              # This file (/speckit.plan command output)
├── spec.md              # Feature specification (already created)
├── research.md          # Phase 0 output (/speckit.plan command)
├── data-model.md        # Phase 1 output (/speckit.plan command)
├── quickstart.md        # Phase 1 output (/speckit.plan command)
├── contracts/           # Phase 1 output (/speckit.plan command)
│   ├── control-plane-api-v1.yaml    # OpenAPI spec for REST/gRPC APIs
│   ├── websocket-events.md          # WebSocket event schema
│   └── client-metrics-schema.json   # Metrics reporting format
├── checklists/
│   └── requirements.md  # Specification quality checklist (already created)
└── tasks.md             # Phase 2 output (/speckit.tasks command - NOT created by /speckit.plan)
```

### Source Code (repository root)

```
# Multi-component structure (backend services + frontend SPA + clients)

# Control Plane (Go microservices)
backend/
├── cmd/                        # Service entry points
│   ├── api-gateway/           # HTTP/gRPC gateway, authentication
│   ├── device-service/        # Device registration, lifecycle management
│   ├── topology-service/      # Virtual network, IP allocation, routing
│   ├── nat-coordinator/       # STUN/TURN coordination, NAT traversal
│   ├── alert-service/         # Threshold monitoring, notification delivery
│   └── background-worker/     # Scheduled tasks (cleanup, key rotation, health checks)
├── internal/                   # Shared packages (not exported)
│   ├── domain/                # Domain models (Organization, Device, VirtualNetwork, etc.)
│   ├── repository/            # PostgreSQL repositories (GORM)
│   ├── cache/                 # Redis cache abstractions
│   ├── auth/                  # Authentication (pre-shared key, device signature verification)
│   ├── crypto/                # Key generation, WireGuard config building
│   ├── metrics/               # Prometheus instrumentation
│   └── config/                # Environment-specific configuration
├── pkg/                        # Public packages (can be imported by clients)
│   ├── api/                   # Generated gRPC/Protobuf code
│   └── contracts/             # API request/response schemas
└── tests/
    ├── integration/           # Integration tests (Testcontainers: PostgreSQL, Redis)
    ├── contract/              # API contract tests (OpenAPI validation)
    └── e2e/                   # End-to-end scenarios (device registration, NAT traversal)

# Management UI (React 19 SPA)
frontend/
├── src/
│   ├── components/            # Reusable UI components (Ant Design wrappers)
│   ├── pages/                 # Page-level components (Dashboard, DeviceDetails, Topology, etc.)
│   ├── services/              # API clients (TanStack Query, WebSocket)
│   ├── stores/                # Zustand stores (UI state, user preferences)
│   ├── hooks/                 # Custom React hooks
│   ├── types/                 # TypeScript type definitions
│   └── utils/                 # Utilities (formatters, validators)
├── public/                    # Static assets
└── tests/
    ├── unit/                  # Vitest unit tests
    └── e2e/                   # Playwright E2E tests

# Desktop Clients (Go CLI/daemon)
clients/desktop/
├── cmd/
│   ├── edgelink-cli/          # CLI for device registration, config management
│   └── edgelink-daemon/       # Background daemon (WireGuard monitoring, NAT punching)
├── internal/
│   ├── wireguard/             # WireGuard interface management (kernel module or wireguard-go)
│   ├── stun/                  # STUN client for NAT detection
│   ├── config/                # Local config storage (encrypted)
│   ├── metrics/               # Heartbeat and metrics reporter
│   └── platform/              # Platform-specific code (Linux, Windows, macOS)
└── tests/
    ├── integration/           # WireGuard tunnel tests
    └── platform/              # Platform-specific integration tests

# Mobile Clients (iOS Swift, Android Kotlin)
clients/mobile/
├── ios/                       # iOS app (Swift)
│   ├── EdgeLink/             # Main app target
│   │   ├── Views/            # SwiftUI views
│   │   ├── ViewModels/       # MVVM view models
│   │   ├── Services/         # API clients, WireGuard integration
│   │   └── Models/           # Data models
│   └── EdgeLinkTests/        # XCTest tests
└── android/                   # Android app (Kotlin)
    ├── app/src/main/
    │   ├── java/com/edgelink/
    │   │   ├── ui/           # Jetpack Compose UI
    │   │   ├── viewmodels/   # ViewModel (MVVM)
    │   │   ├── services/     # API clients, WireGuard integration
    │   │   └── models/       # Data classes
    │   └── res/              # Resources
    └── app/src/test/         # JUnit tests

# Infrastructure as Code
infrastructure/
├── helm/                      # Helm charts
│   ├── edge-link-control-plane/
│   │   ├── Chart.yaml
│   │   ├── values.yaml       # Default values (overridden per environment)
│   │   └── templates/        # Kubernetes manifests (Deployments, Services, ConfigMaps, Secrets)
│   └── edge-link-stun-turn/  # STUN/TURN server deployment
├── terraform/                 # Terraform for cloud resources (optional)
│   ├── aws/                  # AWS RDS (PostgreSQL), ElastiCache (Redis), S3
│   └── gcp/                  # GCP Cloud SQL, Memorystore, Cloud Storage
└── docker/
    ├── Dockerfile.api-gateway
    ├── Dockerfile.device-service
    ├── Dockerfile.frontend
    └── docker-compose.yml    # Local development stack

# Monitoring and Observability
monitoring/
├── prometheus/
│   ├── prometheus.yml        # Scrape configs for control plane services
│   └── alerts.yml            # Alerting rules
├── grafana/
│   ├── dashboards/           # Pre-built dashboards (device health, tunnel metrics, API performance)
│   └── datasources.yml       # Prometheus, Loki datasources
└── loki/
    └── loki-config.yml       # Log aggregation config
```

**Structure Decision**: Multi-component structure selected due to diverse technology requirements (Go backend, TypeScript frontend, Swift/Kotlin mobile clients) and deployment independence (control plane services can be scaled independently from frontend, clients are distributed separately). Each component has its own build pipeline, testing strategy, and release cadence while sharing API contracts (OpenAPI, Protobuf) and data models.

## Complexity Tracking

*Fill ONLY if Constitution Check has violations that must be justified*

| Violation | Why Needed | Simpler Alternative Rejected Because |
|-----------|------------|-------------------------------------|
| N/A | No violations | All constitution gates passed |

---

**Phase 0 and Phase 1 deliverables will follow in separate files:**
- `research.md` - Technical decisions and research findings
- `data-model.md` - Entity-relationship model and database schema
- `contracts/` - API specifications (OpenAPI, WebSocket events, metrics schema)
- `quickstart.md` - Deployment and usage guide

