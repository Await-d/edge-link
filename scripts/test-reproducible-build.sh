#!/bin/bash
# ============================================
# test-reproducible-build.sh
# Purpose: Build the same artifact twice and compare for reproducibility
# ============================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Project root
PROJECT_ROOT="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

# Options
TARGET=""
PLATFORM="linux/amd64"
OUTPUT_BASE="${PROJECT_ROOT}/build/reproducible"
BUILD1_DIR="${OUTPUT_BASE}/build1"
BUILD2_DIR="${OUTPUT_BASE}/build2"
REPORT_FILE="${OUTPUT_BASE}/report.txt"
VERBOSE=false

# Available targets
AVAILABLE_TARGETS=(
    "backend"
    "backend/api-gateway"
    "backend/alert-service"
    "backend/background-worker"
    "backend/device-service"
    "backend/topology-service"
    "backend/nat-coordinator"
    "desktop-client"
    "desktop-client/linux"
    "desktop-client/windows"
    "desktop-client/macos"
    "docker"
    "docker/api-gateway"
    "docker/alert-service"
    "docker/background-worker"
)

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build the same artifact twice with identical inputs and compare for reproducibility.

OPTIONS:
    --target TARGET      What to build (required)
    --platform PLATFORM  Target platform (default: linux/amd64)
    --verbose           Show detailed output
    --help              Show this help message

TARGETS:
    backend                    All backend services
    backend/SERVICE           Specific backend service
    desktop-client            All desktop clients
    desktop-client/PLATFORM   Desktop client for specific platform
    docker                    All Docker images
    docker/SERVICE            Specific Docker image

PLATFORMS (for cross-platform builds):
    linux/amd64, linux/arm64
    windows/amd64
    darwin/amd64, darwin/arm64

EXAMPLES:
    # Build all backend services
    $0 --target backend

    # Build specific service
    $0 --target backend/api-gateway

    # Build desktop client for ARM64
    $0 --target desktop-client --platform linux/arm64

    # Build Docker images
    $0 --target docker/api-gateway

OUTPUT:
    Build 1: ${BUILD1_DIR}/
    Build 2: ${BUILD2_DIR}/
    Report:  ${REPORT_FILE}

EXIT CODES:
    0 - Builds are identical (reproducible)
    1 - Builds differ (not reproducible) or build failed

EOF
    exit 1
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --target)
                TARGET="$2"
                shift 2
                ;;
            --platform)
                PLATFORM="$2"
                shift 2
                ;;
            --verbose)
                VERBOSE=true
                shift
                ;;
            --help)
                usage
                ;;
            *)
                echo -e "${RED}Unknown option: $1${NC}"
                usage
                ;;
        esac
    done

    if [[ -z "$TARGET" ]]; then
        echo -e "${RED}Error: --target is required${NC}"
        usage
    fi
}

# Print status
print_status() {
    local level=$1
    local message=$2

    case "$level" in
        info)
            echo -e "${BLUE}[INFO]${NC} $message"
            ;;
        ok)
            echo -e "${GREEN}[OK]${NC} $message"
            ;;
        warn)
            echo -e "${YELLOW}[WARN]${NC} $message"
            ;;
        error)
            echo -e "${RED}[ERROR]${NC} $message"
            ;;
    esac
}

# Setup build environment for reproducibility
setup_build_env() {
    print_status "info" "Setting up reproducible build environment..."

    # Fixed timestamp
    export SOURCE_DATE_EPOCH=0
    export BUILD_DATE="1970-01-01T00:00:00Z"
    export VERSION="reproducible"
    export COMMIT_SHA="reproducible"

    # Disable CGO for pure Go builds
    export CGO_ENABLED=0

    # Parse platform
    export GOOS=$(echo "$PLATFORM" | cut -d'/' -f1)
    export GOARCH=$(echo "$PLATFORM" | cut -d'/' -f2)

    print_status "ok" "Build environment configured"
    if [[ "$VERBOSE" == "true" ]]; then
        echo "  SOURCE_DATE_EPOCH: $SOURCE_DATE_EPOCH"
        echo "  BUILD_DATE: $BUILD_DATE"
        echo "  GOOS: $GOOS"
        echo "  GOARCH: $GOARCH"
    fi
}

