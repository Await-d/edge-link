# 构建和打包需求质量检查清单

**目标**: 验证已完成服务的构建/打包需求的完整性、清晰度、一致性和可测性

**重点**: 构建可重现性 (Build Reproducibility)

**适用场景**: PR审查、CI/CD配置审查

**评估范围**: Docker镜像构建 + CI/CD管道 + 多产物管理 + 发布就绪性

---

## 1. Docker 镜像构建需求 (Docker Image Build Requirements)

### 1.1 基础镜像规范 (Base Image Specification)
- [X] **[清晰度, 可重现性]** Dockerfile中的基础镜像是否固定到具体的digest (e.g., `FROM golang:1.21.5@sha256:...`)？
- [ ] **[完整性]** 是否文档化了选择特定基础镜像版本的理由（安全性、兼容性）？
- [ ] **[一致性]** 所有微服务的Dockerfile是否使用相同版本的基础镜像（除非有特殊需求）？
- [ ] **[可测性]** 是否定义了基础镜像漏洞扫描的通过标准（e.g., 无Critical/High CVE）？

**相关文件**:
- `infrastructure/docker/Dockerfile.api-gateway`
- `infrastructure/docker/Dockerfile.device-service`
- `infrastructure/docker/Dockerfile.topology-service`
- `infrastructure/docker/Dockerfile.nat-coordinator`
- `infrastructure/docker/Dockerfile.alert-service`
- `infrastructure/docker/Dockerfile.background-worker`
- `infrastructure/docker/Dockerfile.frontend`

### 1.2 多阶段构建需求 (Multi-stage Build Requirements)
- [ ] **[清晰度]** 每个构建阶段的用途是否明确注释（e.g., `# Stage 1: Build dependencies`）？
- [ ] **[完整性]** 构建阶段依赖是否完整声明（build tools、libraries、certificates）？
- [ ] **[可重现性]** 构建工具版本是否固定（e.g., `go 1.21.5` 而非 `go:latest`）？
- [ ] **[效率]** 是否优化了层缓存策略（将频繁变更的代码放在后面层）？

### 1.3 构建参数和环境变量 (Build Arguments & Environment Variables)
- [ ] **[完整性]** 所有ARG和ENV变量是否在需求文档中列出并说明用途？
- [ ] **[清晰度]** 敏感变量（API keys、密码）是否标记为"构建时禁止硬编码"？
- [ ] **[一致性]** 跨Dockerfile的相同用途ARG是否使用统一命名（e.g., `GO_VERSION`）？
- [ ] **[可测性]** 是否定义了构建参数的有效值范围和验证方法？

**检查项示例**:
```dockerfile
# ✅ 好的实践
ARG GO_VERSION=1.21.5
ARG ALPINE_VERSION=3.19
FROM golang:${GO_VERSION}-alpine${ALPINE_VERSION}@sha256:...

# ❌ 需要改进
FROM golang:latest
```

### 1.4 镜像标签策略 (Image Tagging Strategy)
- [ ] **[完整性]** 是否定义了镜像标签规范（语义化版本、commit SHA、构建时间）？
- [ ] **[清晰度]** 生产环境部署是否明确禁止使用`:latest`标签？
- [ ] **[可重现性]** 是否要求每个镜像都包含可追溯的元数据标签（commit、build-date、version）？
- [ ] **[一致性]** 所有服务的镜像标签是否遵循相同的命名约定？

**标签规范示例**:
```
edgelink/api-gateway:v1.2.3
edgelink/api-gateway:v1.2.3-abc1234
edgelink/api-gateway:latest-dev
edgelink/api-gateway:sha-abc1234def5678
```

### 1.5 镜像安全和优化需求 (Image Security & Optimization)
- [ ] **[完整性]** 是否要求最终镜像使用非root用户运行？
- [ ] **[可测性]** 是否定义了镜像大小限制（e.g., 后端服务 < 100MB）？
- [ ] **[清晰度]** 是否文档化了必须删除的临时文件和缓存（build cache、test files）？
- [X] **[可重现性]** 是否要求容器镜像生成SBOM（Software Bill of Materials）？

---

## 2. CI/CD 管道需求 (CI/CD Pipeline Requirements)

