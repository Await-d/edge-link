#!/bin/bash
# ============================================
# compare-build-hashes.sh
# Purpose: Track and compare build hashes across versions
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

# Hash database
HASH_DB="${PROJECT_ROOT}/.build-hashes.json"

# Options
ACTION=""
SERVICE=""
VERSION=""
BINARY_PATH=""
LIST_ALL=false

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Track and compare build artifact hashes for reproducibility verification.

COMMANDS:
    record      Record hash for a new build
    compare     Compare hash with historical builds
    list        List all recorded hashes
    diff        Show differences between versions
    clean       Remove old entries

OPTIONS:
    --service SERVICE      Service name (required for record/compare)
    --version VERSION      Version tag (required for record)
    --binary PATH          Path to binary (required for record)
    --all                  Show all entries (for list command)
    --help                 Show this help message

EXAMPLES:
    # Record a build hash
    $0 record --service api-gateway --version v1.0.0 --binary ./api-gateway

    # Compare with previous builds
    $0 compare --service api-gateway --version v1.0.0

    # List all recorded hashes
    $0 list --all

    # Show differences between versions
    $0 diff --service api-gateway

DATABASE FORMAT:
    .build-hashes.json stores:
    {
      "service-name": {
        "version": {
          "commit": "git-sha",
          "timestamp": "ISO-8601",
          "sha256": "hash",
          "build_env": "environment-fingerprint",
          "reproducible": true/false
        }
      }
    }

EOF
    exit 1
}

# Parse arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        usage
    fi

    ACTION=$1
    shift

    while [[ $# -gt 0 ]]; do
        case $1 in
            --service)
                SERVICE="$2"
                shift 2
                ;;
            --version)
                VERSION="$2"
                shift 2
                ;;
            --binary)
                BINARY_PATH="$2"
                shift 2
                ;;
            --all)
                LIST_ALL=true
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
}

# Initialize hash database
init_db() {
    if [[ ! -f "$HASH_DB" ]]; then
        echo "{}" > "$HASH_DB"
    fi
}

# Get current git info
get_git_info() {
    local commit="unknown"
    local dirty=""

    if command -v git &> /dev/null && [[ -d "$PROJECT_ROOT/.git" ]]; then
        commit=$(git rev-parse HEAD 2>/dev/null || echo "unknown")
        if [[ -n "$(git status --porcelain)" ]]; then
            dirty="-dirty"
        fi
    fi

    echo "${commit}${dirty}"
}

# Get build environment fingerprint
get_env_fingerprint() {
    local fingerprint=""

    # Check if fingerprint file exists
    if [[ -f "${PROJECT_ROOT}/.build-environment-fingerprint" ]]; then
        fingerprint=$(grep "Overall Fingerprint:" "${PROJECT_ROOT}/.build-environment-fingerprint" | awk '{print $3}')
    else
        # Generate basic fingerprint
        fingerprint=$(echo "$(go version 2>/dev/null || echo 'go-unknown') $(docker --version 2>/dev/null || echo 'docker-unknown')" | sha256sum | awk '{print $1}')
    fi

    echo "$fingerprint"
}

