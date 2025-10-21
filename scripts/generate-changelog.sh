#!/usr/bin/env bash
# scripts/generate-changelog.sh - Generate CHANGELOG from git history
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"
CHANGELOG_FILE="${PROJECT_ROOT}/CHANGELOG.md"

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m'

log() { echo -e "${GREEN}[INFO]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC} $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }

usage() {
  cat <<EOF
Usage: $0 [OPTIONS] [GIT_RANGE]

Generate CHANGELOG from conventional commits

Options:
  --append       Append to existing CHANGELOG.md instead of replacing
  --output FILE  Write to FILE instead of CHANGELOG.md
  --stdout       Output to stdout instead of file
  --from-tag TAG Start from specific tag
  --help         Show this help

Arguments:
  GIT_RANGE      Git range (e.g., v1.0.0..HEAD, v1.0.0..v1.1.0)
                 Default: All commits if no tags exist, or latest tag..HEAD

Examples:
  $0                          # Generate full changelog
  $0 v1.0.0..HEAD             # From v1.0.0 to HEAD
  $0 --from-tag v1.0.0        # From v1.0.0 to HEAD
  $0 --stdout                 # Output to stdout
  $0 --append v1.0.0..HEAD    # Append new changes

EOF
  exit 1
}

# Get latest git tag
get_latest_tag() {
  git describe --tags --abbrev=0 2>/dev/null || echo ""
}

# Get all tags sorted by version
get_all_tags() {
  git tag -l 'v*' --sort=-version:refname 2>/dev/null || echo ""
}

# Parse commit message
parse_commit() {
  local commit_msg="$1"
  local commit_hash="$2"

  # Extract type, scope, and subject from conventional commit
  # Format: type(scope): subject
  if [[ "$commit_msg" =~ ^([a-z]+)(\([a-z0-9_-]+\))?:\ (.+)$ ]]; then
    local type="${BASH_REMATCH[1]}"
    local scope="${BASH_REMATCH[2]}"
    local subject="${BASH_REMATCH[3]}"

    # Remove parentheses from scope
    scope="${scope#(}"
    scope="${scope%)}"

    echo "$type|$scope|$subject|$commit_hash"
  else
    # Not a conventional commit - categorize as 'other'
    echo "other||$commit_msg|$commit_hash"
  fi
}

# Get section name for commit type
get_section_name() {
  local type="$1"

  case "$type" in
    feat) echo "Features" ;;
    fix) echo "Bug Fixes" ;;
    perf) echo "Performance Improvements" ;;
    refactor) echo "Code Refactoring" ;;
    docs) echo "Documentation" ;;
    test) echo "Tests" ;;
    build) echo "Build System" ;;
    ci) echo "CI/CD" ;;
    style) echo "Code Style" ;;
    chore) echo "Chores" ;;
    revert) echo "Reverts" ;;
    *) echo "Other Changes" ;;
  esac
}

# Should include type in changelog
should_include_type() {
  local type="$1"

  case "$type" in
    feat|fix|perf|refactor|docs) return 0 ;;
    test|build|ci|style|chore) return 1 ;;
    *) return 0 ;;
  esac
}

# Generate changelog for a specific range
generate_changelog_section() {
  local git_range="$1"
  local version_tag="$2"

  # Get commits in range
  local commits
  commits=$(git log "$git_range" --pretty=format:"%H|%s" --reverse 2>/dev/null || echo "")

  if [[ -z "$commits" ]]; then
    warn "No commits found in range: $git_range"
    return 0
  fi

  # Parse commits by type
  declare -A sections
  declare -A commit_lists

  while IFS='|' read -r commit_hash commit_msg; do
    local parsed
    parsed=$(parse_commit "$commit_msg" "$commit_hash")

    IFS='|' read -r type scope subject hash <<< "$parsed"

    # Skip if type should not be included
    if ! should_include_type "$type"; then
      continue
    fi

    local section
    section=$(get_section_name "$type")

    # Add to section
    if [[ -z "${sections[$section]:-}" ]]; then
      sections[$section]="1"
      commit_lists[$section]=""
    fi

    # Format commit line
    local commit_line
    if [[ -n "$scope" ]]; then
      commit_line="- **${scope}**: ${subject}"
    else
      commit_line="- ${subject}"
    fi

    # Add commit hash link (short hash)
    local short_hash="${hash:0:7}"
    commit_line="${commit_line} ([\`${short_hash}\`](https://github.com/edgelink/edge-link/commit/${hash}))"

    commit_lists[$section]+="${commit_line}"$'\n'
  done <<< "$commits"

  # Output changelog section
  local version_number="${version_tag#v}"
  local release_date
  release_date=$(date +%Y-%m-%d)

  echo "## [$version_number] - $release_date"
  echo ""

  # Output in order
  local ordered_sections=(
    "Features"
    "Bug Fixes"
    "Performance Improvements"
    "Code Refactoring"
    "Documentation"
    "Other Changes"
  )

  for section in "${ordered_sections[@]}"; do
    if [[ -n "${commit_lists[$section]:-}" ]]; then
      echo "### $section"
      echo ""
      echo -n "${commit_lists[$section]}"
      echo ""
    fi
  done
}

