#!/bin/bash

# Edge-Link 管理端停止脚本

set -euo pipefail

# 颜色定义
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# 日志函数
log_info() {
    echo -e "${BLUE}[INFO]${NC} $1"
}

log_success() {
    echo -e "${GREEN}[SUCCESS]${NC} $1"
}

log_warning() {
    echo -e "${YELLOW}[WARNING]${NC} $1"
}

log_error() {
    echo -e "${RED}[ERROR]${NC} $1"
}

# 停止服务
stop_services() {
    log_info "停止Edge-Link管理端服务..."

    # 停止后端服务
    if [ -f backend/.backend.pid ]; then
        BACKEND_PID=$(cat backend/.backend.pid)
        if kill -0 $BACKEND_PID 2>/dev/null; then
            kill $BACKEND_PID
            log_success "已停止后端服务 (PID: $BACKEND_PID)"
        else
            log_warning "后端服务进程不存在 (PID: $BACKEND_PID)"
        fi
        rm -f backend/.backend.pid
    else
        log_warning "未找到后端服务PID文件"
    fi

    # 停止前端服务
    if [ -f frontend/.frontend.pid ]; then
        FRONTEND_PID=$(cat frontend/.frontend.pid)
        if kill -0 $FRONTEND_PID 2>/dev/null; then
            kill $FRONTEND_PID
            log_success "已停止前端服务 (PID: $FRONTEND_PID)"
        else
            log_warning "前端服务进程不存在 (PID: $FRONTEND_PID)"
        fi
        rm -f frontend/.frontend.pid
    else
        log_warning "未找到前端服务PID文件"
    fi

    # 强制停止可能残留的Node.js进程
    if pgrep -f "npm run dev" > /dev/null; then
        pkill -f "npm run dev"
        log_info "已强制停止残留的前端进程"
    fi

    # 强制停止可能残留的Go进程
    if pgrep -f "api-gateway" > /dev/null; then
        pkill -f "api-gateway"
        log_info "已强制停止残留的后端进程"
    fi
}

# 停止Docker基础设施
stop_infrastructure() {
    if [ -f docker-compose.dev.yml ]; then
        log_info "停止Docker基础设施..."
        docker-compose -f docker-compose.dev.yml down
        log_success "Docker基础设施已停止"
    else
        log_warning "未找到Docker Compose配置文件"
    fi
}

# 清理函数
cleanup() {
    log_info "清理临时文件..."
    rm -f backend/.backend.pid
    rm -f frontend/.frontend.pid
    log_success "临时文件已清理"
}

# 主函数
main() {
    echo "Edge-Link 管理端停止脚本"
    echo "=========================="

    stop_services
    stop_infrastructure
    cleanup

    log_success "Edge-Link 管理端已完全停止！"
}

# 运行主函数
main "$@"