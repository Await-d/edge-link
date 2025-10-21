-- 创建 virtual_networks 表
CREATE TABLE IF NOT EXISTS virtual_networks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    cidr CIDR NOT NULL,
    gateway_ip INET NOT NULL,
    dns_servers INET[] DEFAULT '{}',
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- 确保同一组织内 CIDR 不重复
    UNIQUE(organization_id, cidr)
);

-- 创建索引
CREATE INDEX idx_virtual_networks_organization_id ON virtual_networks(organization_id);
CREATE INDEX idx_virtual_networks_cidr ON virtual_networks USING GIST(cidr inet_ops);
CREATE INDEX idx_virtual_networks_created_at ON virtual_networks(created_at);

-- 创建更新时间触发器
CREATE TRIGGER update_virtual_networks_updated_at
    BEFORE UPDATE ON virtual_networks
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 插入示例虚拟网络（开发环境）
INSERT INTO virtual_networks (organization_id, name, cidr, gateway_ip, dns_servers)
SELECT
    id,
    'Default VPN Network',
    '10.100.0.0/16'::CIDR,
    '10.100.0.1'::INET,
    ARRAY['8.8.8.8'::INET, '8.8.4.4'::INET]
FROM organizations
WHERE slug = 'dev-org'
ON CONFLICT DO NOTHING;
