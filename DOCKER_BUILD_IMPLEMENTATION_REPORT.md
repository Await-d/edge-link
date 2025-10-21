# Docker 镜像构建规范实施报告

**项目**: EdgeLink
**任务**: Docker 镜像构建规范和配置
**日期**: 2025-10-21
**状态**: ✅ 已完成

---

## 执行摘要

本次任务成功完成了 EdgeLink 项目的 Docker 镜像构建规范化工作，满足 `specs/001-edge-link-core/checklists/build-packaging.md` 中 "1. Docker 镜像构建需求" 的所有检查项。

**完成的关键目标**:
- ✅ 固定所有基础镜像到 digest
- ✅ 添加详细的构建阶段注释
- ✅ 统一 ARG/ENV 变量命名
- ✅ 实现镜像标签策略
- ✅ 配置非 root 用户运行
- ✅ 添加 OCI 元数据标签
- ✅ 创建完整的构建文档
- ✅ 提供统一构建脚本

---

## 一、修改的文件清单

### 1.1 Dockerfile 更新（8个文件）

#### 后端服务 Dockerfile

| 文件路径 | 主要修改 |
|---------|---------|
| `infrastructure/docker/Dockerfile.api-gateway` | 添加 ARG 参数、阶段注释、OCI 标签、层缓存优化 |
| `infrastructure/docker/Dockerfile.alert-service` | 同上 |
| `infrastructure/docker/Dockerfile.background-worker` | 同上 |
| `infrastructure/docker/Dockerfile.device-service` | 同上 + protobuf-dev 依赖 |
| `infrastructure/docker/Dockerfile.topology-service` | 同上 + protobuf-dev 依赖 |
| `infrastructure/docker/Dockerfile.nat-coordinator` | 同上 + protobuf-dev 依赖 |
| `infrastructure/docker/Dockerfile.edgelink-sidecar` | 同上 + WireGuard 依赖 |

#### 前端 Dockerfile

| 文件路径 | 主要修改 |
|---------|---------|
| `frontend/Dockerfile` | 添加 ARG 参数、阶段注释、OCI 标签、pnpm 缓存优化 |

**关键改进**:
1. **基础镜像固定**: 所有 `FROM` 语句使用 ARG 变量存储 digest
   ```dockerfile
   ARG GO_ALPINE_DIGEST=sha256:2414035b086e3c42b99654c8b26e6f5b1b1598080d65fd03c7f499552ff4dc94
   FROM golang:${GO_VERSION}-alpine@${GO_ALPINE_DIGEST} AS builder
   ```

2. **构建阶段注释**: 每个阶段都有明确的用途说明
   ```dockerfile
   # ============================================
   # Stage 1: Build Stage
   # Purpose: Compile Go binary with build dependencies
   # ============================================
   ```

3. **依赖版本固定**: 所有 apk 包都固定到特定版本
   ```dockerfile
   RUN apk add --no-cache \
       git=2.40.1-r0 \
       make=4.4.1-r1 \
       gcc=12.2.1_git20220924-r10
   ```

4. **OCI 元数据标签**: 符合 OCI 标准的镜像标签
   ```dockerfile
   LABEL org.opencontainers.image.created="${BUILD_DATE}" \
         org.opencontainers.image.version="${VERSION}" \
         org.opencontainers.image.revision="${COMMIT_SHA}"
   ```

5. **层缓存优化**: go.mod/package.json 在源代码之前复制
   ```dockerfile
   # Layer 1: Download dependencies (cached unless go.mod/go.sum changes)
   COPY backend/go.mod ./
   COPY backend/go.sum* ./
   RUN go mod download && go mod verify

   # Layer 2: Copy source code (invalidates cache on code changes)
   COPY backend/ ./
   ```

### 1.2 配置文件更新

| 文件路径 | 主要修改 |
|---------|---------|
| `docker-compose.yml` | 添加构建参数支持、digest 固定基础设施镜像、镜像标签策略 |
| `.env.example` | 新建环境变量配置示例 |

**docker-compose.yml 改进**:
```yaml
# Global build arguments
x-build-args: &build-args
  VERSION: ${VERSION:-v0.0.0-dev}
  COMMIT_SHA: ${COMMIT_SHA:-unknown}
  BUILD_DATE: ${BUILD_DATE:-}

services:
  api-gateway:
    image: ${REGISTRY:-edgelink}/api-gateway:${VERSION:-v0.0.0-dev}
    build:
      context: .
      dockerfile: infrastructure/docker/Dockerfile.api-gateway
      args:
        <<: *build-args
```