### 2.1 构建触发条件 (Build Triggers)
- [X] **[完整性]** 是否明确定义了哪些事件触发构建（push to main、PR、tag）？
- [X] **[清晰度]** 是否区分了完整构建 vs 快速验证构建的触发条件？
- [X] **[一致性]** 所有代码仓库的触发策略是否统一（monorepo vs multi-repo）？
- [X] **[可测性]** 是否定义了构建超时限制和失败重试策略？

**相关文件**:
- `.github/workflows/backend.yml`
- `.github/workflows/frontend.yml`
- `.github/workflows/desktop-client.yml`
- `.github/workflows/release.yml`
- `.github/workflows/security-scan.yml`
- `.github/workflows/quality-gate.yml`

### 2.2 构建步骤和依赖 (Build Steps & Dependencies)
- [X] **[完整性]** 每个构建步骤的输入输出是否明确定义？
- [X] **[清晰度]** 外部依赖（npm packages、Go modules）是否锁定版本（package-lock.json、go.sum）？
- [X] **[可重现性]** 是否要求CI环境使用固定版本的构建工具（e.g., Go 1.21.5、Node 20.10.0）？
- [X] **[可测性]** 是否定义了依赖项漏洞扫描的集成点（Snyk、Dependabot）？

### 2.3 测试和质量门禁 (Testing & Quality Gates)
- [X] **[完整性]** 是否定义了哪些测试类型必须通过（unit、integration、e2e）？
- [X] **[可测性]** 代码覆盖率阈值是否明确（e.g., 后端 ≥ 80%、前端 ≥ 70%）？
- [X] **[清晰度]** Linting规则是否在CI中强制执行（golangci-lint、ESLint）？
- [X] **[一致性]** 所有服务的质量门禁标准是否一致？

### 2.4 产物管理 (Artifact Management)
- [X] **[完整性]** 是否定义了构建产物的存储位置（Docker registry、S3、GitHub Releases）？
- [X] **[清晰度]** 产物保留策略是否明确（保留期限、清理规则）？
- [X] **[可重现性]** 每个产物是否包含可验证的校验和（SHA256）？
- [X] **[可测性]** 是否定义了产物完整性验证步骤（签名验证、镜像扫描）？

### 2.5 失败处理和回滚 (Failure Handling & Rollback)
- [X] **[完整性]** 是否文档化了构建失败的通知机制（Slack、邮件、PagerDuty）？
- [X] **[清晰度]** 回滚流程是否明确定义（自动 vs 手动、回滚到哪个版本）？
- [X] **[可测性]** 是否定义了回滚验证标准（健康检查、smoke tests）？
- [X] **[例外流程覆盖]** 是否考虑了数据库迁移失败、配置错误等边缘场景的处理？

---

## 3. 多产物管理需求 (Multi-Artifact Management)

### 3.1 产物类型和规范 (Artifact Types & Specifications)
- [ ] **[完整性]** 是否列出了所有需要构建的产物类型？
  - Docker镜像（后端7个服务 + 前端）
  - 桌面客户端二进制（Linux/Windows/macOS）
  - Helm Charts
  - API文档
  - SDK（Go/Python/TypeScript）
- [ ] **[清晰度]** 每种产物的版本号方案是否定义（独立版本 vs 统一版本）？
- [ ] **[一致性]** 所有产物是否使用相同的版本号（如果适用）？

### 3.2 跨平台构建需求 (Cross-platform Build Requirements)
- [ ] **[完整性]** 桌面客户端的目标平台列表是否完整（OS、架构）？
  - Linux (amd64, arm64)
  - Windows (amd64)
  - macOS (amd64, arm64)
- [ ] **[可重现性]** 是否使用交叉编译工具或构建矩阵（GitHub Actions matrix）？
- [ ] **[可测性]** 每个平台的构建产物是否有自动化验证（smoke test）？
- [ ] **[清晰度]** 平台特定依赖是否文档化（e.g., macOS需要CGO、Windows需要Wintun）？

**相关文件**:
- `clients/desktop/internal/platform/linux.go`
- `clients/desktop/internal/platform/windows.go`
- `clients/desktop/internal/platform/macos.go`

### 3.3 Helm Chart 打包需求 (Helm Chart Packaging)
- [ ] **[完整性]** Chart.yaml是否包含完整元数据（version、appVersion、description、maintainers）？
- [ ] **[清晰度]** values.yaml是否清晰注释所有可配置参数？
- [ ] **[可测性]** 是否定义了Helm chart的验证步骤（`helm lint`、`helm template` dry-run）？
- [ ] **[版本一致性]** Chart版本与应用版本的映射关系是否明确？