# Generate full changelog from all tags
generate_full_changelog() {
  local output_file="$1"

  log "Generating full changelog from git history..."

  # Get all tags
  local tags
  tags=$(get_all_tags)

  if [[ -z "$tags" ]]; then
    warn "No version tags found. Generating changelog for all commits..."
    local latest_version="Unreleased"
    local changelog_content
    changelog_content=$(generate_changelog_section "HEAD" "$latest_version")

    cat > "$output_file" <<EOF
# Changelog

All notable changes to the Edge-Link project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

$changelog_content

EOF
    return 0
  fi

  # Generate header
  cat > "$output_file" <<'EOF'
# Changelog

All notable changes to the Edge-Link project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

EOF

  # Generate section for each tag
  local prev_tag=""
  local tag_array=()
  while IFS= read -r tag; do
    tag_array+=("$tag")
  done <<< "$tags"

  # Add unreleased section if there are commits after latest tag
  local latest_tag="${tag_array[0]}"
  local unreleased_commits
  unreleased_commits=$(git log "${latest_tag}..HEAD" --oneline 2>/dev/null | wc -l)

  if [[ "$unreleased_commits" -gt 0 ]]; then
    log "Generating unreleased section (${unreleased_commits} commits)..."
    echo "## [Unreleased]" >> "$output_file"
    echo "" >> "$output_file"
    generate_changelog_section "${latest_tag}..HEAD" "Unreleased" >> "$output_file"
  fi

  # Process each tag
  for i in "${!tag_array[@]}"; do
    local current_tag="${tag_array[$i]}"
    local next_tag="${tag_array[$((i+1))]:-}"

    log "Generating section for $current_tag..."

    if [[ -n "$next_tag" ]]; then
      generate_changelog_section "${next_tag}..${current_tag}" "$current_tag" >> "$output_file"
    else
      # First tag - get all commits up to it
      generate_changelog_section "$current_tag" "$current_tag" >> "$output_file"
    fi
  done

  log "Changelog generated successfully: $output_file"
}

# Main execution
main() {
  local append=false
  local output_file="$CHANGELOG_FILE"
  local to_stdout=false
  local git_range=""
  local from_tag=""

  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --append)
        append=true
        shift
        ;;
      --output)
        output_file="$2"
        shift 2
        ;;
      --stdout)
        to_stdout=true
        shift
        ;;
      --from-tag)
        from_tag="$2"
        shift 2
        ;;
      --help|-h)
        usage
        ;;
      -*)
        error "Unknown option: $1"
        usage
        ;;
      *)
        git_range="$1"
        shift
        ;;
    esac
  done

  cd "$PROJECT_ROOT"

  # Check if git repository
  if ! git rev-parse --git-dir > /dev/null 2>&1; then
    error "Not a git repository"
    exit 1
  fi

  # Determine git range
  if [[ -n "$from_tag" ]]; then
    git_range="${from_tag}..HEAD"
  elif [[ -z "$git_range" ]]; then
    # Generate full changelog
    if [[ "$to_stdout" == true ]]; then
      local temp_file
      temp_file=$(mktemp)
      generate_full_changelog "$temp_file"
      cat "$temp_file"
      rm -f "$temp_file"
    else
      generate_full_changelog "$output_file"
    fi
    exit 0
  fi

  # Generate changelog for specific range
  log "Generating changelog for range: $git_range"

  local latest_version
  latest_version=$(get_latest_tag)
  [[ -z "$latest_version" ]] && latest_version="Unreleased"

  local changelog_content
  changelog_content=$(generate_changelog_section "$git_range" "$latest_version")

  if [[ "$to_stdout" == true ]]; then
    echo "$changelog_content"
  elif [[ "$append" == true ]]; then
    # Append to existing file
    log "Appending to $output_file"
    echo "" >> "$output_file"
    echo "$changelog_content" >> "$output_file"
  else
    # Replace file
    log "Writing to $output_file"
    cat > "$output_file" <<EOF
# Changelog

All notable changes to the Edge-Link project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

$changelog_content

EOF
  fi

  log "Changelog generation completed!"
}

main "$@"
