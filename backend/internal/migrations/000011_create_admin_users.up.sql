-- 创建角色枚举
CREATE TYPE role_enum AS ENUM (
    'super_admin',
    'admin',
    'network_operator',
    'auditor',
    'readonly'
);

-- 创建 admin_users 表
CREATE TABLE IF NOT EXISTS admin_users (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    email VARCHAR(255) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    role role_enum NOT NULL DEFAULT 'readonly',
    oidc_subject VARCHAR(255),
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    last_login_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMP WITH TIME ZONE NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_admin_users_organization_id ON admin_users(organization_id);
CREATE INDEX idx_admin_users_email ON admin_users(email);
CREATE INDEX idx_admin_users_role ON admin_users(role);
CREATE INDEX idx_admin_users_is_active ON admin_users(is_active);
CREATE INDEX idx_admin_users_oidc_subject ON admin_users(oidc_subject) WHERE oidc_subject IS NOT NULL;

CREATE TRIGGER update_admin_users_updated_at
    BEFORE UPDATE ON admin_users
    FOR EACH ROW
    EXECUTE FUNCTION update_updated_at_column();

-- 插入示例管理员（开发环境）
INSERT INTO admin_users (organization_id, email, name, role)
SELECT
    id,
    'admin@edgelink.local',
    'Admin User',
    'super_admin'::role_enum
FROM organizations
WHERE slug = 'dev-org'
ON CONFLICT (email) DO NOTHING;
