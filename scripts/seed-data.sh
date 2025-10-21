#!/bin/bash
# EdgeLink种子数据脚本 - 创建测试组织、虚拟网络和PSK

set -e

# 颜色输出
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${BLUE}EdgeLink种子数据生成${NC}"
echo "=================================="

# 数据库连接信息
DB_HOST="${DB_HOST:-localhost}"
DB_PORT="${DB_PORT:-5432}"
DB_NAME="${DB_NAME:-edgelink}"
DB_USER="${DB_USER:-edgelink}"
DB_PASSWORD="${DB_PASSWORD:-edgelink_dev_password}"

# 检查PostgreSQL连接
echo -e "\n${YELLOW}检查数据库连接...${NC}"
if ! PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME -c "SELECT 1" > /dev/null 2>&1; then
    echo -e "${YELLOW}无法直接连接PostgreSQL，尝试通过Docker...${NC}"
    PSQL_CMD="docker exec -i edgelink-postgres psql -U $DB_USER -d $DB_NAME"
else
    PSQL_CMD="PGPASSWORD=$DB_PASSWORD psql -h $DB_HOST -p $DB_PORT -U $DB_USER -d $DB_NAME"
fi

echo -e "${GREEN}✓${NC} 数据库连接成功"

# 生成UUID函数
generate_uuid() {
    if command -v uuidgen &> /dev/null; then
        uuidgen | tr '[:upper:]' '[:lower:]'
    else
        cat /proc/sys/kernel/random/uuid
    fi
}

# 1. 创建测试组织
echo -e "\n${YELLOW}创建测试组织...${NC}"
ORG_ID=$(generate_uuid)
ORG_SLUG="demo-org"

$PSQL_CMD << EOF
INSERT INTO organizations (id, name, slug, created_at, updated_at)
VALUES (
    '$ORG_ID',
    'Demo Organization',
    '$ORG_SLUG',
    NOW(),
    NOW()
)
ON CONFLICT (slug) DO NOTHING;
EOF

echo -e "${GREEN}✓${NC} 组织创建完成: $ORG_SLUG (ID: $ORG_ID)"

# 2. 创建虚拟网络
echo -e "\n${YELLOW}创建虚拟网络...${NC}"
VN_ID=$(generate_uuid)

$PSQL_CMD << EOF
INSERT INTO virtual_networks (id, organization_id, name, cidr, created_at, updated_at)
VALUES (
    '$VN_ID',
    '$ORG_ID',
    'Demo VPN Network',
    '10.100.0.0/16',
    NOW(),
    NOW()
)
ON CONFLICT DO NOTHING;
EOF

echo -e "${GREEN}✓${NC} 虚拟网络创建完成: Demo VPN Network (ID: $VN_ID)"
echo -e "   CIDR: 10.100.0.0/16"

# 3. 创建预共享密钥（PSK）
echo -e "\n${YELLOW}创建预共享密钥...${NC}"
PSK_ID=$(generate_uuid)
PSK_KEY="demo-psk-$(date +%s)"
# 简化的PSK哈希（生产环境应使用HMAC-SHA256）
PSK_HASH=$(echo -n "$PSK_KEY" | sha256sum | cut -d' ' -f1)

$PSQL_CMD << EOF
INSERT INTO pre_shared_keys (id, organization_id, key_hash, max_uses, used_count, expires_at, created_at)
VALUES (
    '$PSK_ID',
    '$ORG_ID',
    '$PSK_HASH',
    100,
    0,
    NOW() + INTERVAL '30 days',
    NOW()
)
ON CONFLICT DO NOTHING;
EOF

echo -e "${GREEN}✓${NC} 预共享密钥创建完成"
echo -e "   PSK ID: $PSK_ID"
echo -e "   PSK: ${YELLOW}$PSK_KEY${NC}"
echo -e "   最大使用次数: 100"
echo -e "   过期时间: 30天后"

# 4. 保存配置到文件
CONFIG_FILE="/tmp/edgelink-test-config.env"
cat > $CONFIG_FILE << EOF
# EdgeLink测试环境配置
# 生成时间: $(date)

ORGANIZATION_SLUG=$ORG_SLUG
ORGANIZATION_ID=$ORG_ID
VIRTUAL_NETWORK_ID=$VN_ID
PRE_SHARED_KEY=$PSK_KEY
CONTROL_PLANE_URL=http://localhost:8080

# 使用示例:
# edgelink-cli register \\
#   --control-plane http://localhost:8080 \\
#   --psk $PSK_KEY \\
#   --org $ORG_SLUG \\
#   --network $VN_ID \\
#   --name my-device
EOF

echo -e "\n${GREEN}========================================${NC}"
echo -e "${GREEN}种子数据创建完成！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo "配置已保存到: $CONFIG_FILE"
echo ""
echo "测试参数:"
echo "  组织: $ORG_SLUG"
echo "  虚拟网络ID: $VN_ID"
echo "  预共享密钥: $PSK_KEY"
echo ""
echo "注册设备命令示例:"
echo -e "${YELLOW}  edgelink-cli register \\
    --control-plane http://localhost:8080 \\
    --psk $PSK_KEY \\
    --org $ORG_SLUG \\
    --network $VN_ID \\
    --name test-device-1${NC}"
