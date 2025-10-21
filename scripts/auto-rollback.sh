#!/bin/bash
#
# EdgeLink Automatic Rollback Script
# 
# This script performs automatic rollback of Kubernetes deployments based on
# health metrics and alert conditions.
#
# Usage:
#   ./auto-rollback.sh <deployment-name> [reason]
#
# Environment Variables:
#   NAMESPACE              - Kubernetes namespace (default: edgelink)
#   DEPLOYMENT_PREFIX      - Deployment name prefix filter (default: edgelink)
#   MAX_ROLLBACK_RETRIES   - Maximum rollback attempts (default: 3)
#   ROLLBACK_TIMEOUT       - Rollback timeout in seconds (default: 300)
#   DRY_RUN               - Dry run mode, no actual changes (default: false)
#   WEBHOOK_URL           - Webhook URL for notifications
#

set -euo pipefail

# Configuration from environment
NAMESPACE="${NAMESPACE:-edgelink}"
DEPLOYMENT_PREFIX="${DEPLOYMENT_PREFIX:-edgelink}"
MAX_ROLLBACK_RETRIES="${MAX_ROLLBACK_RETRIES:-3}"
ROLLBACK_TIMEOUT="${ROLLBACK_TIMEOUT:-300}"
DRY_RUN="${DRY_RUN:-false}"
WEBHOOK_URL="${WEBHOOK_URL:-}"

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Logging functions
log_info() {
    echo -e "${GREEN}[INFO]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

log_warn() {
    echo -e "${YELLOW}[WARN]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

log_debug() {
    echo -e "${BLUE}[DEBUG]${NC} $(date '+%Y-%m-%d %H:%M:%S') - $*"
}

# Send notification via webhook
send_notification() {
    local title="$1"
    local message="$2"
    local severity="${3:-info}"
    
    if [[ -n "$WEBHOOK_URL" ]]; then
        local payload
        payload=$(cat <<EOF
{
  "title": "$title",
  "message": "$message",
  "severity": "$severity",
  "namespace": "$NAMESPACE",
  "timestamp": "$(date -u +%Y-%m-%dT%H:%M:%SZ)"
}
EOF
)
        
        if curl -s -X POST "$WEBHOOK_URL" \
            -H "Content-Type: application/json" \
            -d "$payload" > /dev/null; then
            log_debug "Notification sent successfully"
        else
            log_warn "Failed to send webhook notification"
        fi
    fi
}

# Check if deployment has previous revision
has_previous_revision() {
    local deployment="$1"
    local revision_count
    revision_count=$(kubectl rollout history deployment/"$deployment" -n "$NAMESPACE" 2>/dev/null | grep -c "^[0-9]" || echo 0)
    [[ $revision_count -gt 1 ]]
}

# Get deployment health metrics
get_deployment_health() {
    local deployment="$1"
    local desired ready unavailable updated

    desired=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.spec.replicas}' 2>/dev/null || echo 0)
    ready=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.status.readyReplicas}' 2>/dev/null || echo 0)
    unavailable=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.status.unavailableReplicas}' 2>/dev/null || echo 0)
    updated=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.status.updatedReplicas}' 2>/dev/null || echo 0)
    
    echo "desired=$desired ready=$ready unavailable=${unavailable:-0} updated=$updated"
}

# Get current and previous revision numbers
get_revision_info() {
    local deployment="$1"
    local current_revision previous_revision
    
    current_revision=$(kubectl get deployment "$deployment" -n "$NAMESPACE" -o jsonpath='{.metadata.annotations.deployment\.kubernetes\.io/revision}' 2>/dev/null || echo "unknown")
    previous_revision=$((current_revision - 1))
    
    echo "current=$current_revision previous=$previous_revision"
}

# Verify rollback was successful
verify_rollback() {
    local deployment="$1"
    local timeout="${2:-60}"
    local end_time=$((SECONDS + timeout))
    
    log_info "Verifying rollback health..."
    
    while [[ $SECONDS -lt $end_time ]]; do
        local health
        health=$(get_deployment_health "$deployment")
        
        local desired ready unavailable
        desired=$(echo "$health" | grep -oP 'desired=\K\d+')
        ready=$(echo "$health" | grep -oP 'ready=\K\d+')
        unavailable=$(echo "$health" | grep -oP 'unavailable=\K\d+')
        
        if [[ "$ready" -eq "$desired" ]] && [[ "$unavailable" -eq 0 ]]; then
            log_info "Rollback verification successful: $ready/$desired pods ready"
            return 0
        fi
        
        log_debug "Waiting for pods to be ready: $ready/$desired (unavailable: $unavailable)"
        sleep 5
    done
    
    log_error "Rollback verification timed out"
    return 1
}

# Get recent pod events
get_pod_events() {
    local deployment="$1"
    log_debug "Recent pod events for $deployment:"
    kubectl get events -n "$NAMESPACE" \
        --field-selector involvedObject.name="$deployment" \
        --sort-by='.lastTimestamp' \
        --no-headers 2>/dev/null | tail -10 || true
}

