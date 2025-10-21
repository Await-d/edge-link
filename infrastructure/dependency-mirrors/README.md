# 依赖镜像备份配置

本目录包含EdgeLink项目的依赖镜像和备份配置，确保构建稳定性和网络故障容错能力。

## 概述

EdgeLink项目依赖多个外部服务来获取依赖包和镜像：

- **Go modules**: proxy.golang.org, sum.golang.org
- **npm/pnpm**: registry.npmjs.org
- **Docker images**: docker.io, ghcr.io
- **Linux packages**: alpine packages, debian packages

为了避免：
- 公共镜像源故障或限流导致构建失败
- 网络问题导致依赖下载超时
- 恶意包投毒攻击
- 依赖包被删除或更改

我们配置了多层镜像备份策略。

## 镜像配置

### 1. Go Module Proxy

#### 主要配置

```bash
# .env or CI environment
export GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org
export GOPRIVATE=github.com/edgelink/*
```

#### 推荐公共镜像

| 镜像源 | URL | 地区 | 可靠性 |
|-------|-----|------|-------|
| 七牛云 | https://goproxy.cn | 中国 | ⭐⭐⭐⭐⭐ |
| 阿里云 | https://mirrors.aliyun.com/goproxy/ | 中国 | ⭐⭐⭐⭐ |
| 官方 | https://proxy.golang.org | 全球 | ⭐⭐⭐⭐⭐ |

#### 私有Go Proxy（可选）

使用Athens作为私有Go module proxy：

```bash
# 部署Athens
docker run -d \
  --name athens-proxy \
  -p 3000:3000 \
  -e ATHENS_STORAGE_TYPE=disk \
  -e ATHENS_DISK_STORAGE_ROOT=/var/lib/athens \
  -v athens-storage:/var/lib/athens \
  gomods/athens:latest

# 配置
export GOPROXY=http://athens-proxy:3000,https://goproxy.cn,direct
```

**配置文件**: `athens-config.toml`

### 2. npm/pnpm Registry

#### 主要配置

```bash
# .npmrc or .pnpmrc
registry=https://registry.npmmirror.com
# 备用
# registry=https://registry.npm.taobao.org
```

#### 推荐公共镜像

| 镜像源 | URL | 地区 | 可靠性 |
|-------|-----|------|-------|
| npmmirror | https://registry.npmmirror.com | 中国 | ⭐⭐⭐⭐⭐ |
| 淘宝镜像 | https://registry.npm.taobao.org | 中国 | ⭐⭐⭐⭐ |
| 官方 | https://registry.npmjs.org | 全球 | ⭐⭐⭐⭐⭐ |

#### 私有npm Registry（可选）

使用Verdaccio作为私有npm registry：

```bash
# 部署Verdaccio
docker run -d \
  --name verdaccio \
  -p 4873:4873 \
  -v verdaccio-storage:/verdaccio/storage \
  verdaccio/verdaccio

# 配置
npm set registry http://verdaccio:4873
pnpm set registry http://verdaccio:4873
```

**配置文件**: `verdaccio-config.yaml`

### 3. Docker Base Image镜像

#### 主要配置

使用Docker Hub镜像加速：

```json
// /etc/docker/daemon.json
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.ccs.tencentyun.com"
  ],
  "insecure-registries": [],
  "max-concurrent-downloads": 10
}
```

#### 推荐镜像源

| 镜像源 | URL | 地区 | 可靠性 |
|-------|-----|------|-------|
| 中科大 | https://docker.mirrors.ustc.edu.cn | 中国 | ⭐⭐⭐⭐⭐ |
| 网易 | https://hub-mirror.c.163.com | 中国 | ⭐⭐⭐⭐ |
| 腾讯云 | https://mirror.ccs.tencentyun.com | 中国 | ⭐⭐⭐⭐ |
| 阿里云 | https://[your-id].mirror.aliyuncs.com | 中国 | ⭐⭐⭐⭐ |

#### 私有Docker Registry

使用Harbor作为私有Docker registry：

```bash
# 部署Harbor（使用docker-compose）
curl -L https://github.com/goharbor/harbor/releases/download/v2.10.0/harbor-offline-installer-v2.10.0.tgz \
  -o harbor.tgz
tar xvf harbor.tgz
cd harbor
./install.sh

# 镜像同步到Harbor
docker pull golang:1.21-alpine
docker tag golang:1.21-alpine harbor.edgelink.com/library/golang:1.21-alpine
docker push harbor.edgelink.com/library/golang:1.21-alpine
```

**配置文件**: `harbor.yml`

### 4. Alpine Package镜像