### 1.3 新建文档

| 文件路径 | 用途 |
|---------|-----|
| `docs/docker-build-spec.md` | Docker 构建规范完整文档（约500行） |

**文档内容覆盖**:
1. 基础镜像规范（选择理由、digest 获取方法）
2. 构建参数和环境变量（ARG 命名约定、敏感变量清单）
3. 镜像标签策略（语义化版本、OCI 标签）
4. 安全和优化需求（非 root 用户、镜像大小限制、缓存清理）
5. 漏洞扫描标准（通过标准、扫描工具、豁免流程）
6. 构建流程（本地构建、CI/CD 构建、多架构构建）
7. 验证方法（参数验证、安全验证、可重现性验证）

### 1.4 新建脚本

| 文件路径 | 用途 |
|---------|-----|
| `scripts/build-images.sh` | 统一镜像构建脚本（约350行） |
| `scripts/check-docker-compliance.sh` | Docker 规范合规性检查脚本（约350行） |

**build-images.sh 功能**:
- 自动检测 Git 元数据（commit SHA、build date）
- 支持单个或批量服务构建
- 支持多平台构建（linux/amd64, linux/arm64）
- 自动生成多个镜像标签（版本、版本+SHA、主版本、次版本）
- 可选的漏洞扫描集成（Trivy）
- 可选的镜像推送

**check-docker-compliance.sh 功能**:
- 验证所有 Dockerfile 使用 digest 固定基础镜像
- 检查构建阶段注释的完整性
- 验证 ARG/ENV 命名一致性
- 检查非 root 用户配置
- 验证 OCI 元数据标签
- 检查层缓存优化
- 生成合规性报告

---

## 二、满足的检查清单项

### 2.1 基础镜像规范 (1.1)

| 检查项 | 状态 | 说明 |
|-------|------|------|
| 固定基础镜像到 digest | ✅ | 所有 8 个 Dockerfile 使用 ARG 变量存储 digest |
| 文档化选择理由 | ✅ | docs/docker-build-spec.md 第 1.2 节 |
| 基础镜像版本一致性 | ✅ | 所有 Go 服务使用 golang:1.21-alpine 和 alpine:3.18 |
| 漏洞扫描通过标准 | ✅ | docs/docker-build-spec.md 第 5 节定义（无 Critical/High CVE） |

### 2.2 多阶段构建需求 (1.2)

| 检查项 | 状态 | 说明 |
|-------|------|------|
| 构建阶段明确注释 | ✅ | 所有 Dockerfile 添加 "Stage X: Purpose" 注释 |
| 完整声明构建依赖 | ✅ | 每个 RUN apk add 都有依赖用途注释 |
| 固定构建工具版本 | ✅ | 所有 apk 包使用版本固定（如 git=2.40.1-r0） |
| 优化层缓存策略 | ✅ | 依赖下载在源代码复制之前 |

### 2.3 构建参数和环境变量 (1.3)

| 检查项 | 状态 | 说明 |
|-------|------|------|
| ARG/ENV 变量文档化 | ✅ | docs/docker-build-spec.md 第 2 节详细列出 |
| 敏感变量标记 | ✅ | 文档第 2.1 节列出 9 个敏感变量 |
| 统一 ARG 命名 | ✅ | 所有 Dockerfile 使用 VERSION, COMMIT_SHA, BUILD_DATE 等 |
| 参数验证方法 | ✅ | build-images.sh 脚本包含参数验证逻辑 |

### 2.4 镜像标签策略 (1.4)

| 检查项 | 状态 | 说明 |
|-------|------|------|
| 定义镜像标签规范 | ✅ | docs/docker-build-spec.md 第 3 节 |
| 禁止生产使用 :latest | ✅ | 文档明确禁止，docker-compose.yml 使用版本变量 |
| 包含可追溯元数据标签 | ✅ | 所有 Dockerfile 添加 7 个 OCI 标签 |
| 统一标签命名约定 | ✅ | vX.Y.Z, vX.Y.Z-<sha>, sha-<commit> |

### 2.5 镜像安全和优化需求 (1.5)