# Record build hash
record_hash() {
    if [[ -z "$SERVICE" ]] || [[ -z "$VERSION" ]] || [[ -z "$BINARY_PATH" ]]; then
        echo -e "${RED}Error: --service, --version, and --binary are required for record${NC}"
        exit 1
    fi

    if [[ ! -f "$BINARY_PATH" ]]; then
        echo -e "${RED}Error: Binary not found at $BINARY_PATH${NC}"
        exit 1
    fi

    echo -e "${BLUE}Recording build hash...${NC}"

    # Calculate hash
    local sha256=$(sha256sum "$BINARY_PATH" | awk '{print $1}')
    local commit=$(get_git_info)
    local timestamp=$(date -u +"%Y-%m-%dT%H:%M:%SZ")
    local env_fingerprint=$(get_env_fingerprint)

    echo "  Service: $SERVICE"
    echo "  Version: $VERSION"
    echo "  SHA256: $sha256"
    echo "  Commit: $commit"
    echo "  Timestamp: $timestamp"
    echo "  Environment: ${env_fingerprint:0:16}..."

    # Update database
    local temp_db=$(mktemp)
    jq --arg service "$SERVICE" \
       --arg version "$VERSION" \
       --arg sha256 "$sha256" \
       --arg commit "$commit" \
       --arg timestamp "$timestamp" \
       --arg env "$env_fingerprint" \
       '.[$service] = (.[$service] // {}) |
        .[$service][$version] = {
          "commit": $commit,
          "timestamp": $timestamp,
          "sha256": $sha256,
          "build_env": $env,
          "reproducible": null
        }' "$HASH_DB" > "$temp_db"

    mv "$temp_db" "$HASH_DB"

    echo -e "${GREEN}✅ Hash recorded successfully${NC}"
}

# Compare hash
compare_hash() {
    if [[ -z "$SERVICE" ]] || [[ -z "$VERSION" ]]; then
        echo -e "${RED}Error: --service and --version are required for compare${NC}"
        exit 1
    fi

    echo -e "${BLUE}Comparing build hashes for $SERVICE $VERSION...${NC}"
    echo ""

    # Get recorded hash
    local recorded_sha=$(jq -r --arg service "$SERVICE" --arg version "$VERSION" \
        '.[$service][$version].sha256 // "not-found"' "$HASH_DB")

    if [[ "$recorded_sha" == "not-found" ]]; then
        echo -e "${YELLOW}No recorded hash found for $SERVICE $VERSION${NC}"
        echo "Use 'record' command to store the first hash"
        exit 1
    fi

    echo "Recorded hash: $recorded_sha"
    echo ""

    # Compare with other versions
    local versions=$(jq -r --arg service "$SERVICE" '.[$service] | keys[]' "$HASH_DB" 2>/dev/null || echo "")

    if [[ -z "$versions" ]]; then
        echo -e "${YELLOW}No other versions found for comparison${NC}"
        exit 0
    fi

    echo "Comparison with other versions:"
    echo ""

    local has_match=false
    while IFS= read -r other_version; do
        if [[ "$other_version" == "$VERSION" ]]; then
            continue
        fi

        local other_sha=$(jq -r --arg service "$SERVICE" --arg version "$other_version" \
            '.[$service][$version].sha256' "$HASH_DB")
        local other_commit=$(jq -r --arg service "$SERVICE" --arg version "$other_version" \
            '.[$service][$version].commit' "$HASH_DB")

        if [[ "$recorded_sha" == "$other_sha" ]]; then
            echo -e "  ${GREEN}✅ $other_version${NC} (commit: ${other_commit:0:8})"
            echo -e "     ${GREEN}IDENTICAL HASH - Build is reproducible!${NC}"
            has_match=true
        else
            echo -e "  ${YELLOW}⚠️  $other_version${NC} (commit: ${other_commit:0:8})"
            echo -e "     Hash: $other_sha"
            echo -e "     ${YELLOW}DIFFERENT - May indicate build changes${NC}"
        fi
        echo ""
    done <<< "$versions"

    if [[ "$has_match" == "true" ]]; then
        # Update reproducible flag
        local temp_db=$(mktemp)
        jq --arg service "$SERVICE" \
           --arg version "$VERSION" \
           '.[$service][$version].reproducible = true' "$HASH_DB" > "$temp_db"
        mv "$temp_db" "$HASH_DB"
    fi
}