```dockerfile
# Dockerfile中配置
RUN sed -i 's/dl-cdn.alpinelinux.org/mirrors.ustc.edu.cn/g' /etc/apk/repositories
RUN apk add --no-cache ca-certificates
```

#### 推荐镜像源

| 镜像源 | URL | 地区 |
|-------|-----|------|
| 中科大 | mirrors.ustc.edu.cn | 中国 |
| 阿里云 | mirrors.aliyun.com | 中国 |
| 清华 | mirrors.tuna.tsinghua.edu.cn | 中国 |

## 自动化镜像同步

### GitHub Actions自动同步Docker镜像

```yaml
# .github/workflows/sync-base-images.yml
name: Sync Base Images

on:
  schedule:
    - cron: '0 2 * * 0'  # 每周日凌晨2点
  workflow_dispatch:

jobs:
  sync:
    runs-on: ubuntu-latest
    strategy:
      matrix:
        image:
          - golang:1.21-alpine
          - alpine:3.18
          - node:20-alpine
          - postgres:15-alpine
          - redis:7-alpine
    steps:
      - name: Pull from Docker Hub
        run: docker pull ${{ matrix.image }}
      
      - name: Tag for private registry
        run: |
          docker tag ${{ matrix.image }} \
            harbor.edgelink.com/library/${{ matrix.image }}
      
      - name: Push to Harbor
        run: |
          echo "${{ secrets.HARBOR_PASSWORD }}" | \
            docker login harbor.edgelink.com -u admin --password-stdin
          docker push harbor.edgelink.com/library/${{ matrix.image }}
```

**配置文件**: `sync-base-images.yml`

### 定期验证镜像可用性

```yaml
# .github/workflows/verify-mirrors.yml
name: Verify Dependency Mirrors

on:
  schedule:
    - cron: '0 */6 * * *'  # 每6小时
  workflow_dispatch:

jobs:
  verify-go-proxy:
    runs-on: ubuntu-latest
    steps:
      - name: Test Go Proxy
        run: |
          GOPROXY=https://goproxy.cn go mod download golang.org/x/text@latest
          echo "✅ Go Proxy OK"
  
  verify-npm-registry:
    runs-on: ubuntu-latest
    steps:
      - name: Test npm Registry
        run: |
          npm config set registry https://registry.npmmirror.com
          npm info react@latest
          echo "✅ npm Registry OK"
  
  verify-docker-registry:
    runs-on: ubuntu-latest
    steps:
      - name: Test Docker Pull
        run: |
          docker pull alpine:3.18
          echo "✅ Docker Registry OK"
```

**配置文件**: `verify-mirrors.yml`

## 使用指南

### 开发环境配置

#### 配置Go Proxy

```bash
# Linux/macOS (~/.bashrc or ~/.zshrc)
export GOPROXY=https://goproxy.cn,https://proxy.golang.org,direct
export GOSUMDB=sum.golang.org

# Windows (PowerShell)
$env:GOPROXY = "https://goproxy.cn,https://proxy.golang.org,direct"
$env:GOSUMDB = "sum.golang.org"

# 验证配置
go env GOPROXY
```

#### 配置npm/pnpm镜像

```bash
# npm
npm config set registry https://registry.npmmirror.com

# pnpm
pnpm config set registry https://registry.npmmirror.com

# 验证配置
npm config get registry
pnpm config get registry
```

#### 配置Docker镜像加速

```bash
# Linux
sudo mkdir -p /etc/docker
sudo tee /etc/docker/daemon.json <<-'EOF'
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn"
  ]
}
EOF
sudo systemctl daemon-reload
sudo systemctl restart docker

# macOS/Windows
# 在Docker Desktop设置中配置Registry Mirrors

# 验证配置
docker info | grep -A 5 "Registry Mirrors"
```

### CI/CD环境配置

#### GitHub Actions

```yaml
# .github/workflows/build.yml
env:
  GOPROXY: https://goproxy.cn,direct
  NPM_REGISTRY: https://registry.npmmirror.com

jobs:
  build:
    steps:
      - name: Setup Go
        uses: actions/setup-go@v5
        with:
          go-version: 1.21
      
      - name: Configure Go Proxy
        run: |
          go env -w GOPROXY=${{ env.GOPROXY }}
          go env -w GOSUMDB=sum.golang.org
      
      - name: Setup pnpm
        run: |
          pnpm config set registry ${{ env.NPM_REGISTRY }}
```

#### GitLab CI

