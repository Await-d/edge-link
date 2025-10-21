#!/usr/bin/env bash
# scripts/get-version.sh - Get current version with various formats
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

VERSION_FILE="${PROJECT_ROOT}/VERSION"
FRONTEND_PACKAGE="${PROJECT_ROOT}/frontend/package.json"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
NC='\033[0m'

error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

usage() {
  cat <<EOF
Usage: $0 [OPTIONS]

Get current version of Edge-Link

Options:
  --short     Output short version (default: 1.0.0)
  --tag       Output with 'v' prefix (v1.0.0)
  --git-tag   Output latest git tag
  --commit    Output with git commit hash (1.0.0-abc1234)
  --full      Output full info including build metadata
  --json      Output as JSON
  --help      Show this help

Examples:
  $0              # 1.0.0
  $0 --tag        # v1.0.0
  $0 --git-tag    # v1.0.0 (from git tags)
  $0 --commit     # 1.0.0+abc1234
  $0 --full       # 1.0.0+abc1234-dirty
  $0 --json       # {"version":"1.0.0","tag":"v1.0.0",...}

EOF
  exit 1
}

# Get version from VERSION file
get_version() {
  if [[ ! -f "$VERSION_FILE" ]]; then
    error "VERSION file not found: $VERSION_FILE"
    # Fallback to package.json if VERSION file doesn't exist
    if [[ -f "$FRONTEND_PACKAGE" ]]; then
      local version
      version=$(grep -Po '"version":\s*"\K[^"]+' "$FRONTEND_PACKAGE" || echo "")
      if [[ -n "$version" ]]; then
        echo "$version"
        return 0
      fi
    fi
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

# Get latest git tag
get_git_tag() {
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "unknown"
    return 0
  fi

  git describe --tags --abbrev=0 2>/dev/null || echo "unknown"
}

# Get git commit hash
get_commit_hash() {
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "unknown"
    return 0
  fi

  git rev-parse --short=7 HEAD 2>/dev/null || echo "unknown"
}

# Get git branch
get_branch() {
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "unknown"
    return 0
  fi

  git rev-parse --abbrev-ref HEAD 2>/dev/null || echo "unknown"
}

# Check if working directory is dirty
is_dirty() {
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    echo "false"
    return 0
  fi

  if [[ -n "$(git status --porcelain)" ]]; then
    echo "true"
  else
    echo "false"
  fi
}

# Get build timestamp
get_timestamp() {
  date -u +"%Y-%m-%dT%H:%M:%SZ"
}

# Output as JSON
output_json() {
  local version="$1"
  local commit="$2"
  local branch="$3"
  local dirty="$4"
  local timestamp="$5"

  cat <<EOF
{
  "version": "$version",
  "tag": "v$version",
  "commit": "$commit",
  "branch": "$branch",
  "dirty": $dirty,
  "timestamp": "$timestamp",
  "full": "$version+$commit"
}
EOF
}

# Main execution
main() {
  local format="short"

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --short)
        format="short"
        shift
        ;;
      --tag)
        format="tag"
        shift
        ;;
      --git-tag)
        format="git-tag"
        shift
        ;;
      --commit)
        format="commit"
        shift
        ;;
      --full)
        format="full"
        shift
        ;;
      --json)
        format="json"
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

  local version commit branch dirty timestamp git_tag
  version=$(get_version)
  commit=$(get_commit_hash)
  branch=$(get_branch)
  dirty=$(is_dirty)
  timestamp=$(get_timestamp)
  git_tag=$(get_git_tag)

  case "$format" in
    short)
      echo "$version"
      ;;
    tag)
      echo "v$version"
      ;;
    git-tag)
      echo "$git_tag"
      ;;
    commit)
      echo "${version}+${commit}"
      ;;
    full)
      local suffix=""
      [[ "$dirty" == "true" ]] && suffix="-dirty"
      echo "${version}+${commit}${suffix}"
      ;;
    json)
      output_json "$version" "$commit" "$branch" "$dirty" "$timestamp"
      ;;
  esac
}

main "$@"