# Build Go binary
build_go_binary() {
    local service=$1
    local output_dir=$2
    local build_num=$3

    print_status "info" "Building $service (build #$build_num)..."

    local cmd_path="${PROJECT_ROOT}/backend/cmd/$service"
    if [[ ! -d "$cmd_path" ]]; then
        print_status "error" "Service directory not found: $cmd_path"
        return 1
    fi

    mkdir -p "$output_dir"

    local output_binary="$output_dir/$service"

    cd "${PROJECT_ROOT}/backend"

    # Build with reproducible flags
    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-w -s -buildid= -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildDate=${BUILD_DATE}" \
        -o "$output_binary" \
        "./cmd/$service" || {
            print_status "error" "Build failed for $service"
            return 1
        }

    cd "$PROJECT_ROOT"

    if [[ ! -f "$output_binary" ]]; then
        print_status "error" "Binary not created: $output_binary"
        return 1
    fi

    local size=$(du -h "$output_binary" | cut -f1)
    print_status "ok" "Build #$build_num completed ($size)"

    echo "$output_binary"
}

# Build Docker image
build_docker_image() {
    local service=$1
    local build_num=$2

    print_status "info" "Building Docker image for $service (build #$build_num)..."

    local dockerfile="${PROJECT_ROOT}/infrastructure/docker/Dockerfile.$service"
    if [[ ! -f "$dockerfile" ]]; then
        print_status "error" "Dockerfile not found: $dockerfile"
        return 1
    fi

    local image_tag="edgelink/${service}:reproducible-test-${build_num}"

    docker build \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT_SHA="$COMMIT_SHA" \
        -f "$dockerfile" \
        -t "$image_tag" \
        "$PROJECT_ROOT" > /dev/null 2>&1 || {
            print_status "error" "Docker build failed for $service"
            return 1
        }

    print_status "ok" "Docker image built: $image_tag"
    echo "$image_tag"
}

# Extract binary from Docker image
extract_from_docker() {
    local image_tag=$1
    local service=$2
    local output_dir=$3

    mkdir -p "$output_dir"

    local container_name="extract-${service}-$$"
    local output_binary="$output_dir/$service"

    docker create --name "$container_name" "$image_tag" > /dev/null 2>&1 || {
        print_status "error" "Failed to create container from $image_tag"
        return 1
    }

    docker cp "$container_name:/app/$service" "$output_binary" 2>/dev/null || {
        print_status "warn" "Failed to extract from /app/$service, trying alternatives..."
        docker cp "$container_name:/usr/local/bin/$service" "$output_binary" 2>/dev/null || {
            docker rm "$container_name" > /dev/null 2>&1
            print_status "error" "Failed to extract binary from container"
            return 1
        }
    }

    docker rm "$container_name" > /dev/null 2>&1

    print_status "ok" "Binary extracted to $output_binary"
    echo "$output_binary"
}

# Build desktop client
build_desktop_client() {
    local platform=$1
    local output_dir=$2
    local build_num=$3

    print_status "info" "Building desktop client for $platform (build #$build_num)..."

    local client_path="${PROJECT_ROOT}/clients/desktop"
    if [[ ! -d "$client_path" ]]; then
        print_status "error" "Desktop client directory not found"
        return 1
    fi

    mkdir -p "$output_dir"

    local output_name="edgelink-client"
    if [[ "$GOOS" == "windows" ]]; then
        output_name="edgelink-client.exe"
    fi

    local output_binary="$output_dir/$output_name"

    cd "$client_path"

    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-w -s -buildid= -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildDate=${BUILD_DATE}" \
        -o "$output_binary" \
        ./cmd/client || {
            print_status "error" "Desktop client build failed"
            cd "$PROJECT_ROOT"
            return 1
        }

    cd "$PROJECT_ROOT"

    print_status "ok" "Desktop client built: $output_binary"
    echo "$output_binary"
}

