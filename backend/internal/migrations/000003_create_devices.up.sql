-- 创建 NAT 类型枚举
CREATE TYPE nat_type_enum AS ENUM (
    'full_cone',
    'restricted_cone',
    'port_restricted_cone',
    'symmetric',
    'unknown'
);

-- 创建平台类型枚举
CREATE TYPE platform_enum AS ENUM (
    'desktop_linux',
    'desktop_windows',
    'desktop_macos',
    'mobile_ios',
    'mobile_android',
    'iot',
    'container'
);

-- 创建 devices 表
CREATE TABLE IF NOT EXISTS devices (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    virtual_network_id UUID NOT NULL REFERENCES virtual_networks(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    virtual_ip INET NOT NULL,
    public_key TEXT NOT NULL UNIQUE,
    platform platform_enum NOT NULL,
    nat_type nat_type_enum DEFAULT 'unknown',
    online BOOLEAN NOT NULL DEFAULT FALSE,
    last_seen_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- 确保同一虚拟网络内 IP 不重复
    UNIQUE(virtual_network_id, virtual_ip)
);

-- 创建索引
CREATE INDEX idx_devices_virtual_network_id ON devices(virtual_network_id);
CREATE INDEX idx_devices_online ON devices(online);
CREATE INDEX idx_devices_last_seen_at ON devices(last_seen_at);
CREATE INDEX idx_devices_public_key ON devices(public_key);
CREATE INDEX idx_devices_platform ON devices(platform);
CREATE INDEX idx_devices_created_at ON devices(created_at);

-- 创建更新时间触发器
CREATE TRIGGER update_devices_updated_at
    BEFORE UPDATE ON devices
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
