#!/bin/bash
# EdgeLink Helm Chart Validation Script
# Purpose: Validate Helm charts for syntax, linting, and required configurations
# Usage: ./scripts/validate-helm-charts.sh [options]

set -euo pipefail

# Color output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Default values
CHART_DIR=""
STRICT=false
VERBOSE=false
FIX=false

# Script directory
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "${SCRIPT_DIR}/.." && pwd)"

# Default chart directory
DEFAULT_CHART_DIR="$PROJECT_ROOT/infrastructure/helm/edge-link-control-plane"

# Validation counters
TOTAL_CHECKS=0
PASSED_CHECKS=0
FAILED_CHECKS=0
WARNING_CHECKS=0

# Function: Print usage
usage() {
  cat <<EOF
Usage: ${0##*/} [OPTIONS]

Validate Helm charts for syntax, linting, and required configurations.

OPTIONS:
  -c, --chart-dir DIR    Chart directory to validate (default: infrastructure/helm/edge-link-control-plane)
  -s, --strict           Enable strict validation (fail on warnings)
  -f, --fix              Attempt to fix issues automatically (where possible)
      --verbose          Enable verbose output
  -h, --help             Show this help message

EXAMPLES:
  # Validate default chart
  ${0##*/}

  # Validate specific chart
  ${0##*/} --chart-dir ./infrastructure/helm/my-chart

  # Strict validation
  ${0##*/} --strict

  # Verbose output
  ${0##*/} --verbose

VALIDATION CHECKS:
  1. Helm lint
  2. Helm template rendering
  3. YAML syntax validation
  4. Required values validation (resources, probes, security contexts)
  5. HPA configuration check
  6. PDB configuration check
  7. Security context validation

EOF
}

# Function: Log message
log() {
  echo -e "${GREEN}[INFO]${NC} $*"
}

# Function: Log debug
debug() {
  if [[ "$VERBOSE" == "true" ]]; then
    echo -e "${BLUE}[DEBUG]${NC} $*"
  fi
}

# Function: Log warning
warn() {
  echo -e "${YELLOW}[WARN]${NC} $*"
  ((WARNING_CHECKS++))
}

# Function: Log error
error() {
  echo -e "${RED}[ERROR]${NC} $*" >&2
  ((FAILED_CHECKS++))
}

# Function: Log success
success() {
  echo -e "${GREEN}[PASS]${NC} $*"
  ((PASSED_CHECKS++))
}

# Function: Die with error
die() {
  error "$*"
  exit 1
}

# Function: Check dependencies
check_dependencies() {
  local deps=("helm")
  local optional_deps=("yamllint")

  for cmd in "${deps[@]}"; do
    if ! command -v "$cmd" &> /dev/null; then
      die "Required command '$cmd' not found. Please install it first."
    fi
  done

  debug "Helm version: $(helm version --short)"

  # Check optional dependencies
  for cmd in "${optional_deps[@]}"; do
    if ! command -v "$cmd" &> /dev/null; then
      warn "Optional command '$cmd' not found. Some checks will be skipped."
    else
      debug "$cmd found: $(command -v $cmd)"
    fi
  done
}

# Function: Validate chart directory
validate_chart_dir() {
  if [[ -z "$CHART_DIR" ]]; then
    CHART_DIR="$DEFAULT_CHART_DIR"
  fi

  if [[ ! -d "$CHART_DIR" ]]; then
    die "Chart directory not found: $CHART_DIR"
  fi

  if [[ ! -f "$CHART_DIR/Chart.yaml" ]]; then
    die "Chart.yaml not found in: $CHART_DIR"
  fi

  if [[ ! -f "$CHART_DIR/values.yaml" ]]; then
    die "values.yaml not found in: $CHART_DIR"
  fi

  log "Validating chart: $CHART_DIR"
}

# Function: Run helm lint
run_helm_lint() {
  ((TOTAL_CHECKS++))
  log "Running helm lint..."

  local lint_output
  if lint_output=$(helm lint "$CHART_DIR" 2>&1); then
    success "Helm lint passed"
    debug "$lint_output"
    return 0
  else
    error "Helm lint failed"
    echo "$lint_output"
    return 1
  fi
}

# Function: Run helm template
run_helm_template() {
  ((TOTAL_CHECKS++))
  log "Running helm template..."

  local template_output
  if template_output=$(helm template test-release "$CHART_DIR" --debug 2>&1); then
    success "Helm template rendering successful"
    debug "Template output length: $(echo "$template_output" | wc -l) lines"
    return 0
  else
    error "Helm template rendering failed"
    echo "$template_output"
    return 1
  fi
}

# Function: Validate YAML syntax
validate_yaml_syntax() {
  if ! command -v yamllint &> /dev/null; then
    warn "yamllint not found, skipping YAML syntax validation"
    return 0
  fi

  ((TOTAL_CHECKS++))
  log "Validating YAML syntax..."

  local yaml_files
  yaml_files=$(find "$CHART_DIR" -name "*.yaml" -o -name "*.yml")

  local failed=false
  for file in $yaml_files; do
    if ! yamllint -d relaxed "$file" > /dev/null 2>&1; then
      error "YAML syntax error in: $file"
      yamllint "$file" || true
      failed=true
    fi
  done

  if [[ "$failed" == "false" ]]; then
    success "YAML syntax validation passed"
    return 0
  else
    return 1
  fi
}

# Function: Check required values
check_required_values() {
  ((TOTAL_CHECKS++))
  log "Checking required values..."

  local values_file="$CHART_DIR/values.yaml"
  local missing_values=()

  # Check for resource limits and requests
  if ! grep -q "resources:" "$values_file"; then
    missing_values+=("resources")
  fi

  # Check for health probes
  if ! grep -q "healthProbes:" "$values_file" && ! grep -q "livenessProbe:" "$values_file"; then
    warn "No health probes configuration found in values.yaml"
  fi

  # Check for security contexts
  if ! grep -q "securityContext:" "$values_file" && ! grep -q "podSecurityContext:" "$values_file"; then
    warn "No security context configuration found in values.yaml"
  fi

  if [[ ${#missing_values[@]} -eq 0 ]]; then
    success "Required values check passed"
    return 0
  else
    error "Missing required values: ${missing_values[*]}"
    return 1
  fi
}

# Function: Validate HPA configuration
check_hpa_config() {
  ((TOTAL_CHECKS++))
  log "Checking HPA configuration..."

  local values_file="$CHART_DIR/values.yaml"
  local issues=()

  # Check if autoscaling is configured
  if ! grep -q "autoscaling:" "$values_file"; then
    warn "No autoscaling configuration found"
    return 0
  fi

  # Check for minReplicas and maxReplicas
  if ! grep -q "minReplicas:" "$values_file"; then
    issues+=("minReplicas not set")
  fi

  if ! grep -q "maxReplicas:" "$values_file"; then
    issues+=("maxReplicas not set")
  fi

  # Check for target metrics
  if ! grep -q "targetCPUUtilizationPercentage:" "$values_file" && \
     ! grep -q "targetMemoryUtilizationPercentage:" "$values_file"; then
    issues+=("No target metrics configured")
  fi

  if [[ ${#issues[@]} -eq 0 ]]; then
    success "HPA configuration check passed"
    return 0
  else
    warn "HPA configuration issues: ${issues[*]}"
    return 0
  fi
}

# Function: Validate PDB configuration
check_pdb_config() {
  ((TOTAL_CHECKS++))
  log "Checking PDB configuration..."

  local values_file="$CHART_DIR/values.yaml"

  # Check if PDB is configured
  if ! grep -q "podDisruptionBudget:" "$values_file"; then
    warn "No PodDisruptionBudget configuration found"
    return 0
  fi

  # Check for minAvailable or maxUnavailable
  if ! grep -q "minAvailable:" "$values_file" && \
     ! grep -q "maxUnavailable:" "$values_file"; then
    warn "PDB configured but no minAvailable or maxUnavailable set"
    return 0
  fi

  success "PDB configuration check passed"
  return 0
}

# Function: Validate security contexts
check_security_contexts() {
  ((TOTAL_CHECKS++))
  log "Checking security contexts..."

  local values_file="$CHART_DIR/values.yaml"
  local security_issues=()

  # Check for runAsNonRoot
  if ! grep -q "runAsNonRoot: true" "$values_file"; then
    security_issues+=("runAsNonRoot not set to true")
  fi

  # Check for readOnlyRootFilesystem
  if ! grep -q "readOnlyRootFilesystem: true" "$values_file"; then
    warn "readOnlyRootFilesystem not consistently set to true (recommended)"
  fi

  # Check for capabilities drop
  if ! grep -q "drop:" "$values_file"; then
    security_issues+=("No capabilities drop configured")
  fi

  if [[ ${#security_issues[@]} -eq 0 ]]; then
    success "Security contexts check passed"
    return 0
  else
    warn "Security context issues: ${security_issues[*]}"
    return 0
  fi
}

# Function: Validate Prometheus annotations
check_prometheus_annotations() {
  ((TOTAL_CHECKS++))
  log "Checking Prometheus service monitor annotations..."

  local values_file="$CHART_DIR/values.yaml"

  # Check if service has prometheus annotations
  if grep -q "prometheus.io/scrape" "$values_file"; then
    success "Prometheus annotations found"
    return 0
  else
    warn "No Prometheus service monitor annotations found in values.yaml"
    return 0
  fi
}

# Function: Print summary
print_summary() {
  echo ""
  log "========================================"
  log "Validation Summary"
  log "========================================"
  log "Total Checks: $TOTAL_CHECKS"
  success "Passed: $PASSED_CHECKS"
  
  if [[ $WARNING_CHECKS -gt 0 ]]; then
    warn "Warnings: $WARNING_CHECKS"
  fi
  
  if [[ $FAILED_CHECKS -gt 0 ]]; then
    error "Failed: $FAILED_CHECKS"
  fi
  
  log "========================================"

  if [[ $FAILED_CHECKS -gt 0 ]]; then
    if [[ "$STRICT" == "true" ]] && [[ $WARNING_CHECKS -gt 0 ]]; then
      die "Validation failed with $FAILED_CHECKS error(s) and $WARNING_CHECKS warning(s) (strict mode)"
    else
      die "Validation failed with $FAILED_CHECKS error(s)"
    fi
  fi

  if [[ "$STRICT" == "true" ]] && [[ $WARNING_CHECKS -gt 0 ]]; then
    die "Validation failed with $WARNING_CHECKS warning(s) (strict mode)"
  fi

  log ""
  log "All validations passed successfully!"
}

# Main function
main() {
  # Parse arguments
  while [[ $# -gt 0 ]]; do
    case $1 in
      -c|--chart-dir)
        CHART_DIR="$2"
        shift 2
        ;;
      -s|--strict)
        STRICT=true
        shift
        ;;
      -f|--fix)
        FIX=true
        shift
        ;;
      --verbose)
        VERBOSE=true
        shift
        ;;
      -h|--help)
        usage
        exit 0
        ;;
      *)
        error "Unknown option: $1"
        usage
        exit 1
        ;;
    esac
  done

  # Check dependencies
  check_dependencies

  # Validate chart directory
  validate_chart_dir

  log ""
  log "Starting Helm chart validation..."
  log ""

  # Run validation checks
  run_helm_lint
  run_helm_template
  validate_yaml_syntax
  check_required_values
  check_hpa_config
  check_pdb_config
  check_security_contexts
  check_prometheus_annotations

  # Print summary
  print_summary
}

# Run main
main "$@"
