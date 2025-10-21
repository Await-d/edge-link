-- 添加告警去重相关字段
ALTER TABLE alerts
ADD COLUMN IF NOT EXISTS occurrence_count INTEGER NOT NULL DEFAULT 1,
ADD COLUMN IF NOT EXISTS first_seen_at TIMESTAMP NOT NULL DEFAULT NOW(),
ADD COLUMN IF NOT EXISTS last_seen_at TIMESTAMP NOT NULL DEFAULT NOW();

-- 为新字段添加索引以优化查询
CREATE INDEX IF NOT EXISTS idx_alerts_last_seen_at ON alerts(last_seen_at);
CREATE INDEX IF NOT EXISTS idx_alerts_device_type_status ON alerts(device_id, type, status);

-- 添加注释
COMMENT ON COLUMN alerts.occurrence_count IS '告警出现次数（用于去重统计）';
COMMENT ON COLUMN alerts.first_seen_at IS '告警首次出现时间';
COMMENT ON COLUMN alerts.last_seen_at IS '告警最后出现时间';