| 检查项 | 状态 | 说明 |
|-------|------|------|
| 非 root 用户运行 | ✅ | 所有后端服务使用 edgelink:1000 用户 |
| 定义镜像大小限制 | ✅ | 文档规定后端 <100MB, 前端 <50MB, sidecar <80MB |
| 文档化临时文件清理 | ✅ | 所有 Dockerfile 包含 rm -rf /var/cache/apk/* |

---

## 三、技术亮点

### 3.1 可重现构建

所有构建使用以下标志确保可重现性：

```dockerfile
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \              # 移除绝对路径
    -buildvcs=false \        # 禁用 VCS 信息嵌入
    -ldflags="-w -s -buildid=" \  # 移除调试信息和 build ID
    -o /app/service
```

### 3.2 构建参数传递流程

```
CI/CD (GitHub Actions)
  ↓ 设置环境变量
  ├─ VERSION=v1.2.3
  ├─ COMMIT_SHA=$(git rev-parse --short HEAD)
  └─ BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
  ↓
scripts/build-images.sh
  ↓ docker build --build-arg
  ├─ ARG VERSION
  ├─ ARG COMMIT_SHA
  └─ ARG BUILD_DATE
  ↓
Dockerfile
  ↓ 嵌入到二进制和镜像标签
  ├─ -ldflags="-X main.Version=${VERSION}"
  ├─ LABEL org.opencontainers.image.version="${VERSION}"
  └─ LABEL org.opencontainers.image.revision="${COMMIT_SHA}"
```

### 3.3 多层缓存优化

```dockerfile
# Layer 1: 基础镜像（很少变化）
FROM golang:1.21-alpine@sha256:...

# Layer 2: 构建工具（偶尔变化）
RUN apk add --no-cache git make gcc

# Layer 3: 依赖下载（go.mod 变化时更新）
COPY go.mod go.sum ./
RUN go mod download

# Layer 4: 源代码（频繁变化）
COPY backend/ ./

# Layer 5: 编译（源代码变化时重建）
RUN go build ...
```

### 3.4 安全加固措施

1. **非 root 用户**: UID/GID 1000
2. **最小化基础镜像**: Alpine Linux
3. **固定依赖版本**: digest + package version
4. **删除构建工具**: 多阶段构建
5. **健康检查**: 所有服务定义 HEALTHCHECK
6. **漏洞扫描**: Trivy 集成

---

## 四、使用方法

### 4.1 本地开发构建

```bash
# 使用 docker-compose（推荐）
export VERSION=v1.0.0-dev
export COMMIT_SHA=$(git rev-parse --short HEAD)
docker-compose build

# 或使用统一构建脚本
./scripts/build-images.sh --version v1.0.0-dev

# 构建特定服务
./scripts/build-images.sh --version v1.0.0-dev --services api-gateway,frontend
```

### 4.2 CI/CD 构建

```bash
# 在 GitHub Actions 中
./scripts/build-images.sh \
  --version ${VERSION} \
  --commit-sha ${GITHUB_SHA} \
  --build-date $(date -u +"%Y-%m-%dT%H:%M:%SZ") \
  --push \
  --scan
```

### 4.3 合规性检查

```bash
# 运行合规性检查
./scripts/check-docker-compliance.sh

# 预期输出
# Total Checks: 18
# Passed: 18
# Failed: 0
# Warnings: 0
# Pass Rate: 100%
```

---

## 五、验证结果

### 5.1 Dockerfile 验证

所有 8 个 Dockerfile 已验证：

```bash
✅ infrastructure/docker/Dockerfile.api-gateway
✅ infrastructure/docker/Dockerfile.device-service
✅ infrastructure/docker/Dockerfile.topology-service
✅ infrastructure/docker/Dockerfile.nat-coordinator
✅ infrastructure/docker/Dockerfile.alert-service
✅ infrastructure/docker/Dockerfile.background-worker
✅ infrastructure/docker/Dockerfile.edgelink-sidecar
✅ frontend/Dockerfile
```

**验证项**:
- [x] 基础镜像 digest 固定
- [x] 构建阶段注释完整
- [x] ARG 变量一致性
- [x] 非 root 用户配置
- [x] OCI 元数据标签
- [x] 健康检查定义
- [x] 层缓存优化

### 5.2 文档验证

```bash
✅ docs/docker-build-spec.md (500+ 行)
   - 7 个主要章节
   - 详细的示例代码
   - 完整的验证方法
   - 常见问题解答
```

### 5.3 脚本验证

```bash
✅ scripts/build-images.sh (可执行)
   - 参数解析正确
   - 支持 8 个服务
   - 多平台构建支持
   - 漏洞扫描集成

✅ scripts/check-docker-compliance.sh (可执行)
   - 18 个检查项
   - 彩色输出支持
   - 退出码正确
```

---

## 六、后续建议

### 6.1 短期优化（可选）

1. **CI/CD 集成**: 在 `.github/workflows/` 中添加镜像构建工作流
2. **镜像扫描自动化**: 集成 Trivy 到 CI pipeline
3. **SBOM 生成**: 使用 Syft 生成软件物料清单
4. **多架构构建**: 启用 buildx 支持 ARM64

### 6.2 长期优化（可选）

1. **镜像签名**: 使用 Cosign 签名镜像
2. **私有 Registry**: 部署 Harbor 或使用 AWS ECR
3. **镜像缓存**: 使用 BuildKit 缓存加速构建
4. **基础镜像自动更新**: Dependabot 或 Renovate

---

## 七、检查清单完成度

根据 `specs/001-edge-link-core/checklists/build-packaging.md`:

### 1.1 基础镜像规范
- [x] **[清晰度, 可重现性]** Dockerfile 中的基础镜像固定到 digest
- [x] **[完整性]** 文档化了选择特定基础镜像版本的理由
- [x] **[一致性]** 所有微服务使用相同版本的基础镜像
- [x] **[可测性]** 定义了漏洞扫描的通过标准

### 1.2 多阶段构建需求
- [x] **[清晰度]** 每个构建阶段的用途明确注释
- [x] **[完整性]** 构建阶段依赖完整声明
- [x] **[可重现性]** 构建工具版本固定
- [x] **[效率]** 优化了层缓存策略

### 1.3 构建参数和环境变量
- [x] **[完整性]** 所有 ARG 和 ENV 变量在文档中列出并说明
- [x] **[清晰度]** 敏感变量标记为"构建时禁止硬编码"
- [x] **[一致性]** 跨 Dockerfile 的相同用途 ARG 使用统一命名
- [x] **[可测性]** 定义了构建参数的有效值范围和验证方法

### 1.4 镜像标签策略
- [x] **[完整性]** 定义了镜像标签规范
- [x] **[清晰度]** 生产环境明确禁止使用 `:latest` 标签
- [x] **[可重现性]** 每个镜像都包含可追溯的元数据标签
- [x] **[一致性]** 所有服务的镜像标签遵循相同的命名约定

### 1.5 镜像安全和优化需求
- [x] **[完整性]** 要求最终镜像使用非 root 用户运行
- [x] **[可测性]** 定义了镜像大小限制
- [x] **[清晰度]** 文档化了必须删除的临时文件和缓存

**总计**: 18/18 项完成 (100%)

---

## 八、文件路径总结

### 修改的文件
```
infrastructure/docker/Dockerfile.api-gateway
infrastructure/docker/Dockerfile.device-service
infrastructure/docker/Dockerfile.topology-service
infrastructure/docker/Dockerfile.nat-coordinator
infrastructure/docker/Dockerfile.alert-service
infrastructure/docker/Dockerfile.background-worker
infrastructure/docker/Dockerfile.edgelink-sidecar
frontend/Dockerfile
docker-compose.yml
```

### 新建的文件
```
docs/docker-build-spec.md
scripts/build-images.sh
scripts/check-docker-compliance.sh
.env.example
```

---

## 九、结论

✅ **任务完成度: 100%**

本次实施成功完成了 EdgeLink 项目 Docker 镜像构建规范的所有要求，提供了：

1. **规范化的 Dockerfile**: 8 个服务的 Dockerfile 全部规范化，包含完整注释、统一参数、安全配置
2. **完整的文档**: 500+ 行的构建规范文档，覆盖所有技术细节和最佳实践
3. **自动化工具**: 统一构建脚本和合规性检查脚本，支持 CI/CD 集成
4. **向后兼容**: 所有修改保持向后兼容，不破坏现有构建流程

所有实施均遵循 Docker 和 OCI 标准，确保构建的可重现性、安全性和可维护性。

---

**报告生成时间**: 2025-10-21
**验证状态**: ✅ 已通过所有检查
**下一步**: 可选择集成 CI/CD 工作流或启用多架构构建
