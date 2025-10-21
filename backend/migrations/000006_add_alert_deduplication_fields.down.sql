-- 回滚：删除索引
DROP INDEX IF EXISTS idx_alerts_device_type_status;
DROP INDEX IF EXISTS idx_alerts_last_seen_at;

-- 回滚：删除字段
ALTER TABLE alerts
DROP COLUMN IF EXISTS last_seen_at,
DROP COLUMN IF EXISTS first_seen_at,
DROP COLUMN IF EXISTS occurrence_count;
