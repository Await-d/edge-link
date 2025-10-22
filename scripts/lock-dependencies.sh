#!/bin/bash
# ============================================
# lock-dependencies.sh
# Purpose: Lock and verify dependency versions for reproducible builds
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
CHECK_ONLY=false
UPDATE_LOCKS=false
PIN_DOCKER=false
VERBOSE=false

# Usage
usage() {
    cat << EOF
Usage: $0 [OPTIONS]

Lock and verify dependency versions for reproducible builds.

OPTIONS:
    --check             Verify lock files are up-to-date (default)
    --update            Update lock files to latest compatible versions
    --pin-docker        Pin Docker base image digests in Dockerfiles
    --verbose           Show detailed output
    --help              Show this help message

OPERATIONS:
    1. Go Dependencies:
       - Run 'go mod tidy' to ensure go.mod is clean
       - Verify go.sum is up-to-date
       - Check for missing or outdated entries

    2. Node.js Dependencies:
       - Verify package-lock.json/pnpm-lock.yaml exists
       - Run 'npm ci' or 'pnpm install --frozen-lockfile' to verify
       - Check for package.json/lock file mismatches

    3. Docker Base Images:
       - Extract current base image tags
       - Fetch latest digests from registry
       - Update Dockerfiles with @sha256: digests
       - Preserve ARG-based version management

EXAMPLES:
    # Check if lock files are up-to-date
    $0 --check

    # Update all lock files
    $0 --update

    # Pin Docker base images to digests
    $0 --pin-docker

    # Update everything
    $0 --update --pin-docker --verbose

EXIT CODES:
    0 - All lock files are up-to-date
    1 - Lock files need updating or errors occurred

EOF
    exit 1
}

# Parse arguments
parse_args() {
    if [[ $# -eq 0 ]]; then
        CHECK_ONLY=true
    fi

    while [[ $# -gt 0 ]]; do
        case $1 in
            --check)
                CHECK_ONLY=true
                shift
                ;;
            --update)
                UPDATE_LOCKS=true
                CHECK_ONLY=false
                shift
                ;;
            --pin-docker)
                PIN_DOCKER=true
                shift
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
}

# Print status
print_status() {
    local status=$1
    local message=$2

    case "$status" in
        ok)
            echo -e "  ${GREEN}✅ $message${NC}"
            ;;
        warn)
            echo -e "  ${YELLOW}⚠️  $message${NC}"
            ;;
        error)
            echo -e "  ${RED}❌ $message${NC}"
            ;;
        info)
            echo -e "  ${BLUE}ℹ️  $message${NC}"
            ;;
    esac
}

# Check and update Go dependencies
check_go_deps() {
    echo ""
    echo -e "${BLUE}Checking Go Dependencies...${NC}"

    local issues=0

    if [[ ! -f "backend/go.mod" ]]; then
        print_status "error" "go.mod not found in backend/"
        return 1
    fi

    cd backend

    # Check if go.mod is tidy
    if [[ "$UPDATE_LOCKS" == "true" ]]; then
        print_status "info" "Running 'go mod tidy'..."
        go mod tidy
        print_status "ok" "go.mod tidied"
    else
        # Check if tidy would make changes
        cp go.mod go.mod.backup
        cp go.sum go.sum.backup 2>/dev/null || touch go.sum.backup

        go mod tidy -diff > /dev/null 2>&1 || {
            print_status "warn" "go.mod needs tidying (run with --update)"
            ((issues++))
        }

        # Restore backups
        mv go.mod.backup go.mod
        mv go.sum.backup go.sum
    fi

    # Verify go.sum
    print_status "info" "Verifying go.sum..."
    if go mod verify &> /dev/null; then
        print_status "ok" "go.sum verified successfully"
    else
        print_status "error" "go.sum verification failed"
        ((issues++))
    fi

    # Check for go.sum existence
    if [[ ! -f "go.sum" ]]; then
        print_status "error" "go.sum not found (run 'go mod download')"
        ((issues++))
    else
        local sum_entries=$(wc -l < go.sum)
        print_status "ok" "go.sum contains $sum_entries entries"
    fi

    # Download dependencies if updating
    if [[ "$UPDATE_LOCKS" == "true" ]]; then
        print_status "info" "Downloading Go modules..."
        go mod download
        print_status "ok" "Go modules downloaded"
    fi

    cd "$PROJECT_ROOT"

    return $issues
}

