<!--
Sync Impact Report - Constitution v1.1.0

Version Change: 1.0.0 → 1.1.0 (Expanded principle guidance and verification standards)

Principles Modified:
- I. Real Data Only - No Mocking (EXPANDED: Added more detailed prohibited/required items)
- II. Complete Implementation - No Shortcuts (EXPANDED: Added comprehensive verification standards)
- III. Production-Ready Code - No Temporary Solutions (EXPANDED: Added detailed quality requirements)
- IV. Clean Project - No Clutter (EXPANDED: Added file cleanup specifications and protected files list)

Sections Modified:
- Acceptance Criteria (EXPANDED: More detailed checklist items aligned with updated principles)

Templates Requiring Updates:
- ✅ .specify/templates/plan-template.md - Constitution Check section already aligned
- ✅ .specify/templates/spec-template.md - Edge case requirements already aligned
- ✅ .specify/templates/tasks-template.md - Already aligned with incremental delivery
- ✅ CLAUDE.md - Runtime guidance already contains detailed source principles

Follow-up TODOs: None - all principles expanded with concrete verification criteria
-->

# Edge-Link Constitution

## Core Principles

### I. Real Data Only - No Mocking

**NON-NEGOTIABLE**: All development, testing, and monitoring MUST use real data and actual system connections.

**禁止行为 (Prohibited)**:
- ❌ 使用Mock数据、假数据、示例数据 (Mock data, fake data, sample data)
- ❌ 使用模拟的API响应或假接口 (Simulated API responses or fake interfaces)
- ❌ 使用虚拟环境替代真实系统 (Virtual environments replacing real systems)

**必须做到 (Required)**:
- ✅ 所有数据源必须连接真实的数据库/API (All data sources must connect to real databases/APIs)
- ✅ 所有监控指标必须采集自实际运行的系统 (All monitoring metrics must be collected from actually running systems)
- ✅ 所有容器操作必须在真实环境中执行 (All container operations must execute in real environments - docker exec)
- ✅ 所有测试必须在实际部署环境中验证 (All tests must validate in actual deployment environments)

**Rationale**: Edge-Link is a distributed network infrastructure system where timing, concurrency, network failures, and resource constraints are core concerns. Mocked data cannot accurately represent the real-world behavior of NAT traversal, packet loss, connection timeouts, or multi-peer coordination. Only real data reveals production issues.

### II. Complete Implementation - No Shortcuts

**NON-NEGOTIABLE**: Every feature MUST be implemented with production-grade completeness from the start.

**禁止行为 (Prohibited)**:
- ❌ 省略错误处理逻辑 (Omitting error handling logic - try-catch, exception capture)
- ❌ 跳过边界条件和异常场景的处理 (Skipping boundary conditions and exceptional scenarios)
- ❌ 忽略性能优化 (Ignoring performance optimization - caching, batching, connection pooling)
- ❌ 简化安全措施 (Simplifying security measures - authentication, authorization, data validation)

**必须做到 (Required)**:
- ✅ 实现全面的异常处理机制 (Implement comprehensive exception handling - network failures, timeouts, data errors)
- ✅ 覆盖所有边缘情况 (Cover all edge cases - null values, concurrency, resource exhaustion)
- ✅ 实施完整的性能优化策略 (Implement complete performance optimization - caching, async, resource reuse)
- ✅ 部署完善的安全防护体系 (Deploy comprehensive security protection - permission checks, SQL injection prevention, XSS protection)

**Rationale**: Network infrastructure systems fail catastrophically when edge cases are ignored. A single unhandled timeout can cascade into connection storms. A missing null check can crash the control plane. Security shortcuts expose the entire network to compromise. There are no "minor" omissions in distributed systems. A single unhandled timeout can cascade into connection storms. A missing null check can crash the control plane. Security shortcuts expose the entire network to compromise. There are no "minor" omissions in distributed systems.

### III. Production-Ready Code - No Temporary Solutions

**NON-NEGOTIABLE**: All code MUST be production-ready on first implementation. No placeholders, no "refactor later" code.

