-- 创建诊断包状态枚举
CREATE TYPE diagnostic_status_enum AS ENUM (
    'requested',
    'collecting',
    'uploaded',
    'failed',
    'expired'
);

-- 创建 diagnostic_bundles 表
CREATE TABLE IF NOT EXISTS diagnostic_bundles (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    device_id UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    status diagnostic_status_enum NOT NULL DEFAULT 'requested',
    s3_object_key TEXT,
    s3_bucket VARCHAR(255),
    file_size_bytes BIGINT,
    include_logs BOOLEAN NOT NULL DEFAULT TRUE,
    include_wireguard_stats BOOLEAN NOT NULL DEFAULT TRUE,
    include_network_trace BOOLEAN NOT NULL DEFAULT FALSE,
    collection_duration_seconds INTEGER DEFAULT 60,
    error_message TEXT,
    requested_by UUID,
    requested_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    completed_at TIMESTAMP WITH TIME ZONE,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_diagnostic_bundles_device_id ON diagnostic_bundles(device_id);
CREATE INDEX idx_diagnostic_bundles_status ON diagnostic_bundles(status);
CREATE INDEX idx_diagnostic_bundles_requested_at ON diagnostic_bundles(requested_at DESC);
CREATE INDEX idx_diagnostic_bundles_expires_at ON diagnostic_bundles(expires_at) WHERE expires_at IS NOT NULL;

CREATE TRIGGER update_diagnostic_bundles_updated_at
    BEFORE UPDATE ON diagnostic_bundles
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();