```yaml
# .gitlab-ci.yml
variables:
  GOPROXY: "https://goproxy.cn,direct"
  NPM_CONFIG_REGISTRY: "https://registry.npmmirror.com"
  DOCKER_REGISTRY_MIRROR: "https://docker.mirrors.ustc.edu.cn"

before_script:
  - go env -w GOPROXY=$GOPROXY
  - npm config set registry $NPM_CONFIG_REGISTRY
```

## 故障切换策略

### Go Proxy故障切换

```bash
# GOPROXY使用逗号分隔的多个源，自动fallback
export GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,https://proxy.golang.org,direct

# 如果goproxy.cn失败，自动尝试aliyun镜像
# 如果所有镜像失败，direct直接从源获取
```

### npm Registry故障切换

```bash
# 使用npm-registry-cli工具自动切换
npm install -g npm-registry-cli

# 自动检测最快镜像
nrc test

# 使用最快镜像
nrc use fastest
```

### Docker Registry故障切换

```json
// daemon.json配置多个镜像源
{
  "registry-mirrors": [
    "https://docker.mirrors.ustc.edu.cn",
    "https://hub-mirror.c.163.com",
    "https://mirror.ccs.tencentyun.com"
  ]
}
```

Docker会自动按顺序尝试各镜像源。

## 监控和告警

### Prometheus监控镜像可用性

```yaml
# prometheus-rules.yml
- alert: DependencyMirrorDown
  expr: |
    probe_success{job="dependency-mirrors"} == 0
  for: 10m
  labels:
    severity: warning
  annotations:
    summary: "Dependency mirror {{ $labels.instance }} is down"
```

### 监控脚本

```bash
#!/bin/bash
# scripts/monitor-mirrors.sh

echo "Checking Go Proxy..."
curl -sf https://goproxy.cn || echo "❌ goproxy.cn down"

echo "Checking npm Registry..."
curl -sf https://registry.npmmirror.com || echo "❌ npmmirror down"

echo "Checking Docker Mirror..."
curl -sf https://docker.mirrors.ustc.edu.cn/v2/ || echo "❌ Docker mirror down"
```

## 安全考虑

### 1. 依赖包完整性验证

```bash
# Go: 使用GOSUMDB验证
go mod verify

# npm: 使用package-lock.json验证
npm ci  # 严格按lockfile安装

# pnpm: 使用pnpm-lock.yaml验证
pnpm install --frozen-lockfile
```

### 2. 私有包保护

```bash
# Go: 配置GOPRIVATE避免私有包泄露到公共proxy
export GOPRIVATE=github.com/edgelink/*

# npm: 使用.npmrc配置scope
@edgelink:registry=https://npm.edgelink.com/
```

### 3. 镜像源信任

只使用可信镜像源：
- 官方镜像
- 知名大学/公司维护的镜像
- 自建私有镜像

避免使用来源不明的镜像源。

## 故障排查

### Go模块下载失败

```bash
# 清除模块缓存
go clean -modcache

# 强制重新下载
go mod download -x

# 检查proxy配置
go env GOPROXY GOSUMDB

# 验证sum database
go mod verify
```

### npm包下载失败

```bash
# 清除npm缓存
npm cache clean --force

# 切换镜像源
npm config set registry https://registry.npmjs.org

# 查看详细日志
npm install --loglevel verbose
```

### Docker镜像拉取失败

```bash
# 检查镜像加速配置
docker info | grep "Registry Mirrors"

# 尝试直接从Docker Hub拉取
docker pull --disable-content-trust docker.io/library/alpine:3.18

# 检查网络连接
curl -v https://docker.mirrors.ustc.edu.cn/v2/
```

## 相关文档

- [Go Module Proxy协议](https://go.dev/ref/mod#goproxy-protocol)
- [npm配置文档](https://docs.npmjs.com/cli/v9/configuring-npm/npmrc)
- [Docker Registry HTTP API](https://docs.docker.com/registry/spec/api/)
- [Athens Go Proxy](https://docs.gomods.io/)
- [Verdaccio npm Proxy](https://verdaccio.org/)
- [Harbor Registry](https://goharbor.io/)

## 维护指南

### 定期检查镜像源状态

```bash
# 每月检查一次镜像源可用性
./scripts/monitor-mirrors.sh

# 更新镜像源列表（如果某个源不可用）
# 编辑本文档和相关配置文件
```

### 更新私有镜像

```bash
# 定期同步上游镜像到私有registry
./scripts/sync-mirrors.sh

# 验证同步结果
./scripts/verify-private-registry.sh
```

---

**最后更新**: 2025-10-20  
**维护者**: EdgeLink DevOps Team  
**联系方式**: devops@edgelink.com
