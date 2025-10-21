#!/bin/bash
# EdgeLink Docker Image Build Script
# Purpose: Unified build script for all EdgeLink Docker images
# Usage: ./scripts/build-images.sh [options]

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Default values
VERSION=""
COMMIT_SHA=""
BUILD_DATE=""
REGISTRY="edgelink"
PUSH=false
SERVICES=""
PLATFORM="linux/amd64"
SCAN=false

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Available services
ALL_SERVICES=(
  "api-gateway"
  "device-service"
  "topology-service"
  "nat-coordinator"
  "alert-service"
  "background-worker"
  "frontend"
  "edgelink-sidecar"
)

# Function: Print usage
usage() {
  cat <<EOF
Usage: ${0##*/} [OPTIONS]

Build EdgeLink Docker images with consistent versioning and metadata.

OPTIONS:
  -v, --version VERSION        Version tag (required, e.g., v1.2.3)
  -c, --commit-sha SHA         Git commit SHA (default: auto-detect)
  -d, --build-date DATE        Build date in RFC3339 (default: now)
  -r, --registry REGISTRY      Docker registry (default: edgelink)
  -s, --services SERVICE(S)    Comma-separated services to build (default: all)
  -p, --platform PLATFORM      Target platform (default: linux/amd64)
      --push                   Push images to registry
      --scan                   Run vulnerability scan after build
  -h, --help                   Show this help message

EXAMPLES:
  # Build all services for development
  ${0##*/} --version v1.2.3-dev

  # Build and push specific services
  ${0##*/} --version v1.2.3 --services api-gateway,frontend --push

  # Multi-platform build
  ${0##*/} --version v1.2.3 --platform linux/amd64,linux/arm64 --push

  # Build with vulnerability scan
  ${0##*/} --version v1.2.3 --scan

SERVICES:
  ${ALL_SERVICES[*]}

EOF
}

# Function: Log message
log() {
  echo -e "${GREEN}[INFO]${NC} $*"
}

# Function: Log warning
warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
}

# Function: Log error
error() {
  echo -e "${RED}[ERROR]${NC} $*" >&2
}

# Function: Die with error
die() {
  error "$*"
  exit 1
}

# Function: Validate dependencies
check_dependencies() {
  local deps=("docker" "git")

  if [[ "$SCAN" == "true" ]]; then
    deps+=("trivy")
  fi

  for cmd in "${deps[@]}"; do
    if ! command -v "$cmd" &> /dev/null; then
      die "Required command '$cmd' not found. Please install it first."
    fi
  done
}

# Function: Auto-detect Git metadata
detect_git_metadata() {
  if [[ -z "$COMMIT_SHA" ]]; then
    if git rev-parse --git-dir > /dev/null 2>&1; then
      COMMIT_SHA=$(git rev-parse --short=8 HEAD)
      log "Auto-detected commit SHA: $COMMIT_SHA"
    else
      warn "Not a git repository, using 'unknown' as commit SHA"
      COMMIT_SHA="unknown"
    fi
  fi

  if [[ -z "$BUILD_DATE" ]]; then
    BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    log "Auto-detected build date: $BUILD_DATE"
  fi
}

# Function: Validate version format
validate_version() {
  if [[ -z "$VERSION" ]]; then
    die "Version is required. Use --version to specify."
  fi

  # Accept semantic version (v1.2.3) or dev versions (v1.2.3-dev, v1.2.3-abc1234)
  if [[ ! "$VERSION" =~ ^v[0-9]+\.[0-9]+\.[0-9]+(-[a-zA-Z0-9]+)?$ ]]; then
    warn "Version '$VERSION' does not match semantic versioning format (vX.Y.Z)"
  fi
}

# Function: Build single service
build_service() {
  local service=$1
  local dockerfile=""
  local context="$PROJECT_ROOT"
  local image_name="${REGISTRY}/${service}:${VERSION}"

  # Determine Dockerfile path
  if [[ "$service" == "frontend" ]]; then
    dockerfile="$PROJECT_ROOT/frontend/Dockerfile"
    context="$PROJECT_ROOT/frontend"
  elif [[ "$service" == "edgelink-sidecar" ]]; then
    dockerfile="$PROJECT_ROOT/infrastructure/docker/Dockerfile.edgelink-sidecar"
  else
    dockerfile="$PROJECT_ROOT/infrastructure/docker/Dockerfile.${service}"
  fi

  # Check if Dockerfile exists
  if [[ ! -f "$dockerfile" ]]; then
    error "Dockerfile not found: $dockerfile"
    return 1
  fi

  log "Building $service..."
  log "  Dockerfile: $dockerfile"
  log "  Context: $context"
  log "  Image: $image_name"
  log "  Version: $VERSION"
  log "  Commit: $COMMIT_SHA"
  log "  Build Date: $BUILD_DATE"
  log "  Platform: $PLATFORM"

  # Build command
  local build_cmd=(
    docker build
    --build-arg "VERSION=${VERSION}"
    --build-arg "COMMIT_SHA=${COMMIT_SHA}"
    --build-arg "BUILD_DATE=${BUILD_DATE}"
    --platform "$PLATFORM"
    -f "$dockerfile"
    -t "$image_name"
  )

  # Add additional tags
  build_cmd+=(-t "${REGISTRY}/${service}:${VERSION}-${COMMIT_SHA}")

  # If version is not a dev version, add stable tags
  if [[ ! "$VERSION" =~ -dev$ ]] && [[ ! "$VERSION" =~ -[a-f0-9]{8}$ ]]; then
    # Extract major.minor version (v1.2.3 -> v1.2)
    local minor_version
    minor_version=$(echo "$VERSION" | grep -oP 'v\d+\.\d+')
    build_cmd+=(-t "${REGISTRY}/${service}:${minor_version}")

    # Extract major version (v1.2.3 -> v1)
    local major_version
    major_version=$(echo "$VERSION" | grep -oP 'v\d+')
    build_cmd+=(-t "${REGISTRY}/${service}:${major_version}")
  fi

  build_cmd+=("$context")

  # Execute build
  if "${build_cmd[@]}"; then
    log "✓ Successfully built $service"

    # Scan for vulnerabilities if requested
    if [[ "$SCAN" == "true" ]]; then
      scan_image "$image_name"
    fi

    # Push if requested
    if [[ "$PUSH" == "true" ]]; then
      push_image "$service"
    fi

    return 0
  else
    error "✗ Failed to build $service"
    return 1
  fi
}

# Function: Scan image for vulnerabilities
scan_image() {
  local image=$1

  log "Scanning $image for vulnerabilities..."

  if trivy image --severity CRITICAL,HIGH --exit-code 1 "$image"; then
    log "✓ No Critical/High vulnerabilities found in $image"
  else
    error "✗ Critical or High vulnerabilities found in $image"
    return 1
  fi
}

# Function: Push image to registry
push_image() {
  local service=$1

  log "Pushing ${service} images to registry..."

  # Push all tags for this service
  docker images --format "{{.Repository}}:{{.Tag}}" | grep "^${REGISTRY}/${service}:" | while read -r image; do
    log "  Pushing $image..."
    if docker push "$image"; then
      log "  ✓ Pushed $image"
    else
      error "  ✗ Failed to push $image"
      return 1
    fi
  done
}

# Function: Build all services
build_all_services() {
  local services_to_build=()

  if [[ -z "$SERVICES" ]]; then
    services_to_build=("${ALL_SERVICES[@]}")
  else
    IFS=',' read -ra services_to_build <<< "$SERVICES"
  fi

  local total=${#services_to_build[@]}
  local success=0
  local failed=0

  log "Building $total service(s): ${services_to_build[*]}"
  echo ""

  for service in "${services_to_build[@]}"; do
    # Validate service name
    if [[ ! " ${ALL_SERVICES[*]} " =~ " ${service} " ]]; then
      warn "Unknown service: $service (skipping)"
      ((failed++))
      continue
    fi

    if build_service "$service"; then
      ((success++))
    else
      ((failed++))
    fi
    echo ""
  done

  # Summary
  log "Build Summary:"
  log "  Total: $total"
  log "  Success: $success"
  log "  Failed: $failed"

  if [[ $failed -gt 0 ]]; then
    die "Some builds failed. Please check the output above."
  fi

  log "All builds completed successfully!"
}

# Main function
main() {
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      -v|--version)
        VERSION="$2"
        shift 2
        ;;
      -c|--commit-sha)
        COMMIT_SHA="$2"
        shift 2
        ;;
      -d|--build-date)
        BUILD_DATE="$2"
        shift 2
        ;;
      -r|--registry)
        REGISTRY="$2"
        shift 2
        ;;
      -s|--services)
        SERVICES="$2"
        shift 2
        ;;
      -p|--platform)
        PLATFORM="$2"
        shift 2
        ;;
      --push)
        PUSH=true
        shift
        ;;
      --scan)
        SCAN=true
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        error "Unknown option: $1"
        usage
        exit 1
        ;;
    esac
  done

  # Check dependencies
  check_dependencies

  # Validate and auto-detect metadata
  validate_version
  detect_git_metadata

  # Build services
  build_all_services
}

# Run main
main "$@"
