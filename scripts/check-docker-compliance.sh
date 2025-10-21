#!/bin/bash
# EdgeLink Docker Build Compliance Checker
# Purpose: Validate Docker build configuration against checklist requirements
# Usage: ./scripts/check-docker-compliance.sh

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Counters
total_checks=0
passed_checks=0
failed_checks=0
warnings=0

# Function: Log message
log() {
  echo -e "${BLUE}[CHECK]${NC} $*"
}

# Function: Pass message
pass() {
  echo -e "${GREEN}[PASS]${NC} $*"
  ((passed_checks++))
  ((total_checks++))
}

# Function: Fail message
fail() {
  echo -e "${RED}[FAIL]${NC} $*"
  ((failed_checks++))
  ((total_checks++))
}

# Function: Warn message
warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
  ((warnings++))
}

# Function: Info message
info() {
  echo -e "${BLUE}[INFO]${NC} $*"
}

echo "==============================================="
echo "EdgeLink Docker Build Compliance Check"
echo "==============================================="
echo ""

# ============================================
# 1.1 基础镜像规范
# ============================================
echo "=== 1.1 基础镜像规范 ==="

log "Checking if base images are pinned to digest..."
dockerfiles=(
  "infrastructure/docker/Dockerfile.api-gateway"
  "infrastructure/docker/Dockerfile.device-service"
  "infrastructure/docker/Dockerfile.topology-service"
  "infrastructure/docker/Dockerfile.nat-coordinator"
  "infrastructure/docker/Dockerfile.alert-service"
  "infrastructure/docker/Dockerfile.background-worker"
  "infrastructure/docker/Dockerfile.edgelink-sidecar"
  "frontend/Dockerfile"
)

all_pinned=true
for dockerfile in "${dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    fail "$dockerfile does not exist"
    all_pinned=false
    continue
  fi

  # Check for FROM statements with digest
  from_lines=$(grep "^FROM " "$filepath" || true)
  # Check if using digest directly (@sha256:) or via ARG variable (DIGEST})
  if echo "$from_lines" | grep -qE "(@sha256:|DIGEST})"; then
    info "  ✓ $dockerfile uses digest pinning"
  else
    fail "  ✗ $dockerfile missing digest pinning"
    all_pinned=false
  fi
done

if [[ "$all_pinned" == "true" ]]; then
  pass "All Dockerfiles use digest-pinned base images"
else
  fail "Some Dockerfiles are missing digest pinning"
fi

log "Checking for base image documentation..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "基础镜像选择标准" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "Base image selection rationale documented"
  else
    fail "Base image rationale not found in documentation"
  fi
else
  fail "docs/docker-build-spec.md not found"
fi

log "Checking for vulnerability scan standards..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "漏洞扫描标准" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "Vulnerability scan standards defined"
  else
    fail "Vulnerability scan standards not documented"
  fi
else
  fail "Vulnerability scan standards documentation missing"
fi

echo ""

# ============================================
# 1.2 多阶段构建需求
# ============================================
echo "=== 1.2 多阶段构建需求 ==="

log "Checking for build stage comments..."
all_commented=true
for dockerfile in "${dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  # Check for stage comments
  if grep -q "# Stage 1: Build Stage" "$filepath" && \
     grep -q "# Purpose:" "$filepath"; then
    info "  ✓ $dockerfile has clear stage comments"
  else
    fail "  ✗ $dockerfile missing stage comments"
    all_commented=false
  fi
done

if [[ "$all_commented" == "true" ]]; then
  pass "All Dockerfiles have clear build stage comments"
else
  fail "Some Dockerfiles missing stage comments"
fi

log "Checking for declared build dependencies..."
all_deps=true
for dockerfile in "${dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  # Check for dependency comments
  if grep -q "# Install build dependencies" "$filepath" || \
     grep -q "# Install runtime dependencies" "$filepath"; then
    info "  ✓ $dockerfile documents dependencies"
  else
    warn "  ! $dockerfile may be missing dependency documentation"
    all_deps=false
  fi
done

if [[ "$all_deps" == "true" ]]; then
  pass "Build dependencies are documented"
else
  warn "Some Dockerfiles may need better dependency documentation"
fi

