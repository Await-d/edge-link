# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.
进行中文对话交流

## Project Overview

Edge-Link is an end-to-end direct connection system (端到端直连系统) implementing WireGuard-based P2P networking with NAT traversal, STUN/TURN coordination, and centralized control plane management.

**Architecture**:
- **Client Layer**: Cross-platform TUN/virtual network interface clients (Desktop/Mobile/IoT/Container)
- **Control Plane**: REST/WebSocket API, STUN coordination, task scheduling, PostgreSQL/Redis backend
- **Data Plane**: WireGuard tunnels with direct P2P connections or relay fallback
- **Management UI**: React 19 + Ant Design 5 + ECharts SPA

**Tech Stack**:
- Backend: Go (gin, fx, gRPC)
- Frontend: React 19, TypeScript, Vite, Ant Design 5
- Infrastructure: PostgreSQL, Redis, WireGuard (userspace/kernel), STUN/TURN
- Deployment: Docker, Kubernetes, Helm

## Development Workflow - Spec-Driven Development

This project uses a **specification-driven development workflow** powered by the SpecKit slash commands. All features follow a structured lifecycle:

### Feature Lifecycle

1. **Specification** (`/speckit.specify`) - Create feature spec and branch
2. **Planning** (`/speckit.plan`) - Design implementation architecture
3. **Tasks** (`/speckit.tasks`) - Break down into actionable tasks
4. **Clarification** (`/speckit.clarify`) - Resolve ambiguities (optional)
5. **Checklists** (`/speckit.checklist`) - Generate validation checklists (optional)
6. **Implementation** (`/speckit.implement`) - Execute the task list
7. **Analysis** (`/speckit.analyze`) - Cross-artifact consistency check

### Key Commands

```bash
# Create new feature (generates branch and spec.md)
/speckit.specify <feature_description>

# Create implementation plan (generates plan.md)
/speckit.plan

# Generate task breakdown (generates tasks.md)
/speckit.tasks

# Execute implementation (reads tasks.md and implements)
/speckit.implement

# Analyze consistency across artifacts
/speckit.analyze
```

### Direct Script Usage

```bash
# Create new feature with custom short name
.specify/scripts/bash/create-new-feature.sh --json "Add OAuth2 authentication" --short-name "oauth2-auth"

# Setup plan (copies template to current feature)
.specify/scripts/bash/setup-plan.sh --json

# Check prerequisites before implementation
.specify/scripts/bash/check-prerequisites.sh --json --require-tasks --include-tasks
```

## Feature Structure

All features live in `specs/NNN-feature-name/` directories:

```
specs/
└── 001-user-auth/
    ├── spec.md              # Feature specification (REQUIRED)
    ├── plan.md              # Implementation plan (REQUIRED before coding)
    ├── tasks.md             # Task breakdown (REQUIRED for /speckit.implement)
    ├── research.md          # Technical decisions (optional)
    ├── data-model.md        # Entity relationships (optional)
    ├── quickstart.md        # Usage guide (optional)
    ├── contracts/           # API specs & test requirements (optional)
    │   └── api-v1.yaml
    └── checklists/          # Validation checklists (optional)
        ├── ux.md
        ├── security.md
        └── test.md
```

### Branch Naming Convention

- Format: `NNN-short-name` (e.g., `001-user-auth`, `002-wireguard-tunnel`)
- The numeric prefix (`NNN`) auto-increments and maps to the spec directory
- Multiple branches can share the same spec directory (e.g., `004-fix-bug` and `004-add-feature` both use `specs/004-*/`)

## Core Development Principles

### 1. Real Data Only - No Mocking
- ❌ Mock data, fake APIs, simulated responses
- ✅ Real database/API connections, actual system metrics, live container operations
- All monitoring must collect from real running systems
- All tests must validate in actual deployment environments

### 2. Complete Implementation - No Shortcuts
- ❌ Simplified error handling, skipped edge cases, hardcoded values
- ✅ Comprehensive exception handling (network failures, timeouts, data errors)
- ✅ Full edge case coverage (null values, concurrency, resource exhaustion)
- ✅ Production-grade performance optimization (caching, async, connection pooling)
- ✅ Complete security measures (authentication, authorization, SQL injection prevention)

### 3. Production-Ready Code - No Temporary Solutions
- ❌ "Get it working first" code, magic numbers, tight coupling
- ✅ SOLID principles, loose coupling, configuration management
- ✅ Multi-environment support (dev/staging/prod)
- ✅ Clear documentation and technical comments
- ✅ Extensibility for future iterations

### 4. Clean Project - No Clutter
- ❌ Feature summary documents after each development cycle
- ❌ Temporary test files, debug logs, unused config backups
- ✅ Auto-cleanup of test files, temp files, debug logs after completion
- ✅ Get explicit user permission before creating documentation
- ✅ Regular audit to remove unused files