# Compare two files
compare_files() {
    local file1=$1
    local file2=$2

    if [[ ! -f "$file1" ]] || [[ ! -f "$file2" ]]; then
        print_status "error" "Cannot compare: one or both files missing"
        return 1
    fi

    local sha1=$(sha256sum "$file1" | awk '{print $1}')
    local sha2=$(sha256sum "$file2" | awk '{print $1}')

    echo "$sha1|$sha2"
}

# Generate comparison report
generate_report() {
    local target=$1
    local build1_files=$2
    local build2_files=$3
    local identical=$4

    {
        echo "========================================"
        echo "Reproducibility Test Report"
        echo "========================================"
        echo ""
        echo "Target: $target"
        echo "Platform: $PLATFORM"
        echo "Date: $(date -u +"%Y-%m-%d %H:%M:%S UTC")"
        echo ""
        echo "Build Configuration:"
        echo "  SOURCE_DATE_EPOCH: $SOURCE_DATE_EPOCH"
        echo "  BUILD_DATE: $BUILD_DATE"
        echo "  VERSION: $VERSION"
        echo "  GOOS: $GOOS"
        echo "  GOARCH: $GOARCH"
        echo ""
        echo "========================================"
        echo "Build Comparison"
        echo "========================================"
        echo ""

        if [[ "$identical" == "true" ]]; then
            echo "Result: ✅ REPRODUCIBLE"
            echo ""
            echo "All artifacts are bit-for-bit identical."
            echo ""
            echo "SHA256 checksums:"
            for file in $build1_files; do
                local sha=$(sha256sum "$file" | awk '{print $1}')
                local name=$(basename "$file")
                echo "  $name: $sha"
            done
        else
            echo "Result: ❌ NOT REPRODUCIBLE"
            echo ""
            echo "Build artifacts differ:"
            echo ""

            # Compare each file
            local files1=($(echo "$build1_files"))
            local files2=($(echo "$build2_files"))

            for i in "${!files1[@]}"; do
                local file1="${files1[$i]}"
                local file2="${files2[$i]}"
                local name=$(basename "$file1")

                local sha1=$(sha256sum "$file1" | awk '{print $1}')
                local sha2=$(sha256sum "$file2" | awk '{print $1}')

                echo "  $name:"
                echo "    Build 1: $sha1"
                echo "    Build 2: $sha2"

                if [[ "$sha1" != "$sha2" ]]; then
                    echo "    Status: DIFFERENT"

                    # Size comparison
                    local size1=$(stat -f%z "$file1" 2>/dev/null || stat -c%s "$file1")
                    local size2=$(stat -f%z "$file2" 2>/dev/null || stat -c%s "$file2")
                    echo "    Size: $size1 vs $size2 bytes"

                    # Binary analysis if available
                    if command -v cmp &> /dev/null; then
                        local diff_offset=$(cmp -l "$file1" "$file2" 2>/dev/null | head -1 | awk '{print $1}' || echo "unknown")
                        echo "    First difference at byte: $diff_offset"
                    fi
                else
                    echo "    Status: IDENTICAL"
                fi
                echo ""
            done
        fi

        echo "========================================"
        echo "Build Artifacts"
        echo "========================================"
        echo ""
        echo "Build 1: $BUILD1_DIR"
        echo "Build 2: $BUILD2_DIR"
        echo ""
        echo "Detailed file listings:"
        echo ""
        echo "Build 1:"
        find "$BUILD1_DIR" -type f -exec ls -lh {} \; | awk '{print "  " $9, "(" $5 ")"}'
        echo ""
        echo "Build 2:"
        find "$BUILD2_DIR" -type f -exec ls -lh {} \; | awk '{print "  " $9, "(" $5 ")"}'
        echo ""

    } > "$REPORT_FILE"

    print_status "ok" "Report generated: $REPORT_FILE"
}

