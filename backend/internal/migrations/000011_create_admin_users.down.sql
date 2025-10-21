DROP TRIGGER IF EXISTS update_admin_users_updated_at ON admin_users;
DROP INDEX IF EXISTS idx_admin_users_organization_id;
DROP INDEX IF EXISTS idx_admin_users_email;
DROP INDEX IF EXISTS idx_admin_users_role;
DROP INDEX IF EXISTS idx_admin_users_is_active;
DROP INDEX IF EXISTS idx_admin_users_oidc_subject;
DROP TABLE IF EXISTS admin_users;
DROP TYPE IF EXISTS role_enum;
