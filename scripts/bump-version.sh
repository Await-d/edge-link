#!/usr/bin/env bash
# scripts/bump-version.sh - Version bump automation
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Files that contain version numbers
VERSION_FILE="${PROJECT_ROOT}/VERSION"
FRONTEND_PACKAGE="${PROJECT_ROOT}/frontend/package.json"
HELM_CHART="${PROJECT_ROOT}/infrastructure/helm/edge-link-control-plane/Chart.yaml"
HELM_CHART_SIDECAR="${PROJECT_ROOT}/infrastructure/helm/edgelink-sidecar/Chart.yaml"

# Flags
DRY_RUN=false
NO_GIT=false
PRE_RELEASE=""

log() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

usage() {
  cat <<EOF
Usage: $0 <major|minor|patch|--current> [OPTIONS]

Version Bump Automation for Edge-Link

Commands:
  major      Bump major version (X.0.0) - Breaking changes
  minor      Bump minor version (0.X.0) - New features
  patch      Bump patch version (0.0.X) - Bug fixes
  --current  Show current version

Options:
  --pre-release <name>  Add pre-release identifier (alpha, beta, rc)
  --dry-run            Preview changes without modifying files
  --no-git             Skip git operations (tag, commit)
  --help               Show this help message

Examples:
  $0 patch                        # 1.0.0 → 1.0.1
  $0 minor --pre-release alpha    # 1.0.1 → 1.1.0-alpha.1
  $0 major --dry-run              # Preview major version bump
  $0 patch --no-git               # Bump version without git commit/tag
  $0 --current                    # Display current version

Pre-release naming:
  alpha   - Early development version
  beta    - Feature complete, testing phase
  rc      - Release candidate

EOF
  exit 1
}

# Parse semantic version
parse_version() {
  local version="$1"
  # Remove 'v' prefix if present
  version="${version#v}"

  if [[ ! "$version" =~ ^[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    error "Invalid version format: $version (expected X.Y.Z)"
    return 1
  fi

  local major minor patch
  IFS='.' read -r major minor patch <<< "$version"
  echo "$major $minor $patch"
}

# Get current version from VERSION file
get_current_version() {
  if [[ ! -f "$VERSION_FILE" ]]; then
    error "VERSION file not found: $VERSION_FILE"
    return 1
  fi

  local version
  version=$(cat "$VERSION_FILE" | tr -d '[:space:]')

  if [[ -z "$version" ]]; then
    error "Could not read version from VERSION file"
    return 1
  fi

  echo "$version"
}

# Bump version
bump_version() {
  local bump_type="$1"
  local current_version
  current_version=$(get_current_version)

  # Strip pre-release suffix if present
  local base_version="$current_version"
  if [[ "$current_version" =~ ^([0-9]+\.[0-9]+\.[0-9]+)(-.*)?$ ]]; then
    base_version="${BASH_REMATCH[1]}"
  fi

  local major minor patch
  read -r major minor patch <<< "$(parse_version "$base_version")"

  case "$bump_type" in
    major)
      major=$((major + 1))
      minor=0
      patch=0
      ;;
    minor)
      minor=$((minor + 1))
      patch=0
      ;;
    patch)
      patch=$((patch + 1))
      ;;
    *)
      error "Invalid bump type: $bump_type"
      return 1
      ;;
  esac

  local new_version="${major}.${minor}.${patch}"

  # Add pre-release suffix if specified
  if [[ -n "$PRE_RELEASE" ]]; then
    # Check if bumping a pre-release version
    if [[ "$current_version" =~ -${PRE_RELEASE}\.([0-9]+)$ ]]; then
      local pre_num="${BASH_REMATCH[1]}"
      pre_num=$((pre_num + 1))
      new_version="${new_version}-${PRE_RELEASE}.${pre_num}"
    else
      new_version="${new_version}-${PRE_RELEASE}.1"
    fi
  fi

  echo "$new_version"
}

# Update VERSION file
update_version_file() {
  local new_version="$1"

  if [[ "$DRY_RUN" == true ]]; then
    log "[DRY RUN] Would update VERSION file to: $new_version"
    return 0
  fi

  log "Updating VERSION file to: $new_version"
  echo "$new_version" > "$VERSION_FILE"
}

# Update package.json
update_package_json() {
  local new_version="$1"
  local file="$2"

  if [[ ! -f "$file" ]]; then
    warn "File not found: $file (skipping)"
    return 0
  fi

  if [[ "$DRY_RUN" == true ]]; then
    log "[DRY RUN] Would update $file to version $new_version"
    return 0
  fi

  log "Updating $file to version $new_version"

  # Use sed for in-place replacement
  sed -i "s/\"version\":\s*\"[^\"]*\"/\"version\": \"$new_version\"/" "$file"
}

# Update Helm Chart.yaml
update_helm_chart() {
  local new_version="$1"
  local file="$2"

  if [[ ! -f "$file" ]]; then
    warn "File not found: $file (skipping)"
    return 0
  fi

  if [[ "$DRY_RUN" == true ]]; then
    log "[DRY RUN] Would update $file to version $new_version"
    return 0
  fi

  log "Updating $file to version $new_version"

  # Update both version and appVersion
  sed -i "s/^version:\s*.*/version: $new_version/" "$file"
  sed -i "s/^appVersion:\s*.*/appVersion: \"$new_version\"/" "$file"
}