**相关文件**:
- `infrastructure/helm/edge-link-control-plane/Chart.yaml`
- `infrastructure/helm/edge-link-control-plane/values.yaml`
- `infrastructure/helm/edgelink-sidecar/`

### 3.4 依赖版本锁定 (Dependency Version Locking)
- [X] **[可重现性]** Go依赖是否通过go.sum锁定？
- [X] **[可重现性]** Node.js依赖是否通过package-lock.json锁定？
- [ ] **[可重现性]** Python SDK依赖是否通过requirements.txt或poetry.lock锁定？
- [X] **[可测性]** 是否定期运行依赖更新和安全审计（Dependabot、`go mod tidy`）？

---

## 4. 发布就绪性需求 (Release Readiness Requirements)

### 4.1 版本号管理 (Version Numbering)
- [ ] **[清晰度]** 是否采用语义化版本规范（SemVer: MAJOR.MINOR.PATCH）？
- [ ] **[完整性]** 版本号生成是否自动化（git tags、自动递增）？
- [ ] **[一致性]** 所有组件的版本号是否同步更新？
- [ ] **[可测性]** 是否禁止手动修改版本号（通过自动化工具管理）？

### 4.2 变更日志 (Changelog)
- [ ] **[完整性]** 是否要求每个发布版本生成CHANGELOG.md？
- [ ] **[清晰度]** 变更日志是否按类别组织（Features、Bug Fixes、Breaking Changes）？
- [ ] **[可测性]** 是否自动从commit message生成（Conventional Commits）？
- [ ] **[可追溯性]** 每条变更是否链接到对应的PR或Issue？

### 4.3 发布检查清单 (Release Checklist)
- [ ] **[完整性]** 是否定义了发布前必须完成的验证项？
  - 所有CI测试通过
  - 安全扫描无Critical/High漏洞
  - 文档更新完成
  - 迁移脚本测试通过
  - 性能基准测试通过
- [ ] **[清晰度]** 是否区分了dev/staging/prod环境的发布要求？
- [ ] **[可测性]** 发布验证是否自动化（smoke tests、health checks）？

### 4.4 回滚和灾难恢复 (Rollback & Disaster Recovery)
- [X] **[完整性]** 是否定义了回滚触发条件（错误率阈值、性能下降）？
- [X] **[清晰度]** 回滚步骤是否文档化并可一键执行？
- [ ] **[可测性]** 是否定期演练回滚流程（chaos engineering）？
- [X] **[例外流程覆盖]** 是否考虑了数据不兼容、配置冲突等回滚失败场景？

### 4.5 监控和告警集成 (Monitoring & Alerting Integration)
- [X] **[完整性]** 新版本部署后是否自动验证监控指标（Prometheus metrics可访问）？
- [X] **[清晰度]** 发布相关的告警是否预先配置（部署失败、健康检查失败）？
- [X] **[可测性]** 是否定义了发布后的观察期（soak period）和成功标准？

---

## 5. 可重现性验证 (Build Reproducibility Validation)

### 5.1 确定性构建 (Deterministic Builds)
- [X] **[可重现性]** 相同源代码在不同时间/机器上构建是否产生相同的二进制（bit-for-bit）？
- [X] **[可测性]** 是否实现了构建哈希校验机制（对比两次构建的SHA256）？
- [X] **[清晰度]** 是否文档化了影响可重现性的因素（时间戳、随机数、并行编译）？
- [X] **[工具支持]** 是否使用了支持可重现构建的工具链（Go 1.13+、Bazel）？

### 5.2 构建环境隔离 (Build Environment Isolation)
- [ ] **[可重现性]** 是否使用容器化构建环境（Docker、GitHub Actions）以消除环境差异？
- [ ] **[清晰度]** 构建环境的规格是否完整记录（OS版本、预装软件、环境变量）？
- [ ] **[一致性]** 本地开发构建环境是否与CI环境一致？

### 5.3 依赖固定 (Dependency Pinning)
- [X] **[可重现性]** 所有直接和间接依赖的版本是否完全固定？
- [X] **[可测性]** 是否验证依赖的完整性（checksum验证、签名验证）？
- [X] **[异常处理]** 依赖源不可用时是否有备份方案（镜像仓库、vendoring）？