**Files to Clean**:
- Test data files (`test_*.txt`, `temp_*.json`)
- Debug logs (`debug.log`, `trace.log`)
- Config backups (`*.bak`, `*.old`)
- Temporary scripts
- Draft documents

**DO NOT delete core documents**: `TASKS_BY_TYPE.md`, spec artifacts, plan files

## Acceptance Criteria

Before marking any feature complete, verify:
- [ ] Using real data and production environment
- [ ] Complete error handling and logging
- [ ] Performance tests passing (response time, concurrency)
- [ ] Security scan with no high-severity vulnerabilities
- [ ] Code review approved
- [ ] Documentation complete and accurate
- [ ] All temporary and test files cleaned up
- [ ] No unauthorized documentation created
- [ ] Core documents (`TASKS_BY_TYPE.md`，`specs`, etc.) intact

## Documentation Creation Rules

**Before creating ANY Markdown documentation**:
1. Ask the user explicitly if documentation is needed
2. Explain the document's purpose and content outline
3. Get explicit approval before creating

**Exceptions**: Only create directly when user explicitly requests "write docs", "generate README", "create documentation", etc.

## Project Constitution

The project follows principles defined in `.specify/memory/constitution.md` (currently uses template placeholders). When implementing features, reference this file for:
- Coding standards and patterns
- Testing requirements (e.g., TDD if specified)
- Architecture constraints
- Quality gates

## Common Patterns

### Feature Development Flow
1. User requests feature → `/speckit.specify "feature description"`
2. Review generated spec.md → `/speckit.plan` to create architecture
3. Break down work → `/speckit.tasks` to generate task list
4. (Optional) Generate validation checklists → `/speckit.checklist`
5. Execute implementation → `/speckit.implement`
6. Verify consistency → `/speckit.analyze`

### Checklist Validation (during `/speckit.implement`)
- If `checklists/` directory exists, implementation command auto-checks completion status
- **Blocks implementation** if any checklist has incomplete items
- User must explicitly approve to proceed with incomplete checklists
- All checklists must be complete (0 incomplete items) to auto-proceed

### Multiple Branches per Spec
- Can create multiple branches for the same feature number (e.g., `003-fix`, `003-enhance`)
- All branches sharing prefix `003-` work with the same `specs/003-*/` directory
- Enables parallel work on different aspects of the same feature

## Technical Notes

### Script Portability
- All bash scripts support both Git and non-Git repositories
- Use `SPECIFY_FEATURE` environment variable to override branch detection
- Scripts fall back to finding latest feature directory by numeric prefix if Git unavailable

### Path Resolution
- `common.sh` provides shared functions for all scripts
- `get_feature_paths()` returns all standard file paths for current feature
- `find_feature_dir_by_prefix()` supports multi-branch feature development

### JSON Output Mode
- All scripts support `--json` flag for programmatic consumption
- Use `--paths-only` to skip validation and only retrieve path variables

## Architecture Highlights

### Client Architecture
- **Platform Support**: Linux/Windows/macOS desktop (Go + CGO/wireguard-go), Android/iOS (WireGuard SDK), IoT/Container (lightweight CLI/daemon)
- **Configuration Flow**: Server address + pre-shared key → device keypair generation → register via `/api/v1/device/register` → receive virtual IP/subnet/peer list → local encrypted config storage
- **Runtime Modules**: Daemon (WireGuard monitoring, auto-reconnect, key rotation), NAT puncher (STUN detection, UDP hole punching, TURN fallback), telemetry (`/api/v1/device/metrics` heartbeat/bandwidth/latency)

### Control Plane Architecture
- **API Layer**: gRPC + gRPC-Gateway/REST, WebSocket for real-time push, auth via pre-shared key + device key signatures
- **Business Services**: Device management, topology/routing, NAT coordination, auditing/alerting
- **Data Layer**: PostgreSQL (entities, sessions, audit logs), Redis (online state, tokens, rate limiting), Object storage (diagnostic bundles)
- **Background Tasks**: Cleanup (inactive devices, expired keys), key rotation scheduler, NAT capability testing

### Management UI Architecture
- **Stack**: React 19 + Vite + TypeScript, Ant Design 5, React Query/SWR, WebSocket real-time updates
- **Core Pages**: Dashboard (ECharts metrics), device details, network topology, key management, alerts/audit, operations tools
- **Security**: RBAC (admin/network ops/auditor), OIDC/SAML support, optional 2FA

### Supporting Infrastructure
- **Monitoring**: Prometheus/Grafana metrics, Loki/ELK logs, Jaeger/OpenTelemetry tracing
- **Testing**: Unit/integration tests (Go test), NAT simulation (containers), E2E (Playwright), load testing (k6/Vegeta)
- **Deployment**: CI/CD via GitHub Actions, Helm charts, blue-green deployments, gradual client rollout