# Verify all files have consistent version
verify_version_consistency() {
  local expected_version="$1"
  local errors=0

  log "Verifying version consistency across all files..."

  # Check VERSION file
  if [[ -f "$VERSION_FILE" ]]; then
    local file_version
    file_version=$(cat "$VERSION_FILE" | tr -d '[:space:]')
    if [[ "$file_version" != "$expected_version" ]]; then
      error "Version mismatch in VERSION file: expected $expected_version, got $file_version"
      ((errors++))
    else
      log "✓ VERSION file: $file_version"
    fi
  fi

  # Check package.json
  if [[ -f "$FRONTEND_PACKAGE" ]]; then
    local pkg_version
    pkg_version=$(grep -Po '"version":\s*"\K[^"]+' "$FRONTEND_PACKAGE" || echo "")
    if [[ "$pkg_version" != "$expected_version" ]]; then
      error "Version mismatch in package.json: expected $expected_version, got $pkg_version"
      ((errors++))
    else
      log "✓ package.json: $pkg_version"
    fi
  fi

  # Check Helm charts
  for chart_file in "$HELM_CHART" "$HELM_CHART_SIDECAR"; do
    if [[ -f "$chart_file" ]]; then
      local chart_version
      chart_version=$(grep -Po '^version:\s*\K.*' "$chart_file" || echo "")
      local app_version
      app_version=$(grep -Po '^appVersion:\s*"\K[^"]+' "$chart_file" || echo "")

      if [[ "$chart_version" != "$expected_version" ]] || [[ "$app_version" != "$expected_version" ]]; then
        error "Version mismatch in $(basename "$chart_file"): expected $expected_version, got version=$chart_version appVersion=$app_version"
        ((errors++))
      else
        log "✓ $(basename "$chart_file"): $chart_version / $app_version"
      fi
    fi
  done

  return "$errors"
}

# Create git tag
create_git_tag() {
  local version="$1"
  local tag="v${version}"

  if [[ "$NO_GIT" == true ]]; then
    log "[NO GIT] Skipping git tag creation"
    return 0
  fi

  if [[ "$DRY_RUN" == true ]]; then
    log "[DRY RUN] Would create git tag: $tag"
    return 0
  fi

  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    warn "Not a git repository - skipping tag creation"
    return 0
  fi

  # Check if tag already exists
  if git rev-parse "$tag" >/dev/null 2>&1; then
    warn "Tag $tag already exists - skipping tag creation"
    return 0
  fi

  log "Creating git tag: $tag"
  git tag -a "$tag" -m "chore(release): $tag"

  log "Tag created successfully. Push with: git push origin $tag"
}

# Main execution
main() {
  if [[ $# -eq 0 ]]; then
    usage
  fi

  local bump_type=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --current)
        cd "$PROJECT_ROOT"
        local current
        current=$(get_current_version)
        echo "$current"
        exit 0
        ;;
      major|minor|patch)
        bump_type="$1"
        shift
        ;;
      --pre-release)
        PRE_RELEASE="$2"
        if [[ ! "$PRE_RELEASE" =~ ^(alpha|beta|rc)$ ]]; then
          error "Invalid pre-release name: $PRE_RELEASE (expected: alpha, beta, or rc)"
          exit 1
        fi
        shift 2
        ;;
      --dry-run)
        DRY_RUN=true
        shift
        ;;
      --no-git)
        NO_GIT=true
        shift
        ;;
      --help|-h)
        usage
        ;;
      *)
        error "Unknown option: $1"
        usage
        ;;
    esac
  done

  # Validate bump type
  if [[ -z "$bump_type" ]]; then
    error "Missing bump type (major, minor, or patch)"
    usage
  fi

  cd "$PROJECT_ROOT"

  local new_version
  new_version=$(bump_version "$bump_type")

  if [[ "$DRY_RUN" == true ]]; then
    echo -e "${BLUE}[DRY RUN MODE]${NC}"
    echo ""
  fi

  log "Bumping version: $(get_current_version) → $new_version"

  # Update all files
  update_version_file "$new_version"
  update_package_json "$new_version" "$FRONTEND_PACKAGE"
  update_helm_chart "$new_version" "$HELM_CHART"
  update_helm_chart "$new_version" "$HELM_CHART_SIDECAR"

  if [[ "$DRY_RUN" == true ]]; then
    log "[DRY RUN] Version bump preview complete"
    exit 0
  fi

  # Verify consistency
  if ! verify_version_consistency "$new_version"; then
    error "Version consistency check failed!"
    exit 1
  fi

  log "Successfully updated all files to version $new_version"

  # Git operations
  if [[ "$NO_GIT" == false ]] && git rev-parse --git-dir > /dev/null 2>&1; then
    log "Staging version changes..."
    git add "$VERSION_FILE" "$FRONTEND_PACKAGE" "$HELM_CHART" "$HELM_CHART_SIDECAR" 2>/dev/null || true

    # Commit changes
    git commit -m "chore(release): bump version to $new_version" || warn "No changes to commit"

    # Create tag
    create_git_tag "$new_version"
  fi

  log "Version bump completed!"

  if [[ "$NO_GIT" == false ]]; then
    log "Next steps:"
    log "  1. Review changes: git show"
    log "  2. Push changes: git push && git push origin v${new_version}"
    log "  3. Generate changelog: ./scripts/generate-changelog.sh"
  fi
}

main "$@"
