#!/bin/bash
# ============================================
# quick-reproducibility-check.sh
# Purpose: Fast reproducibility check for critical services
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

# Critical services to check
CRITICAL_SERVICES=(
    "api-gateway"
    "alert-service"
    "background-worker"
)

# Output
OUTPUT_DIR="/tmp/quick-repro-check-$$"
TIMEOUT=300  # 5 minutes max

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Quick reproducibility check for critical Edge-Link services.

This is a fast version of verify-reproducibility.sh that:
- Only checks critical services (api-gateway, alert-service, background-worker)
- Uses parallel builds where possible
- Has a timeout limit (default: 5 minutes)
- Provides simple pass/fail output

OPTIONS:
    --timeout SECONDS    Set timeout in seconds (default: 300)
    --verbose           Show detailed output
    --keep-artifacts    Keep build artifacts for inspection
    --help              Show this help message

EXIT CODES:
    0 - All critical services are reproducible
    1 - One or more services are not reproducible
    2 - Build failed or timeout

EXAMPLES:
    $0
    $0 --verbose
    $0 --timeout 600 --keep-artifacts

EOF
    exit 1
}

# Parse arguments
VERBOSE=false
KEEP_ARTIFACTS=false

while [[ $# -gt 0 ]]; do
    case $1 in
        --timeout)
            TIMEOUT="$2"
            shift 2
            ;;
        --verbose)
            VERBOSE=true
            shift
            ;;
        --keep-artifacts)
            KEEP_ARTIFACTS=true
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

# Print with verbosity control
log_verbose() {
    if [[ "$VERBOSE" == "true" ]]; then
        echo "$@"
    fi
}

# Build single service
build_service_quick() {
    local service=$1
    local build_num=$2
    local output_path=$3

    log_verbose "  Building $service (build #$build_num)..."

    # Set reproducible build environment
    export SOURCE_DATE_EPOCH=0
    export BUILD_DATE="1970-01-01T00:00:00Z"
    export VERSION="quick-check"
    export COMMIT_SHA="quick-check"

    # Build with timeout
    timeout "${TIMEOUT}" docker-compose build --no-cache "$service" > "${OUTPUT_DIR}/build-${service}-${build_num}.log" 2>&1 || {
        log_verbose "    ${RED}Build failed or timed out${NC}"
        return 1
    }

    # Extract binary
    local container_name="quick-check-${service}-${build_num}-$$"
    docker create --name "$container_name" "edgelink/${service}:quick-check" > /dev/null 2>&1 || \
    docker create --name "$container_name" "edgelink/${service}:v0.0.0-dev" > /dev/null 2>&1 || {
        log_verbose "    ${RED}Failed to create container${NC}"
        return 1
    }

    docker cp "${container_name}:/app/${service}" "$output_path" 2>/dev/null || {
        log_verbose "    ${YELLOW}Warning: Using alternative extraction${NC}"
        docker cp "${container_name}:/app/$(basename $service)" "$output_path" 2>/dev/null || {
            docker rm "$container_name" > /dev/null 2>&1
            return 1
        }
    }

    docker rm "$container_name" > /dev/null 2>&1

    if [[ ! -f "$output_path" ]]; then
        log_verbose "    ${RED}Binary not found${NC}"
        return 1
    fi

    # Calculate hash
    local sha=$(sha256sum "$output_path" | awk '{print $1}')
    log_verbose "    SHA256: $sha"
    echo "$sha"
    return 0
}

