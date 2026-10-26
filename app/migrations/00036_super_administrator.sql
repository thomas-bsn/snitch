-- +goose Up
-- Third role: the Super-administrator owns the installation. It is unique (a partial unique index),
-- and the database itself refuses to demote or delete it, so no code path can leave the instance
-- without an owner. An existing installation keeps every access: its oldest Administrator becomes
-- the Super-administrator, the others stay Administrators.
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('super_admin','admin','user'));
UPDATE users SET role='super_admin'
 WHERE id=(SELECT id FROM users WHERE role='admin' ORDER BY created_at,id LIMIT 1);
CREATE UNIQUE INDEX users_single_super_admin ON users ((true)) WHERE role='super_admin';

-- +goose StatementBegin
CREATE FUNCTION protect_super_admin() RETURNS trigger LANGUAGE plpgsql AS $$
BEGIN
 IF TG_OP='DELETE' THEN
  RAISE EXCEPTION 'super_administrator_protected: the Super-administrator cannot be deleted' USING ERRCODE='23514';
 END IF;
 RAISE EXCEPTION 'super_administrator_protected: the Super-administrator cannot be demoted' USING ERRCODE='23514';
END $$;
-- +goose StatementEnd
CREATE TRIGGER users_protect_super_admin_update BEFORE UPDATE OF role ON users
 FOR EACH ROW WHEN (OLD.role='super_admin' AND NEW.role<>'super_admin') EXECUTE FUNCTION protect_super_admin();
CREATE TRIGGER users_protect_super_admin_delete BEFORE DELETE ON users
 FOR EACH ROW WHEN (OLD.role='super_admin') EXECUTE FUNCTION protect_super_admin();

-- +goose Down
DROP TRIGGER users_protect_super_admin_delete ON users;
DROP TRIGGER users_protect_super_admin_update ON users;
DROP FUNCTION protect_super_admin();
DROP INDEX users_single_super_admin;
UPDATE users SET role='admin' WHERE role='super_admin';
ALTER TABLE users DROP CONSTRAINT users_role_check;
ALTER TABLE users ADD CONSTRAINT users_role_check CHECK (role IN ('admin','user'));