**禁止行为 (Prohibited)**:
- ❌ 编写"先跑起来再说"的临时代码 (Writing "get it working first" temporary code)
- ❌ 使用硬编码、魔法数字、全局变量 (Using hardcoded values, magic numbers, global variables)
- ❌ 创建紧耦合、不可扩展的架构 (Creating tightly coupled, non-extensible architecture)
- ❌ 忽略代码规范和文档注释 (Ignoring code standards and documentation comments)

**必须做到 (Required)**:
- ✅ 所有代码达到生产环境部署标准 (All code meets production deployment standards)
- ✅ 遵循SOLID原则，设计松耦合的架构 (Follow SOLID principles, design loosely coupled architecture)
- ✅ 使用配置管理，支持多环境部署 (Use configuration management, support multi-environment deployment)
- ✅ 编写清晰的注释和完整的技术文档 (Write clear comments and complete technical documentation)
- ✅ 预留扩展点，支持未来功能迭代 (Reserve extension points, support future iterations)
- ✅ 通过Code Review和自动化测试验证质量 (Verify quality through Code Review and automated testing)

**Rationale**: Edge-Link will scale to thousands of devices across heterogeneous network environments. Technical debt compounds exponentially in distributed systems. Code that is "good enough for now" becomes the foundation for brittle, unmaintainable systems. Refactoring distributed protocols in production is dangerous and expensive.

### IV. Clean Project - No Clutter

**NON-NEGOTIABLE**: The codebase MUST remain clean and focused. Temporary files, debug artifacts, and unauthorized documentation are prohibited.

**禁止行为 (Prohibited)**:
- ❌ 每次开发后生成冗余的功能总结文档 (Creating redundant feature summary documents after each development cycle)
- ❌ 保留临时测试文件和调试文件 (Keeping temporary test files and debug files)
- ❌ 未经用户明确同意创建Markdown文档 (Creating Markdown documents without explicit user consent)
- ❌ 在代码库中积累不必要的文件 (Accumulating unnecessary files in the codebase)

**必须做到 (Required)**:
- ✅ 开发过程中专注代码实现，避免过度文档化 (Focus on code implementation during development, avoid over-documentation)
- ✅ 功能完成后自动清理测试文件、临时文件、调试日志 (Auto-cleanup test files, temp files, debug logs after task completion)
- ✅ 创建任何文档前必须获得用户明确授权 (Must obtain explicit user authorization before creating any documentation)
- ✅ 定期审查和清理无用文件，保持项目结构清晰 (Regular audit and cleanup of unused files, maintain clear project structure)
- ✅ 仅在必要时（如API文档、部署说明）创建文档 (Only create documentation when necessary - e.g., API docs, deployment guides)

**需要清理的文件类型 (Files Requiring Cleanup)**:
- 🗑️ 测试用的临时数据文件 (Test data files: test_*.txt, temp_*.json, etc.)
- 🗑️ 调试日志文件 (Debug logs: debug.log, trace.log, etc.)
- 🗑️ 未使用的配置文件副本 (Unused config backups: *.bak, *.old, etc.)
- 🗑️ 代码生成的临时脚本文件 (Temporary scripts generated during development)
- 🗑️ 开发过程中的草稿文档 (Draft documents created during development)

**受保护的文件 (Protected Files - DO NOT Delete)**:
- 核心文档 (Core documentation): TASKS_BY_TYPE.md, spec artifacts (spec.md, plan.md, tasks.md, research.md, data-model.md, quickstart.md)
- 配置文件 (Configuration): CLAUDE.md, .specify/ directory contents
- 源代码和测试 (Source code and tests)

**Rationale**: Clutter obscures the essential. In a complex distributed system, developers must quickly locate critical code, configs, and documentation. Stale files create confusion, slow onboarding, and hide real issues. A clean codebase is a maintainable codebase.

### V. Distributed Systems Resilience

**NON-NEGOTIABLE**: All components MUST be designed for failure, eventual consistency, and network partitions.