log "Checking for fixed build tool versions..."
# Check for versioned apk packages
if grep -r "apk add.*=" infrastructure/docker/ frontend/Dockerfile 2>/dev/null | grep -q "="; then
  pass "Build tools have version pinning in apk commands"
else
  warn "Consider adding version pinning to apk packages"
fi

log "Checking for layer caching optimization..."
# Check for COPY go.mod before COPY source
go_dockerfiles=(
  "infrastructure/docker/Dockerfile.api-gateway"
  "infrastructure/docker/Dockerfile.device-service"
  "infrastructure/docker/Dockerfile.topology-service"
  "infrastructure/docker/Dockerfile.nat-coordinator"
  "infrastructure/docker/Dockerfile.alert-service"
  "infrastructure/docker/Dockerfile.background-worker"
  "infrastructure/docker/Dockerfile.edgelink-sidecar"
)

layer_optimized=true
for dockerfile in "${go_dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  # Check if go.mod is copied before source code
  gomod_line=$(grep -n "COPY.*go.mod" "$filepath" | head -1 | cut -d: -f1)
  source_line=$(grep -n "COPY backend/" "$filepath" | head -1 | cut -d: -f1)

  if [[ -n "$gomod_line" ]] && [[ -n "$source_line" ]] && [[ "$gomod_line" -lt "$source_line" ]]; then
    info "  ✓ $dockerfile optimizes layer caching"
  else
    warn "  ! $dockerfile may not optimize layer caching"
    layer_optimized=false
  fi
done

if [[ "$layer_optimized" == "true" ]]; then
  pass "Dockerfiles optimize layer caching"
else
  warn "Some Dockerfiles could improve layer caching"
fi

echo ""

# ============================================
# 1.3 构建参数和环境变量
# ============================================
echo "=== 1.3 构建参数和环境变量 ==="

log "Checking for ARG/ENV documentation..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "构建参数和环境变量" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "ARG and ENV variables are documented"
  else
    fail "ARG/ENV documentation not found"
  fi
else
  fail "Docker build spec documentation missing"
fi

log "Checking for sensitive variable warnings..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "敏感变量清单" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "Sensitive variables are marked in documentation"
  else
    fail "Sensitive variables not documented"
  fi
else
  fail "Sensitive variables documentation missing"
fi

log "Checking for consistent ARG naming..."
# Check if all Dockerfiles use standard ARG names
standard_args=("VERSION" "COMMIT_SHA" "BUILD_DATE" "GO_VERSION" "ALPINE_VERSION")
all_consistent=true

for dockerfile in "${go_dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  for arg in "${standard_args[@]}"; do
    if grep -q "ARG $arg" "$filepath"; then
      continue
    else
      # VERSION, COMMIT_SHA, BUILD_DATE are optional in first FROM
      if [[ "$arg" != "VERSION" ]] && [[ "$arg" != "COMMIT_SHA" ]] && [[ "$arg" != "BUILD_DATE" ]]; then
        warn "  ! $dockerfile missing ARG $arg"
        all_consistent=false
      fi
    fi
  done
done

if [[ "$all_consistent" == "true" ]]; then
  pass "ARG naming is consistent across Dockerfiles"
else
  warn "ARG naming could be more consistent"
fi

echo ""

# ============================================
# 1.4 镜像标签策略
# ============================================
echo "=== 1.4 镜像标签策略 ==="

log "Checking for image tagging strategy documentation..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "镜像标签策略" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "Image tagging strategy is documented"
  else
    fail "Image tagging strategy not documented"
  fi
else
  fail "Docker build spec missing"
fi

log "Checking for :latest prohibition..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "禁止.*:latest" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "Production use of :latest is prohibited in docs"
  else
    warn "Should explicitly prohibit :latest in production"
  fi
else
  warn "Documentation should prohibit :latest tag"
fi

log "Checking for OCI metadata labels..."
all_labeled=true
for dockerfile in "${dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  if grep -q "org.opencontainers.image" "$filepath"; then
    info "  ✓ $dockerfile includes OCI labels"
  else
    fail "  ✗ $dockerfile missing OCI metadata labels"
    all_labeled=false
  fi
done

if [[ "$all_labeled" == "true" ]]; then
  pass "All Dockerfiles include OCI metadata labels"