# Check and update Node.js dependencies
check_node_deps() {
    echo ""
    echo -e "${BLUE}Checking Node.js Dependencies...${NC}"

    local issues=0

    if [[ ! -d "frontend" ]]; then
        print_status "info" "Frontend directory not found, skipping"
        return 0
    fi

    if [[ ! -f "frontend/package.json" ]]; then
        print_status "error" "package.json not found in frontend/"
        return 1
    fi

    cd frontend

    # Detect package manager
    local pkg_manager=""
    local lock_file=""

    if [[ -f "pnpm-lock.yaml" ]]; then
        pkg_manager="pnpm"
        lock_file="pnpm-lock.yaml"
    elif [[ -f "package-lock.json" ]]; then
        pkg_manager="npm"
        lock_file="package-lock.json"
    elif [[ -f "yarn.lock" ]]; then
        pkg_manager="yarn"
        lock_file="yarn.lock"
    else
        print_status "warn" "No lock file found (run 'npm install' or 'pnpm install')"
        cd "$PROJECT_ROOT"
        return 1
    fi

    print_status "ok" "Using $pkg_manager with $lock_file"

    # Check if package manager is installed
    if ! command -v $pkg_manager &> /dev/null; then
        print_status "warn" "$pkg_manager not installed, skipping verification"
        cd "$PROJECT_ROOT"
        return 0
    fi

    # Update or verify lock file
    if [[ "$UPDATE_LOCKS" == "true" ]]; then
        print_status "info" "Updating lock file with $pkg_manager..."

        case "$pkg_manager" in
            npm)
                npm install --package-lock-only
                print_status "ok" "package-lock.json updated"
                ;;
            pnpm)
                pnpm install --lockfile-only
                print_status "ok" "pnpm-lock.yaml updated"
                ;;
            yarn)
                yarn install --mode update-lockfile
                print_status "ok" "yarn.lock updated"
                ;;
        esac
    else
        # Verify lock file is up-to-date
        print_status "info" "Verifying lock file..."

        case "$pkg_manager" in
            npm)
                if npm ci --dry-run &> /dev/null; then
                    print_status "ok" "package-lock.json is up-to-date"
                else
                    print_status "warn" "package-lock.json may be outdated (run with --update)"
                    ((issues++))
                fi
                ;;
            pnpm)
                if pnpm install --frozen-lockfile --dry-run &> /dev/null; then
                    print_status "ok" "pnpm-lock.yaml is up-to-date"
                else
                    print_status "warn" "pnpm-lock.yaml may be outdated (run with --update)"
                    ((issues++))
                fi
                ;;
            yarn)
                if yarn install --frozen-lockfile --dry-run &> /dev/null; then
                    print_status "ok" "yarn.lock is up-to-date"
                else
                    print_status "warn" "yarn.lock may be outdated (run with --update)"
                    ((issues++))
                fi
                ;;
        esac
    fi

    cd "$PROJECT_ROOT"

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

# Fetch Docker image digest
fetch_image_digest() {
    local image=$1
    local tag=$2

    # Try to get digest from Docker Hub or other registries
    if command -v skopeo &> /dev/null; then
        # Use skopeo if available (more reliable)
        skopeo inspect "docker://${image}:${tag}" 2>/dev/null | jq -r '.Digest' || echo ""
    elif command -v docker &> /dev/null; then
        # Fall back to docker pull + inspect
        docker pull "${image}:${tag}" &> /dev/null || return 1
        docker inspect --format='{{index .RepoDigests 0}}' "${image}:${tag}" 2>/dev/null | cut -d'@' -f2 || echo ""
    else
        echo ""
    fi
}