- ✅ Failure handling: graceful degradation, circuit breakers, retry with exponential backoff, fallback mechanisms
- ✅ Idempotency: all state-changing operations MUST be safely retryable
- ✅ Timeout policies: every network operation MUST have explicit timeouts (connection, read, write)
- ✅ Observability: structured logging, distributed tracing, metrics collection, health checks
- ✅ State management: handle split-brain scenarios, conflict resolution, eventual consistency
- ❌ PROHIBITED: Assumptions of perfect network reliability, unbounded retry loops, missing timeout handling

**Rationale**: Edge-Link operates across unreliable networks (NAT, firewalls, mobile connections). Peers disconnect unexpectedly. Control plane services restart. STUN servers become unavailable. The system MUST remain operational despite partial failures. CAP theorem is not optional—it's the reality we build for.

## Architecture Constraints

### Technology Stack (NON-NEGOTIABLE)

**Backend**:
- Language: Go 1.21+ (required for WireGuard userspace, performance, concurrency)
- Framework: Gin (HTTP), gRPC (service-to-service), Fx (dependency injection)
- Database: PostgreSQL 14+ (entities, sessions, audit logs)
- Cache/State: Redis 7+ (online state, tokens, rate limiting)
- Storage: S3-compatible object storage (diagnostic bundles, logs)

**Frontend**:
- Language: TypeScript 5+ (strict mode required)
- Framework: React 19 + Vite
- UI Library: Ant Design 5 (consistent with design system)
- Charts: ECharts (network topology, metrics visualization)
- State Management: React Query / SWR (server state), Zustand (client state)

**Infrastructure**:
- Networking: WireGuard (kernel module or wireguard-go userspace), STUN/TURN servers
- Orchestration: Docker, Kubernetes, Helm charts
- Monitoring: Prometheus, Grafana, Loki/ELK, Jaeger/OpenTelemetry

### System Architecture (NON-NEGOTIABLE)

**Three-Layer Model**:
1. **Client Layer**: TUN/virtual network interfaces (Desktop/Mobile/IoT/Container)
   - Platform support: Linux/Windows/macOS (Go + CGO), Android/iOS (WireGuard SDK), IoT (lightweight CLI)
   - Configuration: server address + pre-shared key → keypair generation → registration → virtual IP assignment
   - Runtime: daemon (monitoring, auto-reconnect, key rotation), NAT puncher (STUN, UDP hole punching, TURN fallback)

2. **Control Plane**: REST/WebSocket API, STUN coordination, task scheduling
   - API: gRPC + gRPC-Gateway/REST, WebSocket for real-time push
   - Services: device management, topology/routing, NAT coordination, auditing/alerting
   - Data: PostgreSQL (persistent state), Redis (session state), object storage (diagnostics)

3. **Data Plane**: WireGuard tunnels with P2P direct connections or relay fallback
   - Direct P2P preferred (STUN-assisted NAT traversal)
   - TURN relay fallback for symmetric NAT
   - Encrypted end-to-end (WireGuard cryptokey routing)

**Communication Flow**:
1. Client startup → input server address + pre-shared key
2. Generate device keypair → register via `/api/v1/device/register`
3. Receive virtual IP, subnet, peer list → store encrypted local config
4. NAT detection (STUN) → hole punching → tunnel establishment
5. Periodic heartbeat + metrics reporting → control plane monitoring
6. Alerting on anomalies (connection failures, latency spikes, key expiration)

### Performance Requirements

- **Latency**: Control plane API p95 < 200ms, peer discovery < 500ms
- **Throughput**: 10,000+ concurrent device connections per control plane instance
- **Availability**: 99.9% uptime for control plane, graceful degradation for data plane
- **Scalability**: Horizontal scaling for control plane services, stateless API design

### Security Requirements (NON-NEGOTIABLE)

- **Authentication**: Pre-shared key (initial registration) + device keypair (ongoing)
- **Authorization**: RBAC (admin/network-ops/auditor), API key scoping
- **Encryption**: WireGuard tunnels (ChaCha20-Poly1305), TLS 1.3 for control plane APIs
- **Key Management**: Automatic key rotation, secure storage (encrypted at rest), audit logging
- **Input Validation**: All API inputs validated, SQL injection prevention, rate limiting