# Quick check for single service
quick_check_service() {
    local service=$1

    if [[ "$VERBOSE" == "true" ]]; then
        echo ""
        echo -e "${BLUE}Checking: $service${NC}"
    fi

    local build1="${OUTPUT_DIR}/${service}-build1"
    local build2="${OUTPUT_DIR}/${service}-build2"

    # First build
    sha1=$(build_service_quick "$service" 1 "$build1") || {
        echo -e "  ${RED}✗${NC} $service (build failed)"
        return 2
    }

    # Second build
    sha2=$(build_service_quick "$service" 2 "$build2") || {
        echo -e "  ${RED}✗${NC} $service (build failed)"
        return 2
    }

    # Compare
    if [[ "$sha1" == "$sha2" ]]; then
        if [[ "$VERBOSE" == "true" ]]; then
            echo -e "  ${GREEN}✅ REPRODUCIBLE${NC}"
            echo -e "  SHA256: $sha1"
        else
            echo -e "  ${GREEN}✓${NC} $service"
        fi
        return 0
    else
        if [[ "$VERBOSE" == "true" ]]; then
            echo -e "  ${RED}❌ NOT REPRODUCIBLE${NC}"
            echo -e "  Build 1: $sha1"
            echo -e "  Build 2: $sha2"
        else
            echo -e "  ${RED}✗${NC} $service (hashes differ)"
        fi
        return 1
    fi
}

# Cleanup
cleanup() {
    if [[ "$KEEP_ARTIFACTS" == "false" ]] && [[ -d "$OUTPUT_DIR" ]]; then
        log_verbose "Cleaning up temporary files..."
        rm -rf "$OUTPUT_DIR"
    else
        log_verbose "Artifacts kept at: $OUTPUT_DIR"
    fi
}

# Main execution
main() {
    cd "$PROJECT_ROOT"

    # Create output directory
    mkdir -p "$OUTPUT_DIR"

    # Header
    echo -e "${BLUE}Quick Reproducibility Check${NC}"
    echo -e "${BLUE}===========================${NC}"
    echo ""
    echo "Services: ${CRITICAL_SERVICES[*]}"
    echo "Timeout: ${TIMEOUT}s per service"
    echo ""

    if [[ "$VERBOSE" == "false" ]]; then
        echo "Checking critical services..."
    fi

    # Track results
    local total=0
    local passed=0
    local failed=0
    local errors=0

    # Check each service
    local start_time=$(date +%s)

    for service in "${CRITICAL_SERVICES[@]}"; do
        ((total++))

        quick_check_service "$service"
        local result=$?

        if [[ $result -eq 0 ]]; then
            ((passed++))
        elif [[ $result -eq 1 ]]; then
            ((failed++))
        else
            ((errors++))
        fi

        # Check overall timeout
        local current_time=$(date +%s)
        local elapsed=$((current_time - start_time))
        if [[ $elapsed -gt $((TIMEOUT * 2)) ]]; then
            echo ""
            echo -e "${RED}Overall timeout exceeded${NC}"
            break
        fi
    done

    local end_time=$(date +%s)
    local total_time=$((end_time - start_time))

    # Summary
    echo ""
    echo -e "${BLUE}===========================${NC}"
    echo -e "${BLUE}Summary${NC}"
    echo -e "${BLUE}===========================${NC}"
    echo "Total services: $total"
    echo -e "${GREEN}Reproducible: $passed${NC}"
    echo -e "${RED}Not reproducible: $failed${NC}"
    if [[ $errors -gt 0 ]]; then
        echo -e "${YELLOW}Build errors: $errors${NC}"
    fi
    echo "Total time: ${total_time}s"

    # Cleanup
    cleanup

    # Exit code
    if [[ $errors -gt 0 ]]; then
        echo ""
        echo -e "${RED}❌ Build failed for one or more services${NC}"
        exit 2
    elif [[ $failed -gt 0 ]]; then
        echo ""
        echo -e "${RED}❌ Some services are not reproducible${NC}"
        echo "Run './scripts/verify-reproducibility.sh --all --report' for details"
        exit 1
    else
        echo ""
        echo -e "${GREEN}✅ All critical services are reproducible${NC}"
        exit 0
    fi
}

# Handle interrupts
trap cleanup EXIT INT TERM

# Run main
main
