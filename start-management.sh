#!/bin/bash

# Edge-Link 管理端启动脚本
# 用于测试前端和后端集成

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

# 检查依赖
check_dependencies() {
    log_info "检查依赖..."

    local missing_deps=()

    # 检查Docker
    if ! command -v docker &> /dev/null; then
        missing_deps+=("docker")
    fi

    # 检查Docker Compose
    if ! command -v docker-compose &> /dev/null; then
        missing_deps+=("docker-compose")
    fi

    # 检查Node.js
    if ! command -v node &> /dev/null; then
        missing_deps+=("node")
    fi

    # 检查Go
    if ! command -v go &> /dev/null; then
        missing_deps+=("go")
    fi

    if [ ${#missing_deps[@]} -gt 0 ]; then
        log_error "缺少依赖: ${missing_deps[*]}"
        echo "请安装缺少的依赖后重试"
        exit 1
    fi

    log_success "所有依赖检查通过"
}

# 启动数据库和Redis
start_infrastructure() {
    log_info "启动基础设施数据库和Redis..."

    # 检查Docker是否运行
    if ! docker info &> /dev/null; then
        log_error "Docker未运行，请先启动Docker"
        exit 1
    fi

    # 启动PostgreSQL和Redis
    docker-compose -f docker-compose.dev.yml up -d postgres redis

    # 等待服务启动
    log_info "等待数据库启动..."
    sleep 10

    # 检查服务状态
    if docker-compose -f docker-compose.dev.yml ps postgres | grep -q "Up"; then
        log_success "PostgreSQL已启动"
    else
        log_error "PostgreSQL启动失败"
        docker-compose -f docker-compose.dev.yml logs postgres
        exit 1
    fi

    if docker-compose -f docker-compose.dev.yml ps redis | grep -q "Up"; then
        log_success "Redis已启动"
    else
        log_error "Redis启动失败"
        docker-compose -f docker-compose.dev.yml logs redis
        exit 1
    fi
}

# 运行数据库迁移
run_migrations() {
    log_info "运行数据库迁移..."

    cd backend
    if go run cmd/migrate/main.go up; then
        log_success "数据库迁移完成"
    else
        log_error "数据库迁移失败"
        exit 1
    fi
    cd ..
}

# 构建后端
build_backend() {
    log_info "构建后端服务..."

    cd backend
    if go build -o bin/api-gateway ./cmd/api-gateway/; then
        log_success "后端构建完成"
    else
        log_error "后端构建失败"
        exit 1
    fi
    cd ..
}

# 启动后端服务
start_backend() {
    log_info "启动后端API服务..."

    cd backend
    # 后台启动API网关
    ./bin/api-gateway &
    BACKEND_PID=$!
    echo $BACKEND_PID > .backend.pid

    # 等待服务启动
    log_info "等待后端服务启动..."
    sleep 5

    # 检查服务是否启动成功
    if curl -s http://localhost:8080/health | grep -q "healthy"; then
        log_success "后端服务已启动 (PID: $BACKEND_PID)"
    else
        log_error "后端服务启动失败"
        kill $BACKEND_PID 2>/dev/null || true
        exit 1
    fi
    cd ..
}

# 安装前端依赖
install_frontend_deps() {
    log_info "安装前端依赖..."

    cd frontend
    if npm install; then
        log_success "前端依赖安装完成"
    else
        log_error "前端依赖安装失败"
        exit 1
    fi
    cd ..
}

# 启动前端开发服务器
start_frontend() {
    log_info "启动前端开发服务器..."

    cd frontend
    # 后台启动前端开发服务器
    npm run dev &
    FRONTEND_PID=$!
    echo $FRONTEND_PID > .frontend.pid

    # 等待服务启动
    log_info "等待前端服务启动..."
    sleep 10

    # 检查服务是否启动成功
    if curl -s http://localhost:5173 | grep -q "Edge-Link"; then
        log_success "前端服务已启动 (PID: $FRONTEND_PID)"
    else
        log_warning "前端服务可能仍在启动中"
    fi
    cd ..
}

# 显示服务信息
show_service_info() {
    echo ""
    echo "========================================"
    echo "Edge-Link 管理端已启动"
    echo "========================================"
    echo ""
    echo "前端地址: http://localhost:5173"
    echo "后端API: http://localhost:8080"
    echo "WebSocket: ws://localhost:8080/ws"
    echo ""
    echo "API端点:"
    echo "  - 健康检查: http://localhost:8080/health"
    echo "  - 设备管理: http://localhost:8080/api/v1/admin/devices"
    echo "  - 仪表板统计: http://localhost:8080/api/v1/stats/dashboard"
    echo "  - 网络拓扑: http://localhost:8080/api/v1/topology/devices"
    echo ""
    echo "进程ID:"
    if [ -f backend/.backend.pid ]; then
        echo "  - 后端: $(cat backend/.backend.pid)"
    fi
    if [ -f frontend/.frontend.pid ]; then
        echo "  - 前端: $(cat frontend/.frontend.pid)"
    fi
    echo ""
    echo "停止服务: ./stop-management.sh"
    echo "========================================"
}

# 创建Docker Compose开发配置
create_docker_compose() {
    if [ ! -f docker-compose.dev.yml ]; then
        log_info "创建Docker Compose开发配置..."
        cat > docker-compose.dev.yml << 'EOF'
version: '3.8'

services:
  postgres:
    image: postgres:15
    environment:
      POSTGRES_DB: edge_link
      POSTGRES_USER: postgres
      POSTGRES_PASSWORD: postgres
    ports:
      - "5432:5432"
    volumes:
      - postgres_data:/var/lib/postgresql/data
    healthcheck:
      test: ["CMD-SHELL", "pg_isready -U postgres"]
      interval: 10s
      timeout: 5s
      retries: 5

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"
    command: redis-server --appendonly yes
    volumes:
      - redis_data:/data
    healthcheck:
      test: ["CMD", "redis-cli", "ping"]
      interval: 10s
      timeout: 5s
      retries: 5

volumes:
  postgres_data:
  redis_data:
EOF
        log_success "Docker Compose配置已创建"
    fi
}

# 主函数
main() {
    echo "Edge-Link 管理端启动脚本"
    echo "=========================="

    # 检查是否在项目根目录
    if [ ! -f "backend/go.mod" ] || [ ! -f "frontend/package.json" ]; then
        log_error "请在项目根目录运行此脚本"
        exit 1
    fi

    # 创建Docker Compose配置
    create_docker_compose

    # 执行启动步骤
    check_dependencies
    start_infrastructure
    run_migrations
    build_backend
    start_backend
    install_frontend_deps
    start_frontend
    show_service_info

    log_success "Edge-Link 管理端启动完成！"
}

# 信号处理
trap 'log_warning "接收到中断信号，正在清理..."; cleanup; exit 130' INT TERM

# 清理函数
cleanup() {
    log_info "清理进程..."

    if [ -f backend/.backend.pid ]; then
        BACKEND_PID=$(cat backend/.backend.pid)
        if kill -0 $BACKEND_PID 2>/dev/null; then
            kill $BACKEND_PID
            log_info "已停止后端服务 (PID: $BACKEND_PID)"
        fi
        rm -f backend/.backend.pid
    fi

    if [ -f frontend/.frontend.pid ]; then
        FRONTEND_PID=$(cat frontend/.frontend.pid)
        if kill -0 $FRONTEND_PID 2>/dev/null; then
            kill $FRONTEND_PID
            log_info "已停止前端服务 (PID: $FRONTEND_PID)"
        fi
        rm -f frontend/.frontend.pid
    fi
}

# 运行主函数
main "$@"