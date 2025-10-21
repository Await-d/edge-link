# EdgeLink Docker 镜像构建规范

**版本**: v1.0
**最后更新**: 2025-10-21
**维护者**: EdgeLink 开发团队

本文档定义 EdgeLink 项目的 Docker 镜像构建标准、安全要求和最佳实践。

---

## 目录

- [1. 基础镜像规范](#1-基础镜像规范)
- [2. 构建参数和环境变量](#2-构建参数和环境变量)
- [3. 镜像标签策略](#3-镜像标签策略)
- [4. 安全和优化需求](#4-安全和优化需求)
- [5. 漏洞扫描标准](#5-漏洞扫描标准)
- [6. 构建流程](#6-构建流程)
- [7. 验证方法](#7-验证方法)

---

## 1. 基础镜像规范

### 1.1 固定镜像 Digest

所有 Dockerfile 必须将基础镜像固定到具体的 SHA256 digest，而不是使用版本标签。

**理由**:
- **安全性**: 防止供应链攻击，确保使用的镜像未被篡改
- **可重现性**: 保证不同时间、不同环境的构建结果一致
- **可追溯性**: 明确知道使用的确切镜像版本

**示例**:
```dockerfile
# ✅ 正确：使用 digest 固定
ARG GO_VERSION=1.21
ARG GO_ALPINE_DIGEST=sha256:2414035b086e3c42b99654c8b26e6f5b1b1598080d65fd03c7f499552ff4dc94
FROM golang:${GO_VERSION}-alpine@${GO_ALPINE_DIGEST}

# ❌ 错误：仅使用标签
FROM golang:1.21-alpine

# ❌ 错误：使用 latest
FROM golang:latest
```

### 1.2 基础镜像选择标准

| 服务类型 | 基础镜像 | Digest | 选择理由 |
|---------|---------|--------|---------|
| Go 后端服务 | golang:1.21-alpine | sha256:2414035b... | 官方镜像，体积小，安全更新及时 |
| Go 运行时 | alpine:3.18 | sha256:de0eb0b3... | 最小化运行时，安全漏洞少 |
| Node.js 构建 | node:20-alpine | sha256:1ab6fc5a... | 官方镜像，包含 corepack 支持 |
| Nginx 服务 | nginx:1.25-alpine | sha256:516475cc... | 官方镜像，性能优秀，轻量级 |

**获取 Digest 方法**:
```bash
# 方法 1: 使用 docker pull
docker pull golang:1.21-alpine
docker inspect golang:1.21-alpine | jq -r '.[0].RepoDigests[0]'

# 方法 2: 使用 crane (推荐)
crane digest golang:1.21-alpine

# 方法 3: 使用 skopeo
skopeo inspect docker://golang:1.21-alpine | jq -r '.Digest'
```

### 1.3 基础镜像版本一致性

所有后端服务必须使用相同版本的基础镜像，除非有特殊技术需求。

**当前标准版本**:
- Go 构建镜像: `golang:1.21-alpine`
- Go 运行时镜像: `alpine:3.18`
- Node.js 构建镜像: `node:20-alpine`
- Nginx 运行时镜像: `nginx:1.25-alpine`

---

## 2. 构建参数和环境变量

### 2.1 标准 ARG 变量

所有 Dockerfile 必须声明以下 ARG 变量：

#### 构建阶段 ARG (Build Stage)

```dockerfile
# 基础镜像版本控制
ARG GO_VERSION=1.21              # Go 版本
ARG ALPINE_VERSION=3.18          # Alpine 版本
ARG NODE_VERSION=20              # Node.js 版本（前端）
ARG NGINX_VERSION=1.25           # Nginx 版本（前端）

# 镜像 Digest
ARG GO_ALPINE_DIGEST=sha256:2414035b086e3c42b99654c8b26e6f5b1b1598080d65fd03c7f499552ff4dc94
ARG ALPINE_DIGEST=sha256:de0eb0b3f2a47ba1eb89389859a9bd88b28e82f5826b6969ad604979713c2d4f

# 构建元数据（由 CI/CD 传入）
ARG BUILD_DATE                   # 构建时间 (RFC 3339)
ARG VERSION                      # 语义化版本号 (e.g., v1.2.3)
ARG COMMIT_SHA                   # Git commit SHA
```

#### 运行时 ENV 变量

运行时环境变量应通过 docker-compose.yml 或 Kubernetes ConfigMap 传入，**禁止硬编码敏感信息**。

**敏感变量清单（禁止硬编码）**:
- `DB_PASSWORD` - 数据库密码
- `REDIS_PASSWORD` - Redis 密码
- `JWT_SECRET` - JWT 签名密钥
- `SMTP_PASSWORD` - 邮件服务密码
- `SENDGRID_API_KEY` - SendGrid API 密钥
- `MAILGUN_API_KEY` - Mailgun API 密钥
- `AWS_SECRET_ACCESS_KEY` - AWS 密钥

**示例**:
```dockerfile
# ✅ 正确：通过运行时传入
# docker run -e DB_PASSWORD=secret app:latest

# ❌ 错误：硬编码敏感信息
ENV DB_PASSWORD=hardcoded_secret
```

### 2.2 ARG 命名约定

| 变量类型 | 命名格式 | 示例 |
|---------|---------|------|
| 版本号 | `<COMPONENT>_VERSION` | `GO_VERSION`, `NODE_VERSION` |
| Digest | `<IMAGE>_DIGEST` | `GO_ALPINE_DIGEST`, `ALPINE_DIGEST` |
| 构建元数据 | 大写下划线 | `BUILD_DATE`, `COMMIT_SHA` |
| 路径 | `<NAME>_PATH` | `BUILD_PATH`, `OUTPUT_PATH` |

### 2.3 构建参数验证

所有构建脚本必须验证必需的 ARG 参数：

```bash
# 示例：scripts/build-images.sh 中的验证
if [ -z "$VERSION" ]; then
  echo "Error: VERSION is required"
  exit 1
fi

if [ -z "$COMMIT_SHA" ]; then
  echo "Error: COMMIT_SHA is required"
  exit 1
fi
```

---

## 3. 镜像标签策略

### 3.1 标签规范

禁止在生产环境使用 `:latest` 标签。所有镜像必须使用以下标签组合：

#### 主标签格式

```
<registry>/<image-name>:<tag>
```

#### 标签类型

| 标签类型 | 格式 | 示例 | 用途 |
|---------|------|------|------|
| 语义化版本 | `v<MAJOR>.<MINOR>.<PATCH>` | `v1.2.3` | 生产发布 |
| 版本+SHA | `v<VERSION>-<SHORT_SHA>` | `v1.2.3-abc1234` | 可追溯发布 |
| Commit SHA | `sha-<COMMIT_SHA>` | `sha-abc1234def5678` | CI/CD 构建 |
| 开发版本 | `latest-dev` | `latest-dev` | 开发环境 |
| 分支版本 | `branch-<BRANCH>` | `branch-feature-auth` | 功能分支测试 |

### 3.2 标签示例

```bash
# 生产发布（推荐）
edgelink/api-gateway:v1.2.3
edgelink/api-gateway:v1.2.3-abc1234

# CI/CD 构建
edgelink/api-gateway:sha-abc1234def5678

# 开发环境
edgelink/api-gateway:latest-dev
edgelink/api-gateway:branch-001-edge-link-core
```

### 3.3 OCI 元数据标签

所有镜像必须包含 OCI 标准元数据标签：

```dockerfile
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT_SHA}" \
      org.opencontainers.image.title="EdgeLink API Gateway" \
      org.opencontainers.image.description="EdgeLink Control Plane API Gateway Service" \
      org.opencontainers.image.vendor="EdgeLink" \
      org.opencontainers.image.authors="EdgeLink Development Team" \
      org.opencontainers.image.source="https://github.com/edgelink/edge-link" \
      org.opencontainers.image.licenses="Apache-2.0"
```

**查看镜像元数据**:
```bash
docker inspect edgelink/api-gateway:v1.2.3 | jq '.[0].Config.Labels'
```

---

## 4. 安全和优化需求

### 4.1 非 Root 用户运行

所有后端服务镜像必须使用非 root 用户运行：

```dockerfile
# 创建非 root 用户
RUN addgroup -g 1000 edgelink && \
    adduser -D -u 1000 -G edgelink edgelink

# 切换到非 root 用户
USER edgelink
```

**验证方法**:
```bash
docker run --rm edgelink/api-gateway:v1.2.3 id
# 预期输出: uid=1000(edgelink) gid=1000(edgelink)
```

### 4.2 镜像大小限制

| 服务类型 | 大小限制 | 说明 |
|---------|---------|------|
| Go 后端服务 | < 100 MB | 使用 alpine 基础镜像 + 静态编译 |
| 前端 (Nginx) | < 50 MB | 仅包含静态资源 + nginx |
| Sidecar 客户端 | < 80 MB | 包含 WireGuard 工具 |

**查看镜像大小**:
```bash
docker images edgelink/api-gateway:v1.2.3 --format "{{.Size}}"
```

### 4.3 临时文件和缓存清理

构建过程必须清理以下内容：

```dockerfile
# ✅ 清理 APK 缓存
RUN apk add --no-cache ca-certificates && \
    rm -rf /var/cache/apk/*

# ✅ 清理构建缓存
RUN pnpm run build && \
    rm -rf node_modules .cache

# ✅ 单层安装+清理
RUN apk add --no-cache --virtual .build-deps gcc musl-dev && \
    go build ... && \
    apk del .build-deps
```

### 4.4 健康检查

所有服务必须定义健康检查：

```dockerfile
# HTTP 服务
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# gRPC 服务
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD nc -z localhost 50051 || exit 1

# 进程检查
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD pgrep background-worker || exit 1
```

---

## 5. 漏洞扫描标准

### 5.1 扫描工具

项目使用以下工具进行漏洞扫描：

- **Trivy**: 容器镜像和文件系统扫描
- **Grype**: 深度漏洞分析
- **Docker Scout** (可选): Docker Hub 集成扫描

### 5.2 通过标准

| 严重性级别 | 允许数量 | 处理要求 |
|-----------|---------|---------|
| Critical | 0 | 必须修复，阻塞发布 |
| High | 0 | 必须修复，阻塞发布 |
| Medium | ≤ 5 | 需评估风险，记录豁免 |
| Low | 不限 | 可接受 |

### 5.3 扫描命令

```bash
# Trivy 扫描
trivy image --severity CRITICAL,HIGH edgelink/api-gateway:v1.2.3

# Grype 扫描
grype edgelink/api-gateway:v1.2.3 --fail-on critical

# 生成 SBOM
syft edgelink/api-gateway:v1.2.3 -o cyclonedx-json > sbom.json
```

### 5.4 豁免流程

对于无法立即修复的 Medium 级别漏洞：

1. 在 `docs/security/cve-exemptions.md` 中记录
2. 说明漏洞影响范围和缓解措施
3. 设置复查日期（不超过 30 天）
4. 由安全负责人审批

---

## 6. 构建流程

### 6.1 本地构建

```bash
# 设置构建参数
export VERSION=v1.2.3
export COMMIT_SHA=$(git rev-parse --short HEAD)
export BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

# 构建单个服务
docker build \
  --build-arg VERSION=${VERSION} \
  --build-arg COMMIT_SHA=${COMMIT_SHA} \
  --build-arg BUILD_DATE=${BUILD_DATE} \
  -f infrastructure/docker/Dockerfile.api-gateway \
  -t edgelink/api-gateway:${VERSION} \
  .

# 使用统一构建脚本
./scripts/build-images.sh --version ${VERSION} --push
```

### 6.2 CI/CD 构建

GitHub Actions 自动构建流程（`.github/workflows/build-images.yml`）：

```yaml
- name: Build and push images
  env:
    VERSION: ${{ github.ref_name }}
    COMMIT_SHA: ${{ github.sha }}
    BUILD_DATE: ${{ steps.date.outputs.date }}
  run: |
    ./scripts/build-images.sh \
      --version ${VERSION} \
      --commit-sha ${COMMIT_SHA} \
      --build-date ${BUILD_DATE} \
      --push
```

### 6.3 多架构构建

生产环境支持 amd64 和 arm64 架构：

```bash
# 使用 buildx 构建多架构镜像
docker buildx build \
  --platform linux/amd64,linux/arm64 \
  --build-arg VERSION=${VERSION} \
  --build-arg COMMIT_SHA=${COMMIT_SHA} \
  --build-arg BUILD_DATE=${BUILD_DATE} \
  -f infrastructure/docker/Dockerfile.api-gateway \
  -t edgelink/api-gateway:${VERSION} \
  --push \
  .
```

---

## 7. 验证方法

### 7.1 构建参数验证

```bash
# 验证 ARG 变量已正确传递
docker history edgelink/api-gateway:v1.2.3 --no-trunc | grep "BUILD_DATE"

# 验证 LABEL 元数据
docker inspect edgelink/api-gateway:v1.2.3 \
  | jq '.[0].Config.Labels["org.opencontainers.image.version"]'
```

### 7.2 安全验证

```bash
# 验证非 root 用户
docker run --rm edgelink/api-gateway:v1.2.3 id

# 验证镜像大小
docker images edgelink/api-gateway:v1.2.3 --format "{{.Size}}"

# 验证无 Critical/High 漏洞
trivy image --severity CRITICAL,HIGH --exit-code 1 edgelink/api-gateway:v1.2.3
```

### 7.3 可重现性验证

```bash
# 构建两次并对比 SHA256
docker build ... -t test1:latest
docker build ... -t test2:latest

docker images --digests | grep test
# 预期：两次构建的 digest 应相同（排除时间戳）
```

### 7.4 健康检查验证

```bash
# 启动容器
docker run -d --name test edgelink/api-gateway:v1.2.3

# 等待健康检查
sleep 10

# 验证健康状态
docker inspect test | jq '.[0].State.Health.Status'
# 预期输出: "healthy"
```

---

## 附录 A: Dockerfile 模板

### Go 后端服务模板

```dockerfile
# ============================================
# Stage 1: Build Stage
# Purpose: Compile Go binary with build dependencies
# ============================================
ARG GO_VERSION=1.21
ARG ALPINE_VERSION=3.18
ARG GO_ALPINE_DIGEST=sha256:2414035b086e3c42b99654c8b26e6f5b1b1598080d65fd03c7f499552ff4dc94
ARG ALPINE_DIGEST=sha256:de0eb0b3f2a47ba1eb89389859a9bd88b28e82f5826b6969ad604979713c2d4f

FROM golang:${GO_VERSION}-alpine@${GO_ALPINE_DIGEST} AS builder

# Declare build-time metadata
ARG BUILD_DATE
ARG VERSION
ARG COMMIT_SHA

# Install build dependencies
RUN apk add --no-cache \
    git=2.40.1-r0 \
    make=4.4.1-r1 \
    gcc=12.2.1_git20220924-r10 \
    musl-dev=1.2.4-r2

WORKDIR /build

# Layer 1: Download dependencies (cached unless go.mod/go.sum changes)
COPY backend/go.mod ./
COPY backend/go.sum* ./
RUN go mod download && go mod verify

# Layer 2: Copy source code (invalidates cache on code changes)
COPY backend/ ./

# Layer 3: Build the binary with reproducible build flags
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-w -s -buildid= -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildDate=${BUILD_DATE}" \
    -o /app/service \
    ./cmd/service

# ============================================
# Stage 2: Final Runtime Stage
# Purpose: Minimal production image with security hardening
# ============================================
FROM alpine:${ALPINE_VERSION}@${ALPINE_DIGEST}

# Declare build-time metadata (must re-declare in multi-stage build)
ARG BUILD_DATE
ARG VERSION
ARG COMMIT_SHA

# Install runtime dependencies
RUN apk add --no-cache \
    ca-certificates=20230506-r0 \
    tzdata=2024a-r0 && \
    rm -rf /var/cache/apk/*

# Create non-root user for security
RUN addgroup -g 1000 edgelink && \
    adduser -D -u 1000 -G edgelink edgelink

WORKDIR /app

# Copy binary from builder stage
COPY --from=builder /app/service /app/service

# Change ownership to non-root user
RUN chown -R edgelink:edgelink /app

# Add OCI metadata labels
LABEL org.opencontainers.image.created="${BUILD_DATE}" \
      org.opencontainers.image.version="${VERSION}" \
      org.opencontainers.image.revision="${COMMIT_SHA}" \
      org.opencontainers.image.title="EdgeLink Service" \
      org.opencontainers.image.description="EdgeLink Service Description" \
      org.opencontainers.image.vendor="EdgeLink" \
      org.opencontainers.image.authors="EdgeLink Development Team"

# Switch to non-root user
USER edgelink

# Expose service port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run the binary
ENTRYPOINT ["/app/service"]
```

---

## 附录 B: 常见问题

### Q1: 为什么要固定 digest 而不是版本标签？

版本标签（如 `alpine:3.18`）是可变的，镜像维护者可以推送新的镜像到同一标签。Digest 是镜像内容的 SHA256 哈希，唯一且不可变，确保构建的可重现性和安全性。

### Q2: 如何更新基础镜像？

1. 使用 `crane digest` 或 `docker pull` 获取新 digest
2. 更新 Dockerfile 中的 `ARG` 变量
3. 运行漏洞扫描验证无 Critical/High CVE
4. 在测试环境验证
5. 提交 PR 并更新本文档

### Q3: 构建时间戳会影响可重现性吗？

会。因此我们使用 `-buildvcs=false` 和 `-buildid=` 标志。时间戳仅用于元数据标签，不嵌入二进制文件。

### Q4: 为什么禁止使用 :latest？

`:latest` 标签指向的镜像内容会变化，导致：
1. 无法准确回滚到特定版本
2. 难以排查问题（不知道部署的确切版本）
3. 破坏构建的可重现性

---

**文档版本历史**:
- v1.0 (2025-10-21): 初始版本，定义基础规范