# Perform rollback for a single deployment
rollback_deployment() {
    local deployment="$1"
    local reason="${2:-Manual rollback}"
    
    log_info "=========================================="
    log_info "Starting rollback for deployment: $deployment"
    log_info "Reason: $reason"
    log_info "=========================================="

    # Check if previous revision exists
    if ! has_previous_revision "$deployment"; then
        log_error "No previous revision found for $deployment, cannot rollback"
        send_notification "Rollback Failed" "No previous revision for $deployment" "error"
        return 1
    fi

    # Get revision info
    local revision_info
    revision_info=$(get_revision_info "$deployment")
    log_info "Revision info: $revision_info"

    # Get current health status
    local health
    health=$(get_deployment_health "$deployment")
    log_info "Current deployment health: $health"
    
    # Get recent events
    get_pod_events "$deployment"

    # Dry run mode
    if [[ "$DRY_RUN" == "true" ]]; then
        log_warn "=========================================="
        log_warn "DRY RUN MODE: Would rollback deployment $deployment"
        log_warn "=========================================="
        send_notification "Rollback Dry Run" "Would rollback $deployment: $reason" "warning"
        return 0
    fi

    # Create Kubernetes event
    kubectl create event \
        --namespace="$NAMESPACE" \
        --type=Warning \
        --reason=AutoRollback \
        --message="Auto-rollback triggered: $reason" \
        --reporting-controller=edgelink-rollback \
        --reporting-instance="$(hostname)" \
        --action=Rollback \
        2>/dev/null || log_warn "Failed to create Kubernetes event"

    # Execute rollback
    log_info "Executing rollback command..."
    if kubectl rollout undo deployment/"$deployment" -n "$NAMESPACE"; then
        log_info "Rollback command executed successfully"
    else
        log_error "Rollback command failed"
        send_notification "Rollback Failed" "kubectl rollout undo failed for $deployment" "error"
        return 1
    fi

    # Wait for rollback to complete
    log_info "Waiting for rollback to complete (timeout: ${ROLLBACK_TIMEOUT}s)..."
    if kubectl rollout status deployment/"$deployment" -n "$NAMESPACE" --timeout="${ROLLBACK_TIMEOUT}s"; then
        log_info "Rollback deployment status check passed"
    else
        log_error "Rollback status check failed or timed out"
        send_notification "Rollback Timeout" "Rollback timed out for $deployment" "error"
        return 1
    fi
    
    # Verify rollback health
    if verify_rollback "$deployment" 60; then
        # Get new health status
        local new_health
        new_health=$(get_deployment_health "$deployment")
        log_info "New deployment health: $new_health"
        
        log_info "=========================================="
        log_info "✅ Rollback completed successfully for $deployment"
        log_info "=========================================="
        
        send_notification "Rollback Successful" "$deployment rolled back successfully: $reason" "success"
        return 0
    else
        log_error "Rollback health verification failed"
        send_notification "Rollback Unhealthy" "Rollback completed but health check failed for $deployment" "error"
        return 1
    fi
}

# Show usage
usage() {
    cat <<EOF
Usage: $0 <deployment-name> [reason]

Automatic rollback script for EdgeLink Kubernetes deployments.

Arguments:
  deployment-name    Name of the deployment to rollback
  reason            (Optional) Reason for rollback

Environment Variables:
  NAMESPACE              Kubernetes namespace (default: edgelink)
  DEPLOYMENT_PREFIX      Deployment name prefix filter (default: edgelink)
  MAX_ROLLBACK_RETRIES   Maximum rollback attempts (default: 3)
  ROLLBACK_TIMEOUT       Rollback timeout in seconds (default: 300)
  DRY_RUN               Dry run mode, no actual changes (default: false)
  WEBHOOK_URL           Webhook URL for notifications

Examples:
  # Rollback specific deployment
  $0 edgelink-api-gateway "High error rate detected"
  
  # Dry run mode
  DRY_RUN=true $0 edgelink-device-service
  
  # With webhook notification
  WEBHOOK_URL=https://hooks.slack.com/... $0 edgelink-topology-service

EOF
}

# Main rollback orchestration
main() {
    log_info "EdgeLink Auto-Rollback Script Starting"
    log_info "Namespace: $NAMESPACE"
    log_info "Deployment Prefix: $DEPLOYMENT_PREFIX"
    log_info "Dry Run: $DRY_RUN"
    log_info "Max Retries: $MAX_ROLLBACK_RETRIES"
    log_info "Timeout: ${ROLLBACK_TIMEOUT}s"

    # Parse arguments
    if [[ $# -eq 0 ]]; then
        usage
        exit 1
    fi

    local target_deployment="$1"
    local rollback_reason="${2:-Automatic rollback triggered}"

    # Validate deployment exists
    if ! kubectl get deployment "$target_deployment" -n "$NAMESPACE" &>/dev/null; then
        log_error "Deployment $target_deployment not found in namespace $NAMESPACE"
        log_info "Available deployments:"
        kubectl get deployments -n "$NAMESPACE" -o name | sed 's/deployment.apps\///'
        exit 1
    fi

    # Execute rollback with retries
    local retry_count=0
    while [[ $retry_count -lt $MAX_ROLLBACK_RETRIES ]]; do
        if [[ $retry_count -gt 0 ]]; then
            log_warn "Retry attempt $((retry_count + 1)) of $MAX_ROLLBACK_RETRIES"
            sleep 10
        fi

        if rollback_deployment "$target_deployment" "$rollback_reason"; then
            log_info "Rollback operation completed successfully"
            exit 0
        else
            ((retry_count++))
        fi
    done

    log_error "=========================================="
    log_error "❌ Rollback failed after $MAX_ROLLBACK_RETRIES attempts"
    log_error "=========================================="
    send_notification "Rollback Failed" "All retry attempts exhausted for $target_deployment" "critical"
    exit 1
}

# Run main function
main "$@"
