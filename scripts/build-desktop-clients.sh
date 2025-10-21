#!/bin/bash
# EdgeLink Desktop Client Cross-Platform Build Script
# Purpose: Build EdgeLink desktop client binaries for multiple platforms
# Usage: ./scripts/build-desktop-clients.sh [options]

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
VERSION=""
PLATFORMS=""
OUTPUT_DIR=""
PARALLEL=false
VERBOSE=false

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CLIENT_DIR="$PROJECT_ROOT/clients/desktop"

# Supported platforms
ALL_PLATFORMS=(
  "linux/amd64"
  "linux/arm64"
  "windows/amd64"
  "windows/arm64"
  "darwin/amd64"
  "darwin/arm64"
)

# Function: Print usage
usage() {
  cat <<EOF
Usage: ${0##*/} [OPTIONS]

Build EdgeLink desktop client binaries for multiple platforms with version stamping.

OPTIONS:
  -v, --version VERSION        Version tag (required, e.g., v1.2.3 or v1.2.3-dev)
  -p, --platform PLATFORM(S)   Target platforms (comma-separated, default: all)
                               Supported: ${ALL_PLATFORMS[*]}
  -o, --output-dir DIR         Output directory (default: ./build/clients)
  -j, --parallel               Build platforms in parallel
      --verbose                Enable verbose output
  -h, --help                   Show this help message

EXAMPLES:
  # Build for all platforms
  ${0##*/} --version v1.2.3

  # Build for specific platforms
  ${0##*/} --version v1.2.3 --platform linux/amd64,darwin/arm64

  # Build with custom output directory
  ${0##*/} --version v1.2.3-dev --output-dir ./dist

  # Build in parallel for faster execution
  ${0##*/} --version v1.2.3 --parallel

PLATFORM FORMATS:
  OS/ARCH combinations where:
    OS:   linux, windows, darwin (macOS)
    ARCH: amd64 (x86_64), arm64 (aarch64)

OUTPUT:
  Binaries are named: edgelink-client-{os}-{arch}-{version}[.exe]
  Checksums: edgelink-client-checksums-{version}.txt

EOF
}

# Function: Log message
log() {
  echo -e "${GREEN}[INFO]${NC} $*"
}

# Function: Log debug
debug() {
  if [[ "$VERBOSE" == "true" ]]; then
    echo -e "${BLUE}[DEBUG]${NC} $*"
  fi
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
  local deps=("go" "git" "sha256sum")

  for cmd in "${deps[@]}"; do
    if ! command -v "$cmd" &> /dev/null; then
      die "Required command '$cmd' not found. Please install it first."
    fi
  done

  # Check Go version (requires 1.21+)
  local go_version
  go_version=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
  local go_major go_minor
  go_major=$(echo "$go_version" | cut -d. -f1)
  go_minor=$(echo "$go_version" | cut -d. -f2)

  if [[ "$go_major" -lt 1 ]] || { [[ "$go_major" -eq 1 ]] && [[ "$go_minor" -lt 21 ]]; }; then
    die "Go 1.21 or higher required (found: go${go_version})"
  fi

  debug "Go version: $(go version)"
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

# Function: Parse platforms
parse_platforms() {
  local platforms_to_build=()

  if [[ -z "$PLATFORMS" ]]; then
    platforms_to_build=("${ALL_PLATFORMS[@]}")
  else
    IFS=',' read -ra platforms_to_build <<< "$PLATFORMS"
  fi

  # Validate platform formats
  for platform in "${platforms_to_build[@]}"; do
    if [[ ! " ${ALL_PLATFORMS[*]} " =~ " ${platform} " ]]; then
      warn "Unknown platform: $platform (will attempt build anyway)"
    fi
  done

  echo "${platforms_to_build[@]}"
}

# Function: Setup output directory
setup_output_dir() {
  if [[ -z "$OUTPUT_DIR" ]]; then
    OUTPUT_DIR="$PROJECT_ROOT/build/clients"
  fi

  # Create output directory
  mkdir -p "$OUTPUT_DIR"

  log "Output directory: $OUTPUT_DIR"
}

# Function: Get Git metadata
get_git_metadata() {
  local commit_sha="unknown"
  local build_date
  build_date=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

  if git rev-parse --git-dir > /dev/null 2>&1; then
    commit_sha=$(git rev-parse --short=8 HEAD)
    debug "Git commit: $commit_sha"
  else
    warn "Not a git repository, using 'unknown' as commit SHA"
  fi

  echo "$commit_sha $build_date"
}

# Function: Build for single platform
build_platform() {
  local platform=$1
  local commit_sha=$2
  local build_date=$3

  # Parse OS and ARCH
  local goos goarch
  IFS='/' read -r goos goarch <<< "$platform"

  # Determine binary name
  local binary_name="edgelink-client-${goos}-${goarch}-${VERSION}"
  if [[ "$goos" == "windows" ]]; then
    binary_name="${binary_name}.exe"
  fi

  local output_path="$OUTPUT_DIR/$binary_name"

  log "Building for ${platform}..."
  debug "  Binary: $binary_name"
  debug "  Output: $output_path"

  # Build command with ldflags for version stamping
  local ldflags="-s -w"
  ldflags+=" -X main.Version=${VERSION}"
  ldflags+=" -X main.CommitSHA=${commit_sha}"
  ldflags+=" -X main.BuildDate=${build_date}"

  # Platform-specific optimizations
  local build_tags=""
  if [[ "$goos" == "linux" ]]; then
    build_tags="netgo"
  fi

  # Set environment variables
  export GOOS="$goos"
  export GOARCH="$goarch"
  export CGO_ENABLED=0

  # Execute build
  local build_cmd=(
    go build
    -trimpath
    -ldflags "$ldflags"
  )

  if [[ -n "$build_tags" ]]; then
    build_cmd+=(-tags "$build_tags")
  fi

  build_cmd+=(
    -o "$output_path"
    "$CLIENT_DIR/cmd/edgelink-client"
  )

  debug "Build command: ${build_cmd[*]}"

  if "${build_cmd[@]}" 2>&1 | while read -r line; do debug "$line"; done; then
    # Calculate checksum
    local checksum
    checksum=$(sha256sum "$output_path" | awk '{print $1}')

    # Get file size
    local file_size
    file_size=$(du -h "$output_path" | awk '{print $1}')

    log "  Successfully built ${platform}"
    log "    Size: ${file_size}"
    log "    SHA256: ${checksum}"

    # Write checksum to file
    echo "${checksum}  ${binary_name}" >> "$OUTPUT_DIR/edgelink-client-checksums-${VERSION}.txt"

    return 0
  else
    error "  Failed to build for ${platform}"
    return 1
  fi
}

# Function: Build all platforms
build_all_platforms() {
  local platforms=("$@")
  local total=${#platforms[@]}
  local success=0
  local failed=0

  # Get Git metadata
  local git_metadata
  git_metadata=$(get_git_metadata)
  local commit_sha build_date
  read -r commit_sha build_date <<< "$git_metadata"

  log "Building for ${total} platform(s): ${platforms[*]}"
  log "Version: $VERSION"
  log "Commit: $commit_sha"
  log "Build Date: $build_date"
  echo ""

  # Remove old checksum file if exists
  rm -f "$OUTPUT_DIR/edgelink-client-checksums-${VERSION}.txt"

  if [[ "$PARALLEL" == "true" ]]; then
    log "Building in parallel mode..."

    # Build platforms in parallel
    local pids=()
    for platform in "${platforms[@]}"; do
      build_platform "$platform" "$commit_sha" "$build_date" &
      pids+=($!)
    done

    # Wait for all builds to complete
    for pid in "${pids[@]}"; do
      if wait "$pid"; then
        ((success++))
      else
        ((failed++))
      fi
    done
  else
    # Build sequentially
    for platform in "${platforms[@]}"; do
      if build_platform "$platform" "$commit_sha" "$build_date"; then
        ((success++))
      else
        ((failed++))
      fi
      echo ""
    done
  fi

  # Summary
  echo ""
  log "========================================"
  log "Build Summary:"
  log "  Total: $total"
  log "  Success: $success"
  log "  Failed: $failed"
  log "========================================"

  if [[ $failed -gt 0 ]]; then
    die "Some builds failed. Please check the output above."
  fi

  # Display checksums
  if [[ -f "$OUTPUT_DIR/edgelink-client-checksums-${VERSION}.txt" ]]; then
    log ""
    log "Checksums saved to: $OUTPUT_DIR/edgelink-client-checksums-${VERSION}.txt"
    log "Checksum contents:"
    cat "$OUTPUT_DIR/edgelink-client-checksums-${VERSION}.txt" | while read -r line; do
      log "  $line"
    done
  fi

  log ""
  log "All builds completed successfully!"
  log "Binaries available in: $OUTPUT_DIR"
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
      -p|--platform)
        PLATFORMS="$2"
        shift 2
        ;;
      -o|--output-dir)
        OUTPUT_DIR="$2"
        shift 2
        ;;
      -j|--parallel)
        PARALLEL=true
        shift
        ;;
      --verbose)
        VERBOSE=true
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

  # Check if client directory exists
  if [[ ! -d "$CLIENT_DIR" ]]; then
    die "Client directory not found: $CLIENT_DIR"
  fi

  # Check dependencies
  check_dependencies

  # Validate version
  validate_version

  # Parse platforms
  local platforms
  IFS=' ' read -ra platforms <<< "$(parse_platforms)"

  # Setup output directory
  setup_output_dir

  # Build all platforms
  build_all_platforms "${platforms[@]}"
}

# Run main
main "$@"
