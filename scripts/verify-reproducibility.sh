#!/bin/bash
# ============================================
# verify-reproducibility.sh
# Purpose: Verify build reproducibility by comparing binaries from multiple builds
# ============================================

set -euo pipefail

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
SERVICES=()
ALL_SERVICES=false
REPORT_MODE=false
OUTPUT_DIR="/tmp/reproducibility-test-$$"
REPORT_FILE="reproducibility-report.txt"

# Available services
AVAILABLE_SERVICES=(
    "api-gateway"
    "alert-service"
    "background-worker"
)

# Critical services for quick check
CRITICAL_SERVICES=(
    "api-gateway"
    "alert-service"
    "background-worker"
)

# Usage information
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Verify build reproducibility for Edge-Link services.

OPTIONS:
    --service SERVICE    Test specific service (can be repeated)
    --all               Test all available services
    --report            Generate detailed report
    --output DIR        Output directory for test artifacts (default: /tmp/reproducibility-test-PID)
    --help              Show this help message

EXAMPLES:
    $0 --service api-gateway
    $0 --all
    $0 --all --report
    $0 --service api-gateway --service alert-service

AVAILABLE SERVICES:
    ${AVAILABLE_SERVICES[*]}

EOF
    exit 1
}

# Parse command line arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --service)
                SERVICES+=("$2")
                shift 2
                ;;
            --all)
                ALL_SERVICES=true
                shift
                ;;
            --report)
                REPORT_MODE=true
                shift
                ;;
            --output)
                OUTPUT_DIR="$2"
                shift 2
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
}

# Print section header
print_header() {
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}$1${NC}"
    echo -e "${BLUE}========================================${NC}"
}

# Print test result
print_result() {
    local service=$1
    local reproducible=$2
    local sha1=$3
    local sha2=$4
    local details=$5

    if [[ "$reproducible" == "true" ]]; then
        echo -e "  ${GREEN}✅ REPRODUCIBLE${NC}"
        echo -e "  SHA256: ${sha1}"
    else
        echo -e "  ${RED}❌ NOT REPRODUCIBLE${NC}"
        echo -e "  Build 1: ${sha1}"
        echo -e "  Build 2: ${sha2}"
        if [[ -n "$details" ]]; then
            echo -e "  ${YELLOW}Details: ${details}${NC}"
        fi
    fi
}

# Build service in Docker
build_service() {
    local service=$1
    local build_num=$2
    local output_path=$3

    echo "  Building $service (build #$build_num)..."

    # Export build environment variables for reproducibility
    export SOURCE_DATE_EPOCH=0
    export BUILD_DATE="1970-01-01T00:00:00Z"
    export VERSION="test-reproducibility"
    export COMMIT_SHA="reproducibility-test"

    # Build using docker-compose
    docker-compose build --no-cache "$service" > "${OUTPUT_DIR}/build-${service}-${build_num}.log" 2>&1

    # Extract binary from container
    local container_name="edgelink-${service}-extract-${build_num}"
    docker create --name "$container_name" "edgelink/${service}:test-reproducibility" > /dev/null 2>&1 || \
    docker create --name "$container_name" "edgelink/${service}:v0.0.0-dev" > /dev/null 2>&1

    docker cp "${container_name}:/app/${service}" "$output_path" 2>/dev/null || {
        echo -e "    ${YELLOW}Warning: Could not extract binary from standard path${NC}"
        # Try alternative paths
        docker cp "${container_name}:/app/$(basename $service)" "$output_path" 2>/dev/null || true
    }

    docker rm "$container_name" > /dev/null 2>&1

    if [[ ! -f "$output_path" ]]; then
        echo -e "    ${RED}Failed to extract binary${NC}"
        return 1
    fi

    # Calculate SHA256
    local sha=$(sha256sum "$output_path" | awk '{print $1}')
    echo "    SHA256: $sha"
    echo "$sha"
}

# Check for non-deterministic factors in binary
check_binary_determinism() {
    local binary=$1
    local issues=""

    # Check for timestamps (common issue)
    if strings "$binary" | grep -E '[0-9]{4}-[0-9]{2}-[0-9]{2}T[0-9]{2}:[0-9]{2}:[0-9]{2}' > /dev/null 2>&1; then
        issues="${issues}Embedded timestamps found; "
    fi

    # Check for build paths (should be stripped by -trimpath)
    if strings "$binary" | grep -E '/home/|/root/|/tmp/' > /dev/null 2>&1; then
        issues="${issues}Build paths detected; "
    fi

    # Check for Go build ID (should be empty)
    if go tool buildid "$binary" 2>/dev/null | grep -v '^$' > /dev/null 2>&1; then
        issues="${issues}Build ID present; "
    fi

    echo "$issues"
}

# Test single service reproducibility
test_service() {
    local service=$1

    print_header "Testing: $service"

    local build1="${OUTPUT_DIR}/${service}-build1"
    local build2="${OUTPUT_DIR}/${service}-build2"

    # First build
    sha1=$(build_service "$service" 1 "$build1") || return 1

    # Small delay between builds
    sleep 2

    # Second build
    sha2=$(build_service "$service" 2 "$build2") || return 1

    # Compare
    echo ""
    echo "Comparing builds..."

    local reproducible="false"
    local details=""

    if [[ "$sha1" == "$sha2" ]]; then
        reproducible="true"
    else
        # Investigate differences
        details=$(check_binary_determinism "$build1")
    fi

    print_result "$service" "$reproducible" "$sha1" "$sha2" "$details"

    # Return status
    if [[ "$reproducible" == "true" ]]; then
        return 0
    else
        return 1
    fi
}

