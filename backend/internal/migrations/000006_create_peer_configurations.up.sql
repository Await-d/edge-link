-- 创建 peer_configurations 表
CREATE TABLE IF NOT EXISTS peer_configurations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    peer_device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    peer_public_key TEXT NOT NULL,
    peer_virtual_ip INET NOT NULL,
    allowed_ips CIDR[] NOT NULL DEFAULT '{}',
    persistent_keepalive INTEGER DEFAULT 25,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    -- 确保设备对之间的配置唯一
    UNIQUE(device_id, peer_device_id),
    -- 防止设备配置自己为对等设备
    CHECK(device_id != peer_device_id)
);

CREATE INDEX idx_peer_configurations_device_id ON peer_configurations(device_id);
CREATE INDEX idx_peer_configurations_peer_device_id ON peer_configurations(peer_device_id);

CREATE TRIGGER update_peer_configurations_updated_at
    BEFORE UPDATE ON peer_configurations
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
