#!/usr/bin/env bash
# scripts/validate-commit-msg.sh - Validate commit messages against Conventional Commits
set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m'

error() { echo -e "${RED}✗${NC} $*" >&2; }
success() { echo -e "${GREEN}✓${NC} $*"; }
info() { echo -e "${YELLOW}ℹ${NC} $*"; }

# Valid commit types
VALID_TYPES=(
  "feat"      # New feature
  "fix"       # Bug fix
  "docs"      # Documentation
  "style"     # Code style (formatting, etc.)
  "refactor"  # Code refactoring
  "perf"      # Performance improvement
  "test"      # Tests
  "build"     # Build system
  "ci"        # CI/CD
  "chore"     # Maintenance
  "revert"    # Revert previous commit
)

usage() {
  cat <<EOF
Usage: $0 [COMMIT_MSG_FILE]

Validate commit message against Conventional Commits specification.

Format: <type>(<scope>): <subject>

Valid types:
$(printf "  - %s\n" "${VALID_TYPES[@]}")

Examples:
  ✓ feat(api): add user authentication endpoint
  ✓ fix(ui): resolve button alignment issue
  ✓ docs: update README with installation steps
  ✓ perf(database): optimize query performance
  ✓ feat(auth)!: breaking change in auth API

  ✗ Add new feature (missing type)
  ✗ feat: add feature (missing scope for important features)
  ✗ FIX(api): wrong case (type must be lowercase)

Optional elements:
  - Scope: (component) affected by the change
  - !: Breaking change indicator
  - Body: Detailed description (after blank line)
  - Footer: Issue references, breaking changes

Special patterns:
  - Merge commits: "Merge ..." (auto-allowed)
  - Revert commits: "Revert ..." (auto-allowed)
  - WIP commits: Allowed in feature branches (discouraged in main)

EOF
  exit 1
}

# Check if type is valid
is_valid_type() {
  local type="$1"
  for valid_type in "${VALID_TYPES[@]}"; do
    if [[ "$type" == "$valid_type" ]]; then
      return 0
    fi
  done
  return 1
}

# Validate conventional commit format
validate_commit_msg() {
  local msg="$1"

  # Remove leading/trailing whitespace
  msg=$(echo "$msg" | sed -e 's/^[[:space:]]*//' -e 's/[[:space:]]*$//')

  # Allow merge commits
  if [[ "$msg" =~ ^Merge ]]; then
    success "Merge commit detected (auto-allowed)"
    return 0
  fi

  # Allow revert commits
  if [[ "$msg" =~ ^Revert ]]; then
    success "Revert commit detected (auto-allowed)"
    return 0
  fi

  # Extract first line (subject)
  local subject
  subject=$(echo "$msg" | head -n1)

  # Conventional commit pattern: type(scope): subject
  # type(scope)!: subject (breaking change)
  # type: subject (no scope)
  local pattern='^([a-z]+)(\([a-z0-9_/-]+\))?(!)?:[[:space:]](.+)$'

  if [[ ! "$subject" =~ $pattern ]]; then
    error "Invalid commit message format"
    echo ""
    echo "Expected format: <type>(<scope>): <subject>"
    echo "Your message: $subject"
    echo ""
    info "Examples:"
    echo "  feat(api): add new endpoint"
    echo "  fix(ui): resolve alignment issue"
    echo "  docs: update README"
    echo ""
    info "Valid types: ${VALID_TYPES[*]}"
    return 1
  fi

  local type="${BASH_REMATCH[1]}"
  local scope="${BASH_REMATCH[2]}"
  local breaking="${BASH_REMATCH[3]}"
  local description="${BASH_REMATCH[4]}"

  # Remove parentheses from scope
  scope="${scope#(}"
  scope="${scope%)}"

  # Validate type
  if ! is_valid_type "$type"; then
    error "Invalid commit type: '$type'"
    info "Valid types: ${VALID_TYPES[*]}"
    return 1
  fi

  # Validate subject length
  if [[ ${#subject} -gt 100 ]]; then
    error "Subject line too long (${#subject} > 100 characters)"
    info "Keep subject concise, use body for details"
    return 1
  fi

  # Check subject starts with lowercase
  if [[ "$description" =~ ^[A-Z] ]]; then
    error "Subject should start with lowercase letter"
    info "Change: '$description' → '$(echo "${description:0:1}" | tr '[:upper:]' '[:lower:]')${description:1}'"
    return 1
  fi

  # Check subject doesn't end with period
  if [[ "$description" =~ \.$ ]]; then
    error "Subject should not end with period"
    return 1
  fi

  # Success - show parsed components
  success "Valid conventional commit"
  echo ""
  echo "  Type:        $type"
  [[ -n "$scope" ]] && echo "  Scope:       $scope"
  [[ -n "$breaking" ]] && echo "  Breaking:    YES"
  echo "  Description: $description"

  return 0
}

# Main execution
main() {
  # If no argument, show usage
  if [[ $# -eq 0 ]]; then
    usage
  fi

  local commit_msg_file="$1"

  # Check if file exists
  if [[ ! -f "$commit_msg_file" ]]; then
    error "Commit message file not found: $commit_msg_file"
    exit 1
  fi

  # Read commit message
  local commit_msg
  commit_msg=$(cat "$commit_msg_file")

  # Skip empty messages
  if [[ -z "$commit_msg" ]] || [[ "$commit_msg" =~ ^[[:space:]]*$ ]]; then
    error "Empty commit message"
    exit 1
  fi

  # Validate
  if validate_commit_msg "$commit_msg"; then
    exit 0
  else
    echo ""
    error "Commit message validation failed"
    echo ""
    info "See commit message guidelines: https://www.conventionalcommits.org/"
    exit 1
  fi
}

main "$@"