## Development Workflow

### Spec-Driven Development (NON-NEGOTIABLE)

All features MUST follow the structured SpecKit lifecycle:

1. **Specification** (`/speckit.specify`) - Create feature spec.md and feature branch
2. **Planning** (`/speckit.plan`) - Design implementation architecture (plan.md, research.md, data-model.md)
3. **Tasks** (`/speckit.tasks`) - Break down into actionable, prioritized tasks (tasks.md)
4. **Clarification** (`/speckit.clarify`) - Resolve ambiguities (optional, update spec.md)
5. **Checklists** (`/speckit.checklist`) - Generate validation checklists (optional)
6. **Implementation** (`/speckit.implement`) - Execute task list with checklist validation
7. **Analysis** (`/speckit.analyze`) - Cross-artifact consistency check

### Feature Structure (REQUIRED)

```
specs/
└── NNN-feature-name/
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

### Branching Convention

- Format: `NNN-short-name` (e.g., `001-user-auth`, `002-wireguard-tunnel`)
- Numeric prefix auto-increments and maps to spec directory
- Multiple branches may share the same spec directory (e.g., `004-fix-bug`, `004-add-feature` → `specs/004-*/`)

### Acceptance Criteria (ALL features)

Before marking any feature complete, verify:
- [ ] **使用真实数据和环境运行** (Using real data and production environment - Principle I)
- [ ] **完整的错误处理和日志记录** (Complete error handling and logging - Principle II)
- [ ] **性能测试达标**（响应时间、并发量）(Performance tests passing: response time, concurrency - Principle II)
- [ ] **安全扫描无高危漏洞** (Security scan with no high-severity vulnerabilities - Principle II)
- [ ] **代码审查通过** (Code review approved - Principle III)
- [ ] **文档完整且准确** (Documentation complete and accurate - Principle III)
- [ ] **已清理所有临时和测试文件** (All temporary and test files cleaned up - Principle IV)
- [ ] **未创建未经授权的文档** (No unauthorized documentation created - Principle IV)
- [ ] **核心文档未被误删** (Core documents like TASKS_BY_TYPE.md, spec artifacts intact - Principle IV)
- [ ] **分布式系统弹性已验证** (Distributed system resilience verified - Principle V)

### Documentation Creation Policy

**在创建任何Markdown文档前，必须** (Before creating ANY Markdown documentation):
1. **明确询问用户是否需要创建文档** (Explicitly ask the user if documentation is needed)
2. **说明文档的目的和内容概要** (Explain the document's purpose and content outline)
3. **获得用户明确同意后再创建** (Get explicit approval before creating)

**例外情况** (Exceptions): 仅当用户明确要求"写文档"、"生成说明"、"创建README"等时，才可直接创建 (Only create directly when user explicitly requests "write docs", "generate README", "create documentation", etc.)

## Governance

### Authority

This constitution supersedes all other development practices, coding guidelines, and team conventions. In case of conflict, the constitution takes precedence.

### Amendment Process

1. **Proposal**: Amendments MUST be proposed with rationale and impact analysis
2. **Review**: Technical review by project maintainers required
3. **Approval**: Explicit approval required (no implicit acceptance)
4. **Migration Plan**: For breaking changes, MUST include migration path for existing code
5. **Version Bump**: Follow semantic versioning (MAJOR.MINOR.PATCH)
   - MAJOR: Backward-incompatible governance/principle removals or redefinitions
   - MINOR: New principle/section added or materially expanded guidance
   - PATCH: Clarifications, wording, typo fixes, non-semantic refinements

### Compliance Verification

- All PRs MUST verify compliance with these principles before merge
- Code reviews MUST explicitly check for violations
- Complexity MUST be justified against Principle III (Production-Ready Code)
- CI/CD pipelines MUST enforce testing requirements (Principle II)

### Runtime Guidance

For operational development guidance, refer to `CLAUDE.md` in the repository root. That file provides tactical workflows, common patterns, and tool usage that implement these constitutional principles.

**Version**: 1.1.0 | **Ratified**: 2025-10-19 | **Last Amended**: 2025-10-19
