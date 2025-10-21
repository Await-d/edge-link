#!/bin/bash
#
# EdgeLink Dependency Mirrors Monitor
#
# 监控依赖镜像源的可用性和性能
#

set -euo pipefail

# Colors
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

log_ok() {
    echo -e "${GREEN}✅${NC} $*"
}

log_warn() {
    echo -e "${YELLOW}⚠️${NC} $*"
}

log_error() {
    echo -e "${RED}❌${NC} $*"
}

# Check Go Proxy
check_go_proxy() {
    local proxy="$1"
    echo "Checking Go Proxy: $proxy"
    
    if curl -sf --max-time 10 "$proxy" > /dev/null; then
        log_ok "Go Proxy $proxy is accessible"
        return 0
    else
        log_error "Go Proxy $proxy is down"
        return 1
    fi
}

# Check npm Registry
check_npm_registry() {
    local registry="$1"
    echo "Checking npm Registry: $registry"
    
    if curl -sf --max-time 10 "$registry/react" > /dev/null; then
        log_ok "npm Registry $registry is accessible"
        return 0
    else
        log_error "npm Registry $registry is down"
        return 1
    fi
}

# Check Docker Mirror
check_docker_mirror() {
    local mirror="$1"
    echo "Checking Docker Mirror: $mirror"
    
    if curl -sf --max-time 10 "$mirror/v2/" > /dev/null; then
        log_ok "Docker Mirror $mirror is accessible"
        return 0
    else
        log_error "Docker Mirror $mirror is down"
        return 1
    fi
}

# Main
main() {
    echo "=========================================="
    echo "EdgeLink Dependency Mirrors Health Check"
    echo "$(date '+%Y-%m-%d %H:%M:%S')"
    echo "=========================================="
    echo ""
    
    local failures=0
    
    # Go Proxies
    echo "## Go Module Proxies"
    check_go_proxy "https://goproxy.cn" || ((failures++))
    check_go_proxy "https://mirrors.aliyun.com/goproxy/" || ((failures++))
    check_go_proxy "https://proxy.golang.org" || ((failures++))
    echo ""
    
    # npm Registries
    echo "## npm Registries"
    check_npm_registry "https://registry.npmmirror.com" || ((failures++))
    check_npm_registry "https://registry.npmjs.org" || ((failures++))
    echo ""
    
    # Docker Mirrors
    echo "## Docker Registry Mirrors"
    check_docker_mirror "https://docker.mirrors.ustc.edu.cn" || ((failures++))
    check_docker_mirror "https://hub-mirror.c.163.com" || ((failures++))
    echo ""
    
    echo "=========================================="
    if [ $failures -eq 0 ]; then
        log_ok "All dependency mirrors are healthy"
        exit 0
    else
        log_error "$failures mirror(s) failed health check"
        exit 1
    fi
}

main "$@"