else
  fail "Some Dockerfiles missing OCI labels"
fi

echo ""

# ============================================
# 1.5 镜像安全和优化需求
# ============================================
echo "=== 1.5 镜像安全和优化需求 ==="

log "Checking for non-root user..."
all_nonroot=true
for dockerfile in "${go_dockerfiles[@]}" "infrastructure/docker/Dockerfile.alert-service" "infrastructure/docker/Dockerfile.background-worker"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  if grep -q "USER edgelink" "$filepath" || grep -q "USER 1000" "$filepath"; then
    info "  ✓ $dockerfile uses non-root user"
  else
    fail "  ✗ $dockerfile missing non-root user"
    all_nonroot=false
  fi
done

if [[ "$all_nonroot" == "true" ]]; then
  pass "All backend services use non-root user"
else
  fail "Some services missing non-root user configuration"
fi

log "Checking for image size limits documentation..."
if [[ -f "$PROJECT_ROOT/docs/docker-build-spec.md" ]]; then
  if grep -q "镜像大小限制" "$PROJECT_ROOT/docs/docker-build-spec.md"; then
    pass "Image size limits are documented"
  else
    fail "Image size limits not documented"
  fi
else
  fail "Image size documentation missing"
fi

log "Checking for cache cleanup..."
all_cleaned=true
for dockerfile in "${dockerfiles[@]}"; do
  filepath="$PROJECT_ROOT/$dockerfile"
  if [[ ! -f "$filepath" ]]; then
    continue
  fi

  if grep -q "rm -rf.*cache" "$filepath" || grep -q "rm -rf /var/cache/apk" "$filepath"; then
    info "  ✓ $dockerfile cleans up caches"
  else
    warn "  ! $dockerfile may not clean up caches"
    all_cleaned=false
  fi
done

if [[ "$all_cleaned" == "true" ]]; then
  pass "All Dockerfiles clean up temporary files and caches"
else
  warn "Some Dockerfiles could improve cache cleanup"
fi

echo ""

# ============================================
# Additional checks
# ============================================
echo "=== Additional Checks ==="

log "Checking for build script..."
if [[ -f "$PROJECT_ROOT/scripts/build-images.sh" ]]; then
  if [[ -x "$PROJECT_ROOT/scripts/build-images.sh" ]]; then
    pass "Unified build script exists and is executable"
  else
    warn "Build script exists but is not executable"
  fi
else
  fail "Unified build script not found"
fi

log "Checking docker-compose.yml..."
if [[ -f "$PROJECT_ROOT/docker-compose.yml" ]]; then
  if grep -q "x-build-args" "$PROJECT_ROOT/docker-compose.yml"; then
    pass "docker-compose.yml uses build args"
  else
    warn "docker-compose.yml could use build args"
  fi

  if grep -q '@sha256:' "$PROJECT_ROOT/docker-compose.yml"; then
    pass "docker-compose.yml uses digest-pinned infrastructure images"
  else
    warn "docker-compose.yml could pin infrastructure images"
  fi
else
  fail "docker-compose.yml not found"
fi

log "Checking for .env.example..."
if [[ -f "$PROJECT_ROOT/.env.example" ]]; then
  pass ".env.example file exists for configuration reference"
else
  warn ".env.example file not found"
fi

echo ""

# ============================================
# Summary
# ============================================
echo "==============================================="
echo "Summary"
echo "==============================================="
echo "Total Checks: $total_checks"
echo -e "${GREEN}Passed:${NC} $passed_checks"
echo -e "${RED}Failed:${NC} $failed_checks"
echo -e "${YELLOW}Warnings:${NC} $warnings"
echo ""

pass_rate=$((passed_checks * 100 / total_checks))
echo "Pass Rate: ${pass_rate}%"

if [[ $failed_checks -eq 0 ]]; then
  echo -e "${GREEN}✓ All critical checks passed!${NC}"
  exit 0
elif [[ $pass_rate -ge 80 ]]; then
  echo -e "${YELLOW}⚠ Most checks passed, but some improvements needed${NC}"
  exit 0
else
  echo -e "${RED}✗ Too many failed checks. Please review the output above.${NC}"
  exit 1
fi
