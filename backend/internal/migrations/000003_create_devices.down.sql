-- 删除触发器
DROP TRIGGER IF EXISTS update_devices_updated_at ON devices;

-- 删除索引
DROP INDEX IF EXISTS idx_devices_virtual_network_id;
DROP INDEX IF EXISTS idx_devices_online;
DROP INDEX IF EXISTS idx_devices_last_seen_at;
DROP INDEX IF EXISTS idx_devices_public_key;
DROP INDEX IF EXISTS idx_devices_platform;
DROP INDEX IF EXISTS idx_devices_created_at;

-- 删除表
DROP TABLE IF EXISTS devices;

-- 删除枚举类型
DROP TYPE IF EXISTS nat_type_enum;
DROP TYPE IF EXISTS platform_enum;
