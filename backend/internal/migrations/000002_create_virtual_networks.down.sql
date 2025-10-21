-- 删除触发器
DROP TRIGGER IF EXISTS update_virtual_networks_updated_at ON virtual_networks;

-- 删除索引
DROP INDEX IF EXISTS idx_virtual_networks_organization_id;
DROP INDEX IF EXISTS idx_virtual_networks_cidr;
DROP INDEX IF EXISTS idx_virtual_networks_created_at;

-- 删除表
DROP TABLE IF EXISTS virtual_networks;