# List all hashes
list_hashes() {
    echo -e "${BLUE}Build Hash Database${NC}"
    echo -e "${BLUE}==================${NC}"
    echo ""

    if [[ ! -f "$HASH_DB" ]] || [[ $(jq 'length' "$HASH_DB") -eq 0 ]]; then
        echo "No hashes recorded yet"
        exit 0
    fi

    local services=$(jq -r 'keys[]' "$HASH_DB")

    while IFS= read -r service; do
        echo -e "${BLUE}Service: $service${NC}"

        local versions=$(jq -r --arg service "$service" '.[$service] | keys[]' "$HASH_DB")

        while IFS= read -r version; do
            local sha256=$(jq -r --arg service "$service" --arg version "$version" \
                '.[$service][$version].sha256' "$HASH_DB")
            local commit=$(jq -r --arg service "$service" --arg version "$version" \
                '.[$service][$version].commit' "$HASH_DB")
            local timestamp=$(jq -r --arg service "$service" --arg version "$version" \
                '.[$service][$version].timestamp' "$HASH_DB")
            local reproducible=$(jq -r --arg service "$service" --arg version "$version" \
                '.[$service][$version].reproducible' "$HASH_DB")

            echo "  Version: $version"
            echo "    Commit: $commit"
            echo "    Timestamp: $timestamp"
            echo "    SHA256: $sha256"

            if [[ "$reproducible" == "true" ]]; then
                echo -e "    Status: ${GREEN}✅ Reproducible${NC}"
            elif [[ "$reproducible" == "false" ]]; then
                echo -e "    Status: ${RED}❌ Not reproducible${NC}"
            else
                echo "    Status: Unknown (not compared)"
            fi

            if [[ "$LIST_ALL" == "true" ]]; then
                local env=$(jq -r --arg service "$service" --arg version "$version" \
                    '.[$service][$version].build_env' "$HASH_DB")
                echo "    Build Env: ${env:0:16}..."
            fi

            echo ""
        done <<< "$versions"

    done <<< "$services"
}

# Show diff between versions
show_diff() {
    if [[ -z "$SERVICE" ]]; then
        echo -e "${RED}Error: --service is required for diff${NC}"
        exit 1
    fi

    echo -e "${BLUE}Hash Changes for $SERVICE${NC}"
    echo -e "${BLUE}========================${NC}"
    echo ""

    local versions=$(jq -r --arg service "$SERVICE" '.[$service] | keys[]' "$HASH_DB" 2>/dev/null || echo "")

    if [[ -z "$versions" ]]; then
        echo "No versions found for $SERVICE"
        exit 0
    fi

    local prev_sha=""
    local prev_version=""

    while IFS= read -r version; do
        local sha256=$(jq -r --arg service "$SERVICE" --arg version "$version" \
            '.[$service][$version].sha256' "$HASH_DB")
        local timestamp=$(jq -r --arg service "$SERVICE" --arg version "$version" \
            '.[$service][$version].timestamp' "$HASH_DB")

        echo "Version: $version ($timestamp)"
        echo "  SHA256: $sha256"

        if [[ -n "$prev_sha" ]]; then
            if [[ "$sha256" == "$prev_sha" ]]; then
                echo -e "  ${GREEN}✅ Same as $prev_version (reproducible)${NC}"
            else
                echo -e "  ${YELLOW}⚠️  Different from $prev_version${NC}"
            fi
        fi

        echo ""

        prev_sha="$sha256"
        prev_version="$version"
    done <<< "$versions"
}

# Clean old entries
clean_db() {
    echo -e "${BLUE}Cleaning old entries...${NC}"

    # Keep only last 10 versions per service
    local temp_db=$(mktemp)

    jq 'with_entries(.value = (.value | to_entries | sort_by(.value.timestamp) | reverse | .[0:10] | from_entries))' \
        "$HASH_DB" > "$temp_db"

    mv "$temp_db" "$HASH_DB"

    echo -e "${GREEN}✅ Cleanup complete${NC}"
}

# Main execution
main() {
    parse_args "$@"

    cd "$PROJECT_ROOT"
    init_db

    case "$ACTION" in
        record)
            record_hash
            ;;
        compare)
            compare_hash
            ;;
        list)
            list_hashes
            ;;
        diff)
            show_diff
            ;;
        clean)
            clean_db
            ;;
        *)
            echo -e "${RED}Unknown action: $ACTION${NC}"
            usage
            ;;
    esac
}

main "$@"
