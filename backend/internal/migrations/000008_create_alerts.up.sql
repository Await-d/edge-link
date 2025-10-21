-- 创建告警严重程度枚举
CREATE TYPE severity_enum AS ENUM ('critical', 'high', 'medium', 'low');

-- 创建告警类型枚举
CREATE TYPE alert_type_enum AS ENUM (
    'device_offline',
    'high_latency',
    'failed_auth',
    'key_expiration',
    'tunnel_failure'
);

-- 创建告警状态枚举
CREATE TYPE alert_status_enum AS ENUM ('active', 'acknowledged', 'resolved');

-- 创建 alerts 表
CREATE TABLE IF NOT EXISTS alerts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    severity severity_enum NOT NULL,
    type alert_type_enum NOT NULL,
    message TEXT NOT NULL,
    metadata JSONB DEFAULT '{}',
    status alert_status_enum NOT NULL DEFAULT 'active',
    acknowledged_by UUID,
    acknowledged_at TIMESTAMP WITH TIME ZONE,
    resolved_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_alerts_device_id ON alerts(device_id);
CREATE INDEX idx_alerts_severity ON alerts(severity);
CREATE INDEX idx_alerts_type ON alerts(type);
CREATE INDEX idx_alerts_status ON alerts(status);
CREATE INDEX idx_alerts_created_at ON alerts(created_at);
CREATE INDEX idx_alerts_metadata ON alerts USING GIN(metadata);

CREATE TRIGGER update_alerts_updated_at
    BEFORE UPDATE ON alerts
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