# Pin Docker base images
pin_docker_images() {
    echo ""
    echo -e "${BLUE}Pinning Docker Base Images...${NC}"

    local issues=0

    if ! command -v docker &> /dev/null; then
        print_status "warn" "Docker not installed, skipping digest pinning"
        return 0
    fi

    # Find all Dockerfiles
    local dockerfiles=$(find . -name "Dockerfile*" -type f 2>/dev/null)

    for dockerfile in $dockerfiles; do
        echo ""
        print_status "info" "Processing $(basename $dockerfile)..."

        # Extract FROM lines
        local from_lines=$(grep "^FROM " "$dockerfile" || true)

        if [[ -z "$from_lines" ]]; then
            print_status "warn" "No FROM statements found"
            continue
        fi

        local temp_file=$(mktemp)
        cp "$dockerfile" "$temp_file"

        while IFS= read -r line; do
            if [[ -z "$line" ]]; then
                continue
            fi

            # Skip if already has digest
            if echo "$line" | grep -q "@sha256:"; then
                print_status "ok" "Already pinned: $(echo $line | awk '{print $2}')"
                continue
            fi

            # Extract image and tag
            local image_full=$(echo "$line" | awk '{print $2}')

            # Strip existing digest if present
            image_full="${image_full%%@*}"

            # Expand ARG variables if present
            if [[ $image_full =~ \$\{ ]]; then
                image_full=$(expand_image_vars "$image_full" "$dockerfile")
            fi

            local image_name=$(echo "$image_full" | cut -d':' -f1)
            local image_tag=$(echo "$image_full" | cut -d':' -f2)

            # Default to 'latest' if no tag
            if [[ "$image_name" == "$image_tag" ]]; then
                image_tag="latest"
            fi

            print_status "info" "Fetching digest for ${image_name}:${image_tag}..."

            # Fetch digest
            local digest=$(fetch_image_digest "$image_name" "$image_tag")

            if [[ -n "$digest" ]] && [[ "$digest" == sha256:* ]]; then
                print_status "ok" "Digest: ${digest:0:20}..."

                # Update Dockerfile
                if [[ "$PIN_DOCKER" == "true" ]]; then
                    # Replace the FROM line
                    sed -i "s|^FROM ${image_full}|FROM ${image_name}:${image_tag}@${digest}|g" "$temp_file"
                    print_status "ok" "Updated Dockerfile"
                fi
            else
                print_status "error" "Failed to fetch digest for ${image_name}:${image_tag}"
                ((issues++))
            fi

        done <<< "$from_lines"

        # Write back if pinning
        if [[ "$PIN_DOCKER" == "true" ]]; then
            mv "$temp_file" "$dockerfile"
            print_status "ok" "Saved $(basename $dockerfile)"
        else
            rm "$temp_file"
        fi
    done

    if [[ "$PIN_DOCKER" != "true" ]]; then
        print_status "info" "Digest pinning skipped (use --pin-docker to apply)"
    fi

    return $issues
}

# Check Docker Compose files
check_compose_images() {
    echo ""
    echo -e "${BLUE}Checking Docker Compose Images...${NC}"

    local compose_files=$(find . -name "docker-compose*.yml" -o -name "docker-compose*.yaml" 2>/dev/null)

    for compose_file in $compose_files; do
        if [[ ! -f "$compose_file" ]]; then
            continue
        fi

        print_status "info" "Checking $(basename $compose_file)..."

        # Check for 'latest' tags
        if grep -E 'image:.*:latest' "$compose_file" > /dev/null 2>&1; then
            print_status "warn" "Contains 'latest' tag (consider pinning versions)"
        fi

        # Check for digest pinning
        local digest_count=$(grep -c '@sha256:' "$compose_file" || echo 0)
        if [[ $digest_count -gt 0 ]]; then
            print_status "ok" "$digest_count images pinned with digests"
        else
            print_status "warn" "No digest pinning found"
        fi
    done
}

# Main execution
main() {
    parse_args "$@"

    cd "$PROJECT_ROOT"

    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Dependency Lock Management${NC}"
    echo -e "${BLUE}========================================${NC}"

    if [[ "$CHECK_ONLY" == "true" ]]; then
        echo -e "${YELLOW}Mode: Check Only${NC}"
    fi

    if [[ "$UPDATE_LOCKS" == "true" ]]; then
        echo -e "${YELLOW}Mode: Update Lock Files${NC}"
    fi

    if [[ "$PIN_DOCKER" == "true" ]]; then
        echo -e "${YELLOW}Mode: Pin Docker Images${NC}"
    fi

    local total_issues=0

    # Check Go dependencies
    check_go_deps || { local rc=$?; total_issues=$((total_issues + rc)); }

    # Check Node.js dependencies
    check_node_deps || { local rc=$?; total_issues=$((total_issues + rc)); }

    # Pin Docker images
    if [[ "$PIN_DOCKER" == "true" ]] || [[ "$CHECK_ONLY" == "true" ]]; then
        pin_docker_images || { local rc=$?; total_issues=$((total_issues + rc)); }
    fi

    # Check Docker Compose
    check_compose_images

    # Summary
    echo ""
    echo -e "${BLUE}========================================${NC}"
    echo -e "${BLUE}Summary${NC}"
    echo -e "${BLUE}========================================${NC}"

    if [[ $total_issues -eq 0 ]]; then
        echo -e "${GREEN}✅ All dependencies are properly locked${NC}"
        exit 0
    else
        echo -e "${YELLOW}⚠️  Found $total_issues issue(s)${NC}"

        if [[ "$CHECK_ONLY" == "true" ]]; then
            echo ""
            echo "Run with --update to fix these issues:"
            echo "  $0 --update --pin-docker"
        fi

        exit 1
    fi
}

main "$@"
