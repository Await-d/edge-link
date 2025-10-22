#!/bin/bash
# ============================================
# verify-build-environment.sh
# Purpose: Verify build environment consistency and generate fingerprint
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

# Output
FINGERPRINT_FILE="${PROJECT_ROOT}/.build-environment-fingerprint"
VERBOSE=false

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Verify build environment consistency for reproducible builds.

OPTIONS:
    --verbose           Show detailed information
    --fingerprint-only  Only generate fingerprint, skip checks
    --help             Show this help message

CHECKS:
    - Go version pinning
    - Node version pinning
    - Docker base image digests
    - Dependency lock files (go.sum, package-lock.json)
    - Use of 'latest' tags

OUTPUT:
    - Environment verification report
    - Build environment fingerprint (.build-environment-fingerprint)

EOF
    exit 1
}

# Parse arguments
parse_args() {
    while [[ $# -gt 0 ]]; do
        case $1 in
            --verbose)
                VERBOSE=true
                shift
                ;;
            --fingerprint-only)
                FINGERPRINT_ONLY=true
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

# Print check result
print_check() {
    local status=$1
    local message=$2

    if [[ "$status" == "ok" ]]; then
        echo -e "  ${GREEN}✅ $message${NC}"
    elif [[ "$status" == "warn" ]]; then
        echo -e "  ${YELLOW}⚠️  $message${NC}"
    else
        echo -e "  ${RED}❌ $message${NC}"
    fi
}

# Extract version from Dockerfile
extract_dockerfile_version() {
    local dockerfile=$1
    local arg_name=$2

    if [[ ! -f "$dockerfile" ]]; then
        echo "not-found"
        return 1
    fi

    local version=$(grep "^ARG ${arg_name}=" "$dockerfile" | head -1 | cut -d'=' -f2)
    echo "${version:-not-set}"
}

# Extract digest from Dockerfile
extract_dockerfile_digest() {
    local dockerfile=$1
    local pattern=$2

    if [[ ! -f "$dockerfile" ]]; then
        echo "not-found"
        return 1
    fi

    local digest=$(grep "$pattern" "$dockerfile" | grep -oE 'sha256:[a-f0-9]{64}' | head -1)
    echo "${digest:-not-set}"
}

# Check Go version consistency
check_go_version() {
    echo ""
    echo -e "${BLUE}Checking Go Version...${NC}"

    local issues=0

    # Check Dockerfiles
    for dockerfile in infrastructure/docker/Dockerfile.*; do
        if [[ ! -f "$dockerfile" ]]; then
            continue
        fi

        local service=$(basename "$dockerfile" | sed 's/Dockerfile\.//')
        local go_version=$(extract_dockerfile_version "$dockerfile" "GO_VERSION")
        local go_digest=$(extract_dockerfile_digest "$dockerfile" "GO_ALPINE_DIGEST")

        if [[ "$go_version" != "not-set" && "$go_version" != "not-found" ]]; then
            if [[ "$go_digest" != "not-set" && "$go_digest" != "not-found" ]]; then
                print_check "ok" "$service: Go $go_version (digest: ${go_digest:0:15}...)"
            else
                print_check "warn" "$service: Go $go_version (no digest pinning)"
                ((issues++))
            fi
        else
            print_check "error" "$service: Go version not pinned"
            ((issues++))
        fi
    done

    # Check go.mod
    if [[ -f "backend/go.mod" ]]; then
        local go_mod_version=$(grep "^go " backend/go.mod | awk '{print $2}')
        print_check "ok" "go.mod requires Go $go_mod_version"
    else
        print_check "error" "go.mod not found"
        ((issues++))
    fi

    return $issues
}

# Check Node version consistency
check_node_version() {
    echo ""
    echo -e "${BLUE}Checking Node Version...${NC}"

    local issues=0

    # Check frontend Dockerfile
    if [[ -f "frontend/Dockerfile" ]]; then
        local node_version=$(grep "FROM node:" frontend/Dockerfile | grep -oE '[0-9]+\.[0-9]+\.[0-9]+' | head -1)
        local node_digest=$(grep "FROM node:" frontend/Dockerfile | grep -oE 'sha256:[a-f0-9]{64}' | head -1)

        if [[ -n "$node_version" ]]; then
            if [[ -n "$node_digest" ]]; then
                print_check "ok" "Frontend: Node $node_version (digest: ${node_digest:0:15}...)"
            else
                print_check "warn" "Frontend: Node $node_version (no digest pinning)"
                ((issues++))
            fi
        else
            print_check "error" "Frontend: Node version not pinned"
            ((issues++))
        fi
    fi

    # Check package.json engines
    if [[ -f "frontend/package.json" ]]; then
        local engines_node=$(grep -A2 '"engines"' frontend/package.json | grep '"node"' | grep -oE '[0-9]+\.[0-9]+' | head -1)
        if [[ -n "$engines_node" ]]; then
            print_check "ok" "package.json specifies Node >=$engines_node"
        else
            print_check "warn" "package.json does not specify Node version"
        fi
    fi

    return $issues
}

# Extract ARG values from Dockerfile
extract_arg_value() {
    local dockerfile="$1"
    local arg_name="$2"

    grep "^ARG ${arg_name}=" "$dockerfile" | head -1 | cut -d'=' -f2 | tr -d '"'
}

# Expand variables in image string
expand_image_vars() {
    local image="$1"
    local dockerfile="$2"

    # Find all ${VAR} patterns
    while [[ $image =~ \$\{([A-Z_]+)\} ]]; do
        local var_name="${BASH_REMATCH[1]}"
        local var_value=$(extract_arg_value "$dockerfile" "$var_name")

        if [ -z "$var_value" ]; then
            # If not found, keep original
            break
        fi

        image="${image//\$\{${var_name}\}/${var_value}}"
    done

    echo "$image"
}

# Check Docker base images
check_base_images() {
    echo ""
    echo -e "${BLUE}Checking Docker Base Images...${NC}"

    local issues=0

    # Check all Dockerfiles
    local dockerfiles=$(find . -name "Dockerfile*" -type f 2>/dev/null)

    for dockerfile in $dockerfiles; do
        local from_lines=$(grep "^FROM " "$dockerfile" || true)

        while IFS= read -r line; do
            if [[ -z "$line" ]]; then
                continue
            fi

            # Extract image reference
            local image=$(echo "$line" | awk '{print $2}')

            # Check for digest pinning
            if echo "$line" | grep -q "sha256:"; then
                local image_display=$(echo "$image" | cut -d'@' -f1)

                # Expand ARG variables for display
                if [[ $image_display =~ \$\{ ]]; then
                    image_display=$(expand_image_vars "$image_display" "$dockerfile")
                fi

                print_check "ok" "$(basename $dockerfile): $image_display (digest pinned)"
            else
                # Expand ARG variables before checking
                if [[ $image =~ \$\{ ]]; then
                    image=$(expand_image_vars "$image" "$dockerfile")
                fi

                if [[ "$image" =~ :latest$ ]] || [[ ! "$image" =~ : ]]; then
                    print_check "error" "$(basename $dockerfile): $image (using 'latest' or no tag)"
                    ((issues++))
                else
                    print_check "warn" "$(basename $dockerfile): $image (no digest pinning)"
                    ((issues++))
                fi
            fi
        done <<< "$from_lines"
    done

    return $issues
}

# Check dependency lock files
check_dependency_locks() {
    echo ""
    echo -e "${BLUE}Checking Dependency Lock Files...${NC}"

    local issues=0

    # Check go.sum
    if [[ -f "backend/go.sum" ]]; then
        local go_sum_lines=$(wc -l < backend/go.sum)
        print_check "ok" "go.sum present ($go_sum_lines entries)"

        # Verify go.sum is up to date
        if command -v go &> /dev/null; then
            cd backend
            if go mod verify &> /dev/null; then
                print_check "ok" "go.sum verified successfully"
            else
                print_check "error" "go.sum verification failed"
                ((issues++))
            fi
            cd ..
        fi
    else
        print_check "error" "go.sum not found"
        ((issues++))
    fi

    # Check frontend lock files
    if [[ -f "frontend/pnpm-lock.yaml" ]]; then
        print_check "ok" "pnpm-lock.yaml present"
    elif [[ -f "frontend/package-lock.json" ]]; then
        print_check "ok" "package-lock.json present"
    elif [[ -f "frontend/yarn.lock" ]]; then
        print_check "ok" "yarn.lock present"
    else
        print_check "warn" "No frontend lock file found"
        ((issues++))
    fi

    return $issues
}

# Check for 'latest' tags in compose files
check_compose_files() {
    echo ""
    echo -e "${BLUE}Checking Docker Compose Files...${NC}"

    local issues=0

    local compose_files=$(find . -name "docker-compose*.yml" -o -name "docker-compose*.yaml" 2>/dev/null)

    for compose_file in $compose_files; do
        if [[ ! -f "$compose_file" ]]; then
            continue
        fi

        # Check for 'latest' tag
        if grep -E 'image:.*:latest' "$compose_file" > /dev/null 2>&1; then
            print_check "warn" "$(basename $compose_file): Contains 'latest' tag"
            if [[ "$VERBOSE" == "true" ]]; then
                grep -n 'image:.*:latest' "$compose_file" | sed 's/^/      /'
            fi
            ((issues++))
        else
            print_check "ok" "$(basename $compose_file): No 'latest' tags found"
        fi

        # Check for version pinning with digests
        local digest_count=$(grep -c '@sha256:' "$compose_file" || echo 0)
        if [[ $digest_count -gt 0 ]]; then
            print_check "ok" "$(basename $compose_file): $digest_count images use digest pinning"
        fi
    done

    return $issues
}

# Generate build environment fingerprint
generate_fingerprint() {
    echo ""
    echo -e "${BLUE}Generating Build Environment Fingerprint...${NC}"

    local fingerprint_data=""

    # Collect versions
    fingerprint_data+="# Build Environment Fingerprint\n"
    fingerprint_data+="Generated: $(date -u +"%Y-%m-%dT%H:%M:%SZ")\n"
    fingerprint_data+="\n"

    # Go version
    if command -v go &> /dev/null; then
        fingerprint_data+="Go Version: $(go version)\n"
    fi

    # Node version
    if command -v node &> /dev/null; then
        fingerprint_data+="Node Version: $(node --version)\n"
    fi

    # Docker version
    if command -v docker &> /dev/null; then
        fingerprint_data+="Docker Version: $(docker --version)\n"
    fi

    # Git commit
    if command -v git &> /dev/null && [[ -d .git ]]; then
        fingerprint_data+="Git Commit: $(git rev-parse HEAD)\n"
        fingerprint_data+="Git Status: $(git status --porcelain | wc -l) modified files\n"
    fi

    # Dockerfile digests
    fingerprint_data+="\nDockerfile Base Images:\n"
    for dockerfile in infrastructure/docker/Dockerfile.*; do
        if [[ ! -f "$dockerfile" ]]; then
            continue
        fi
        local service=$(basename "$dockerfile" | sed 's/Dockerfile\.//')
        local digests=$(grep '@sha256:' "$dockerfile" | grep -oE 'sha256:[a-f0-9]{64}' || true)
        fingerprint_data+="  $service:\n"
        while IFS= read -r digest; do
            fingerprint_data+="    - $digest\n"
        done <<< "$digests"
    done

    # Dependency checksums
    fingerprint_data+="\nDependency Checksums:\n"
    if [[ -f "backend/go.sum" ]]; then
        local go_sum_hash=$(sha256sum backend/go.sum | awk '{print $1}')
        fingerprint_data+="  go.sum: $go_sum_hash\n"
    fi
    if [[ -f "frontend/pnpm-lock.yaml" ]]; then
        local pnpm_hash=$(sha256sum frontend/pnpm-lock.yaml | awk '{print $1}')
        fingerprint_data+="  pnpm-lock.yaml: $pnpm_hash\n"
    fi
    if [[ -f "frontend/package-lock.json" ]]; then
        local npm_hash=$(sha256sum frontend/package-lock.json | awk '{print $1}')
        fingerprint_data+="  package-lock.json: $npm_hash\n"
    fi

    # Calculate overall fingerprint
    local overall_fingerprint=$(echo -e "$fingerprint_data" | sha256sum | awk '{print $1}')
    fingerprint_data+="\nOverall Fingerprint: $overall_fingerprint\n"

    # Save to file
    echo -e "$fingerprint_data" > "$FINGERPRINT_FILE"

    echo -e "${GREEN}Fingerprint saved to: $FINGERPRINT_FILE${NC}"
    echo -e "Overall Fingerprint: ${BLUE}$overall_fingerprint${NC}"

    if [[ "$VERBOSE" == "true" ]]; then
        echo ""
        cat "$FINGERPRINT_FILE"
    fi
}

# Main execution
main() {
    parse_args "$@"

    cd "$PROJECT_ROOT"

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Build Environment Verification${NC}"
    echo -e "${BLUE}========================================${NC}"

    local total_issues=0

    # Run checks
    check_go_version || { local rc=$?; total_issues=$((total_issues + rc)); }
    check_node_version || { local rc=$?; total_issues=$((total_issues + rc)); }
    check_base_images || { local rc=$?; total_issues=$((total_issues + rc)); }
    check_dependency_locks || { local rc=$?; total_issues=$((total_issues + rc)); }
    check_compose_files || { local rc=$?; total_issues=$((total_issues + rc)); }

    # Generate fingerprint
    generate_fingerprint

    # Summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Summary${NC}"
    echo -e "${BLUE}========================================${NC}"

    if [[ $total_issues -eq 0 ]]; then
        echo -e "${GREEN}✅ All checks passed${NC}"
        echo -e "${GREEN}Build environment is properly configured for reproducible builds${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠️  Found $total_issues issue(s)${NC}"
        echo -e "${YELLOW}Please review the warnings and errors above${NC}"
        exit 1
    fi
}

main "$@"
