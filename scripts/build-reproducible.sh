#!/bin/bash
# ============================================
# build-reproducible.sh
# Purpose: Build services with reproducible build parameters
# ============================================

set -euo pipefail

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

# Default values
SERVICE=""
OUTPUT_DIR="./bin"
REPRODUCIBLE_MODE=true
RECORD_HASH=false

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Build Edge-Link services with reproducible build parameters.

OPTIONS:
    --service SERVICE       Service to build (required)
    --output DIR           Output directory for binary (default: ./bin)
    --normal-mode          Disable reproducible mode (use current timestamp)
    --record-hash          Record build hash in database
    --help                 Show this help message

SERVICES:
    api-gateway, alert-service, background-worker

REPRODUCIBLE BUILD SETTINGS:
    - SOURCE_DATE_EPOCH=0 (Unix epoch)
    - BUILD_DATE=1970-01-01T00:00:00Z (fixed timestamp)
    - CGO_ENABLED=0 (pure Go, no C dependencies)
    - -trimpath (remove build path information)
    - -buildvcs=false (no VCS info)
    - -buildid= (remove build ID)

EXAMPLES:
    # Build with reproducible settings
    $0 --service api-gateway

    # Build and record hash
    $0 --service api-gateway --record-hash

    # Build with normal settings (for production)
    $0 --service api-gateway --normal-mode --output ./dist

EOF
    exit 1
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --service)
                SERVICE="$2"
                shift 2
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
                ;;
            --normal-mode)
                REPRODUCIBLE_MODE=false
                shift
                ;;
            --record-hash)
                RECORD_HASH=true
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

    if [[ -z "$SERVICE" ]]; then
        echo -e "${RED}Error: --service is required${NC}"
        usage
    fi
}

# Set build environment
set_build_env() {
    if [[ "$REPRODUCIBLE_MODE" == "true" ]]; then
        # Reproducible build settings
        export SOURCE_DATE_EPOCH=0
        export BUILD_DATE="1970-01-01T00:00:00Z"
        export VERSION="${VERSION:-reproducible}"
        export COMMIT_SHA="${COMMIT_SHA:-reproducible}"

        echo -e "${BLUE}Reproducible Build Mode${NC}"
        echo "  SOURCE_DATE_EPOCH: $SOURCE_DATE_EPOCH"
        echo "  BUILD_DATE: $BUILD_DATE"
        echo "  VERSION: $VERSION"
        echo "  COMMIT_SHA: $COMMIT_SHA"
    else
        # Normal build settings
        export BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
        export VERSION=$(git describe --tags --always 2>/dev/null || echo "dev")
        export COMMIT_SHA=$(git rev-parse HEAD 2>/dev/null || echo "unknown")

        echo -e "${BLUE}Normal Build Mode${NC}"
        echo "  BUILD_DATE: $BUILD_DATE"
        echo "  VERSION: $VERSION"
        echo "  COMMIT_SHA: $COMMIT_SHA"
    fi

    # Common settings
    export CGO_ENABLED=0
    export GOOS=linux
    export GOARCH=amd64
}

# Build Go binary directly
build_go_binary() {
    echo ""
    echo -e "${BLUE}Building $SERVICE...${NC}"

    local cmd_path="backend/cmd/$SERVICE"
    if [[ ! -d "$cmd_path" ]]; then
        echo -e "${RED}Error: Service directory not found: $cmd_path${NC}"
        exit 1
    fi

    mkdir -p "$OUTPUT_DIR"

    local output_binary="$OUTPUT_DIR/$SERVICE"

    # Go to backend directory
    cd backend

    # Build with reproducible flags
    go build \
        -trimpath \
        -buildvcs=false \
        -ldflags="-w -s -buildid= -X main.Version=${VERSION} -X main.CommitSHA=${COMMIT_SHA} -X main.BuildDate=${BUILD_DATE}" \
        -o "../$output_binary" \
        "./cmd/$SERVICE"

    cd ..

    if [[ ! -f "$output_binary" ]]; then
        echo -e "${RED}Build failed: binary not created${NC}"
        exit 1
    fi

    # Display info
    local size=$(du -h "$output_binary" | cut -f1)
    local sha256=$(sha256sum "$output_binary" | awk '{print $1}')

    echo -e "${GREEN}✅ Build successful${NC}"
    echo "  Binary: $output_binary"
    echo "  Size: $size"
    echo "  SHA256: $sha256"

    # Verify build ID is removed
    if command -v go &> /dev/null; then
        local build_id=$(go tool buildid "$output_binary" 2>/dev/null || echo "")
        if [[ -z "$build_id" ]]; then
            echo -e "  ${GREEN}✅ Build ID removed${NC}"
        else
            echo -e "  ${YELLOW}⚠️  Build ID present: $build_id${NC}"
        fi
    fi

    # Record hash if requested
    if [[ "$RECORD_HASH" == "true" ]]; then
        echo ""
        echo "Recording build hash..."
        ./scripts/compare-build-hashes.sh record \
            --service "$SERVICE" \
            --version "$VERSION" \
            --binary "$output_binary"
    fi
}

# Build Docker image
build_docker_image() {
    echo ""
    echo -e "${BLUE}Building Docker image for $SERVICE...${NC}"

    local dockerfile="infrastructure/docker/Dockerfile.$SERVICE"
    if [[ ! -f "$dockerfile" ]]; then
        echo -e "${RED}Error: Dockerfile not found: $dockerfile${NC}"
        exit 1
    fi

    docker build \
        --build-arg BUILD_DATE="$BUILD_DATE" \
        --build-arg VERSION="$VERSION" \
        --build-arg COMMIT_SHA="$COMMIT_SHA" \
        -f "$dockerfile" \
        -t "edgelink/$SERVICE:$VERSION" \
        .

    echo -e "${GREEN}✅ Docker image built${NC}"
    echo "  Image: edgelink/$SERVICE:$VERSION"

    # Extract binary from image for hash recording
    if [[ "$RECORD_HASH" == "true" ]]; then
        echo ""
        echo "Extracting binary for hash recording..."

        local temp_container="temp-extract-$$"
        docker create --name "$temp_container" "edgelink/$SERVICE:$VERSION" > /dev/null 2>&1

        local temp_binary="/tmp/$SERVICE-$$"
        docker cp "$temp_container:/app/$SERVICE" "$temp_binary" 2>/dev/null || {
            echo -e "${YELLOW}Warning: Could not extract binary for hash recording${NC}"
            docker rm "$temp_container" > /dev/null 2>&1
            return
        }

        docker rm "$temp_container" > /dev/null 2>&1

        ./scripts/compare-build-hashes.sh record \
            --service "$SERVICE" \
            --version "$VERSION" \
            --binary "$temp_binary"

        rm -f "$temp_binary"
    fi
}

# Main execution
main() {
    parse_args "$@"

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Reproducible Build Script${NC}"
    echo -e "${BLUE}========================================${NC}"
    echo ""

    set_build_env

    # Build Go binary directly
    build_go_binary

    echo ""
    echo -e "${GREEN}========================================${NC}"
    echo -e "${GREEN}Build Complete${NC}"
    echo -e "${GREEN}========================================${NC}"
}

main "$@"