# Build target
build_target() {
    local target=$1
    local build_num=$2
    local output_dir=$3

    case "$target" in
        backend)
            # Build all backend services
            for service in api-gateway alert-service background-worker device-service topology-service nat-coordinator; do
                build_go_binary "$service" "$output_dir" "$build_num" || return 1
            done
            ;;
        backend/*)
            local service="${target#backend/}"
            build_go_binary "$service" "$output_dir" "$build_num" || return 1
            ;;
        desktop-client)
            build_desktop_client "$PLATFORM" "$output_dir" "$build_num" || return 1
            ;;
        desktop-client/*)
            local platform="${target#desktop-client/}"
            GOOS="$platform"
            build_desktop_client "$platform" "$output_dir" "$build_num" || return 1
            ;;
        docker)
            # Build all Docker images and extract
            for service in api-gateway alert-service background-worker; do
                local image=$(build_docker_image "$service" "$build_num") || return 1
                extract_from_docker "$image" "$service" "$output_dir" || return 1
            done
            ;;
        docker/*)
            local service="${target#docker/}"
            local image=$(build_docker_image "$service" "$build_num") || return 1
            extract_from_docker "$image" "$service" "$output_dir" || return 1
            ;;
        *)
            print_status "error" "Unknown target: $target"
            return 1
            ;;
    esac
}

# Main execution
main() {
    parse_args "$@"

    cd "$PROJECT_ROOT"

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Reproducible Build Test${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""
    echo "Target: $TARGET"
    echo "Platform: $PLATFORM"
    echo ""

    # Clean previous builds
    print_status "info" "Cleaning previous build artifacts..."
    rm -rf "$OUTPUT_BASE"
    mkdir -p "$BUILD1_DIR" "$BUILD2_DIR"

    # Setup environment
    setup_build_env

    # First build
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}First Build${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    build_target "$TARGET" 1 "$BUILD1_DIR" || {
        print_status "error" "First build failed"
        exit 1
    }

    # Small delay to ensure timestamp differences would appear if not handled
    sleep 1

    # Second build
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Second Build${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    build_target "$TARGET" 2 "$BUILD2_DIR" || {
        print_status "error" "Second build failed"
        exit 1
    }

    # Compare builds
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Comparing Builds${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    local build1_files=$(find "$BUILD1_DIR" -type f | sort)
    local build2_files=$(find "$BUILD2_DIR" -type f | sort)

    # Check file counts match
    local count1=$(echo "$build1_files" | wc -l)
    local count2=$(echo "$build2_files" | wc -l)

    if [[ $count1 -ne $count2 ]]; then
        print_status "error" "Different number of files: $count1 vs $count2"
        generate_report "$TARGET" "$build1_files" "$build2_files" "false"
        exit 1
    fi

    print_status "info" "Comparing $count1 file(s)..."

    local all_identical=true
    local files1_arr=($build1_files)
    local files2_arr=($build2_files)

    for i in "${!files1_arr[@]}"; do
        local file1="${files1_arr[$i]}"
        local file2="${files2_arr[$i]}"
        local name=$(basename "$file1")

        local hashes=$(compare_files "$file1" "$file2")
        local sha1=$(echo "$hashes" | cut -d'|' -f1)
        local sha2=$(echo "$hashes" | cut -d'|' -f2)

        if [[ "$sha1" == "$sha2" ]]; then
            print_status "ok" "$name: IDENTICAL ($sha1)"
        else
            print_status "error" "$name: DIFFERENT"
            echo "  Build 1: $sha1"
            echo "  Build 2: $sha2"
            all_identical=false
        fi
    done

    # Generate report
    echo ""
    generate_report "$TARGET" "$build1_files" "$build2_files" "$all_identical"

    # Summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Summary${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    if [[ "$all_identical" == "true" ]]; then
        print_status "ok" "Build is REPRODUCIBLE ✅"
        echo ""
        echo "All artifacts are bit-for-bit identical."
        echo "Report: $REPORT_FILE"
        exit 0
    else
        print_status "error" "Build is NOT REPRODUCIBLE ❌"
        echo ""
        echo "Artifacts differ between builds."
        echo "Report: $REPORT_FILE"
        echo ""
        echo "Common causes:"
        echo "  - Timestamps embedded in binaries"
        echo "  - Build paths not stripped (-trimpath missing)"
        echo "  - Non-deterministic ordering"
        echo "  - Random values or UUIDs"
        exit 1
    fi
}

main "$@"