### 5.4 可重现性审计 (Reproducibility Audit)
- [ ] **[可测性]** 是否定期运行可重现性验证测试（每周/每次发布）？
- [ ] **[可追溯性]** 每次构建的完整环境快照是否保存（用于问题排查）？
- [ ] **[文档化]** 已知的不可重现因素是否记录并有缓解措施？

---

## 6. 文档和沟通需求 (Documentation & Communication Requirements)

### 6.1 构建文档完整性 (Build Documentation Completeness)
- [ ] **[完整性]** 是否存在完整的构建指南（本地构建、CI构建、生产构建）？
- [ ] **[清晰度]** 文档是否包含常见构建错误的排查步骤？
- [ ] **[一致性]** 文档是否与实际构建脚本保持同步？
- [ ] **[可测性]** 新成员是否能通过文档独立完成首次构建？

**相关文件**:
- `docs/deployment.md`
- `docs/troubleshooting.md`
- `README.md`
- `scripts/generate-api-docs.sh`

### 6.2 架构决策记录 (Architecture Decision Records)
- [X] **[完整性]** 关键构建/打包决策是否有ADR文档（e.g., 为何选择多阶段构建）？
- [X] **[可追溯性]** ADR是否包含决策背景、选项对比、最终选择、后果？
- [X] **[文档化]** 技术债务和已知限制是否记录？

### 6.3 构建变更通知 (Build Change Notifications)
- [ ] **[清晰度]** 影响开发者的构建变更是否提前通知（工具升级、依赖更新）？
- [ ] **[完整性]** 通知是否包含迁移指南和截止日期？
- [ ] **[沟通渠道]** 是否通过多渠道通知（Slack、邮件、CHANGELOG）？

---

## 7. 检查清单使用指南 (Checklist Usage Guide)

### 使用场景
1. **PR审查时**: 审查者验证新增/修改的构建配置是否符合需求质量标准
2. **CI/CD配置审查**: DevOps团队验证管道配置的完整性和可维护性
3. **发布前审计**: 确保所有构建需求文档完整且可执行
4. **新成员入职**: 帮助新成员理解项目的构建标准和最佳实践

### 评分标准
- **完成度计算**: (已勾选项 / 总项) × 100%
- **阻塞阈值**: 完成度 < 80% 应阻塞发布
- **优先级**: 标记为 [可重现性] 的项为P0，必须100%完成

### 持续改进
- 每个Sprint结束后审查未勾选项，制定改进计划
- 发现新的构建问题时更新检查清单
- 定期与团队同步检查清单的适用性

---

## 附录: 当前项目构建产物清单 (Appendix: Current Project Build Artifacts)

### Docker 镜像 (7个)
1. `edgelink/api-gateway` - API网关服务
2. `edgelink/device-service` - 设备管理服务（⚠️ 微服务入口点缺失）
3. `edgelink/topology-service` - 拓扑管理服务（⚠️ 微服务入口点缺失）
4. `edgelink/nat-coordinator` - NAT协调器（⚠️ 微服务入口点缺失）
5. `edgelink/alert-service` - 告警服务
6. `edgelink/background-worker` - 后台任务工作器
7. `edgelink/frontend` - 前端Web UI

### 桌面客户端二进制
- `edgelink-cli` - CLI工具（Linux/Windows/macOS）
- `edgelink-daemon` - 守护进程（Linux/Windows/macOS）
- `edgelink-lite` - 轻量级CLI（IoT设备）

### Helm Charts
- `edge-link-control-plane` - 控制平面完整部署
- `edgelink-sidecar` - 容器sidecar模式部署

### API文档和SDK
- OpenAPI/Swagger文档（通过 `scripts/generate-api-docs.sh` 生成）
- Go SDK（代码生成）
- Python SDK（代码生成）
- TypeScript SDK（代码生成）

### 已知问题
- ⚠️ **架构不一致**: tasks.md标记T058-T060为已完成，但微服务入口点实际不存在
  - 缺失: `backend/cmd/device-service/main.go`
  - 缺失: `backend/cmd/topology-service/main.go`
  - 缺失: `backend/cmd/nat-coordinator/main.go`
  - **影响**: quickstart.md假设微服务架构，但实际为单体架构
  - **建议**: 短期更新文档为单体架构，长期补全微服务实现

---

**最后更新**: 2025-10-19
**版本**: v1.0
**维护者**: EdgeLink开发团队
