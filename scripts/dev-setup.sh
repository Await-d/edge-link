#!/bin/bash
# EdgeLink本地开发环境设置脚本

set -e

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${BLUE}EdgeLink开发环境设置${NC}"
echo "=================================="

# 检查Docker
if ! command -v docker &> /dev/null; then
    echo -e "${RED}错误: Docker未安装${NC}"
    echo "请访问 https://docs.docker.com/get-docker/ 安装Docker"
    exit 1
fi

# 检查Docker Compose
if ! command -v docker-compose &> /dev/null && ! docker compose version &> /dev/null; then
    echo -e "${RED}错误: Docker Compose未安装${NC}"
    echo "请访问 https://docs.docker.com/compose/install/ 安装Docker Compose"
    exit 1
fi

# 设置Docker Compose命令
if docker compose version &> /dev/null; then
    DOCKER_COMPOSE="docker compose"
else
    DOCKER_COMPOSE="docker-compose"
fi

echo -e "${GREEN}✓${NC} Docker和Docker Compose已就绪"

# 停止并删除现有容器
echo -e "\n${YELLOW}清理现有容器...${NC}"
$DOCKER_COMPOSE down -v || true

# 构建镜像
echo -e "\n${YELLOW}构建Docker镜像...${NC}"
$DOCKER_COMPOSE build

# 启动服务
echo -e "\n${YELLOW}启动服务...${NC}"
$DOCKER_COMPOSE up -d postgres redis

# 等待PostgreSQL就绪
echo -e "\n${YELLOW}等待PostgreSQL就绪...${NC}"
until docker exec edgelink-postgres pg_isready -U edgelink > /dev/null 2>&1; do
    echo -n "."
    sleep 1
done
echo -e "\n${GREEN}✓${NC} PostgreSQL已就绪"

# 运行数据库迁移
echo -e "\n${YELLOW}运行数据库迁移...${NC}"
# TODO: 需要迁移工具（golang-migrate或自定义迁移脚本）
# docker run --rm --network edge-link_edgelink-network \
#   -v $(pwd)/backend/internal/migrations:/migrations \
#   migrate/migrate -path=/migrations -database "postgresql://edgelink:edgelink_dev_password@postgres:5432/edgelink?sslmode=disable" up

echo -e "${YELLOW}注意: 数据库迁移需要手动运行（migrate工具）${NC}"

# 启动微服务
echo -e "\n${YELLOW}启动微服务...${NC}"
$DOCKER_COMPOSE up -d

# 等待服务启动
echo -e "\n${YELLOW}等待服务启动...${NC}"
sleep 10

# 检查服务状态
echo -e "\n${YELLOW}检查服务状态...${NC}"
$DOCKER_COMPOSE ps

# 检查API Gateway健康状态
echo -e "\n${YELLOW}检查API Gateway健康状态...${NC}"
if curl -s http://localhost:8080/health > /dev/null; then
    echo -e "${GREEN}✓${NC} API Gateway健康检查通过"
else
    echo -e "${YELLOW}警告: API Gateway健康检查失败（可能仍在启动中）${NC}"
fi

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}开发环境设置完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "服务地址:"
echo "  API Gateway:       http://localhost:8080"
echo "  PostgreSQL:        localhost:5432"
echo "  Redis:             localhost:6379"
echo "  Device Service:    localhost:50051 (gRPC)"
echo "  Topology Service:  localhost:50052 (gRPC)"
echo "  NAT Coordinator:   localhost:50053 (gRPC)"
echo ""
echo "有用的命令:"
echo "  查看日志:    $DOCKER_COMPOSE logs -f [service]"
echo "  停止服务:    $DOCKER_COMPOSE down"
echo "  重启服务:    $DOCKER_COMPOSE restart [service]"
echo "  进入容器:    docker exec -it edgelink-[service] sh"
echo ""
echo "下一步:"
echo "  1. 运行种子数据脚本: ./scripts/seed-data.sh"
echo "  2. 测试设备注册: 参见 specs/001-edge-link-core/quickstart.md"
