-- 创建连接类型枚举
CREATE TYPE connection_type_enum AS ENUM ('p2p_direct', 'turn_relay');

-- 创建 sessions 表
CREATE TABLE IF NOT EXISTS sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_a_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    device_b_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    connection_type connection_type_enum NOT NULL,
    started_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    ended_at TIMESTAMP WITH TIME ZONE,
    last_handshake_at TIMESTAMP WITH TIME ZONE,
    bytes_sent BIGINT DEFAULT 0,
    bytes_received BIGINT DEFAULT 0,
    avg_latency_ms INTEGER,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),

    CHECK(device_a_id < device_b_id) -- 确保设备ID有序，避免重复会话
);

CREATE INDEX idx_sessions_device_a_id ON sessions(device_a_id);
CREATE INDEX idx_sessions_device_b_id ON sessions(device_b_id);
CREATE INDEX idx_sessions_started_at ON sessions(started_at);
CREATE INDEX idx_sessions_ended_at ON sessions(ended_at) WHERE ended_at IS NULL;

CREATE TRIGGER update_sessions_updated_at
    BEFORE UPDATE ON sessions
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
