-- 删除触发器
DROP TRIGGER IF EXISTS update_organizations_updated_at ON organizations;

-- 删除索引
DROP INDEX IF EXISTS idx_organizations_slug;
DROP INDEX IF EXISTS idx_organizations_created_at;

-- 删除表
DROP TABLE IF EXISTS organizations;

-- 删除触发器函数（如果没有其他表使用）
-- 注意：由于其他表可能也使用这个函数，所以这里不删除
-- DROP FUNCTION IF EXISTS update_updated_at_column();