# Test in isolated Docker environment
test_isolated_build() {
    local service=$1

    print_header "Isolated Docker Build Test: $service"

    echo "Building in isolated container environment..."

    # Create temporary Dockerfile for isolated build
    local isolated_dockerfile="${OUTPUT_DIR}/Dockerfile.isolated"

    cat > "$isolated_dockerfile" << 'EOF'
FROM golang:1.21-alpine@sha256:2414035b086e3c42b99654c8b26e6f5b1b1598080d65fd03c7f499552ff4dc94

# Install build tools with exact versions
RUN apk add --no-cache \
    git=2.40.1-r0 \
    make=4.4.1-r1

WORKDIR /build

# Set reproducible build environment
ENV SOURCE_DATE_EPOCH=0
ENV CGO_ENABLED=0
ENV GOOS=linux
ENV GOARCH=amd64

COPY backend/go.mod backend/go.sum ./
RUN go mod download && go mod verify

COPY backend/ ./

ARG SERVICE
RUN go build \
    -trimpath \
    -buildvcs=false \
    -ldflags="-w -s -buildid=" \
    -o /output/binary \
    ./cmd/${SERVICE}
EOF

    # Build in isolated environment
    docker build \
        --no-cache \
        --build-arg SERVICE="$service" \
        -f "$isolated_dockerfile" \
        -t "isolated-build-${service}" \
        . > "${OUTPUT_DIR}/isolated-build-${service}.log" 2>&1

    # Extract binary
    local container_name="isolated-extract-${service}"
    docker create --name "$container_name" "isolated-build-${service}" > /dev/null 2>&1
    docker cp "${container_name}:/output/binary" "${OUTPUT_DIR}/${service}-isolated" 2>/dev/null
    docker rm "$container_name" > /dev/null 2>&1

    local sha=$(sha256sum "${OUTPUT_DIR}/${service}-isolated" | awk '{print $1}')
    echo "  Isolated build SHA256: $sha"
    echo "  ${GREEN}✅ Isolated build completed${NC}"
}

# Generate detailed report
generate_report() {
    print_header "Generating Report"

    local report="${OUTPUT_DIR}/${REPORT_FILE}"

    cat > "$report" << EOF
======================================
Build Reproducibility Report
======================================
Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")
Project: Edge-Link
Test ID: $$

Environment:
- OS: $(uname -s)
- Kernel: $(uname -r)
- Docker: $(docker --version)
- Go: $(go version 2>/dev/null || echo "N/A")

Test Results:
======================================

EOF

    for service in "${SERVICES[@]}"; do
        echo "Service: $service" >> "$report"
        if [[ -f "${OUTPUT_DIR}/${service}-build1" ]] && [[ -f "${OUTPUT_DIR}/${service}-build2" ]]; then
            local sha1=$(sha256sum "${OUTPUT_DIR}/${service}-build1" | awk '{print $1}')
            local sha2=$(sha256sum "${OUTPUT_DIR}/${service}-build2" | awk '{print $1}')

            echo "  Build 1 SHA256: $sha1" >> "$report"
            echo "  Build 2 SHA256: $sha2" >> "$report"

            if [[ "$sha1" == "$sha2" ]]; then
                echo "  Status: ✅ REPRODUCIBLE" >> "$report"
            else
                echo "  Status: ❌ NOT REPRODUCIBLE" >> "$report"
                local details=$(check_binary_determinism "${OUTPUT_DIR}/${service}-build1")
                if [[ -n "$details" ]]; then
                    echo "  Issues: $details" >> "$report"
                fi
            fi
        else
            echo "  Status: ⚠️  BUILD FAILED" >> "$report"
        fi
        echo "" >> "$report"
    done

    echo "Report saved to: $report"
    echo ""
    cat "$report"
}

# Cleanup
cleanup() {
    if [[ -d "$OUTPUT_DIR" ]] && [[ "$OUTPUT_DIR" =~ ^/tmp/ ]]; then
        echo ""
        echo "Cleaning up temporary files..."
        # rm -rf "$OUTPUT_DIR"
        echo "Test artifacts kept at: $OUTPUT_DIR"
    fi
}

# Main execution
main() {
    parse_args "$@"

    # Set services to test
    if [[ "$ALL_SERVICES" == "true" ]]; then
        SERVICES=("${AVAILABLE_SERVICES[@]}")
    fi

    if [[ ${#SERVICES[@]} -eq 0 ]]; then
        echo -e "${RED}Error: No services specified${NC}"
        usage
    fi

    print_header "Edge-Link Build Reproducibility Test"
    echo "Services to test: ${SERVICES[*]}"
    echo "Output directory: $OUTPUT_DIR"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Track overall success
    local all_reproducible=true

    # Test each service
    for service in "${SERVICES[@]}"; do
        if ! test_service "$service"; then
            all_reproducible=false
        fi
        echo ""
    done

    # Generate report if requested
    if [[ "$REPORT_MODE" == "true" ]]; then
        generate_report
    fi

    # Summary
    print_header "Summary"
    if [[ "$all_reproducible" == "true" ]]; then
        echo -e "${GREEN}✅ All services are reproducible${NC}"
        cleanup
        exit 0
    else
        echo -e "${RED}❌ Some services are not reproducible${NC}"
        echo -e "${YELLOW}Check the report for details${NC}"
        cleanup
        exit 1
    fi
}

# Handle interrupts
trap cleanup EXIT INT TERM

# Run main
main "$@"
