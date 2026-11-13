-- +goose Up
-- GORM filled these columns itself, so the tables AutoMigrate built have no database default and
-- a row inserted outside GORM got NULL there. The database now fills the timestamps, and the users
-- columns porte/kit does not name get the values modules/auth's OnCreate writes.

ALTER TABLE activity_logs ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE app_folders ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE app_objects ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE app_objects ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE directory_deletion_state ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE entitlement_state ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE file_versions ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE files ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE files ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE folders ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE folders ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE nook_deliveries ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE registered_apps ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE registered_apps ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE settings ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE shares ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE space_members ALTER COLUMN joined_at SET DEFAULT now();
ALTER TABLE spaces ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE spaces ALTER COLUMN updated_at SET DEFAULT now();
ALTER TABLE upload_chunks ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE upload_sessions ALTER COLUMN created_at SET DEFAULT now();
ALTER TABLE user_quotas ALTER COLUMN updated_at SET DEFAULT now();

ALTER TABLE users
    ALTER COLUMN created_at SET DEFAULT now(),
    ALTER COLUMN avatar_url SET DEFAULT '',
    ALTER COLUMN avatar_source SET DEFAULT '',
    ALTER COLUMN avatar_upload_path SET DEFAULT '',
    ALTER COLUMN password_hash SET DEFAULT '',
    ALTER COLUMN oidc_access_token SET DEFAULT '',
    ALTER COLUMN oidc_refresh_token SET DEFAULT '',
    ALTER COLUMN oidc_token_expiry SET DEFAULT '0001-01-01 00:00:00+00',
    ALTER COLUMN profile_synced_at SET DEFAULT '0001-01-01 00:00:00+00';

-- +goose Down
ALTER TABLE users
    ALTER COLUMN created_at DROP DEFAULT,
    ALTER COLUMN avatar_url DROP DEFAULT,
    ALTER COLUMN avatar_source DROP DEFAULT,
    ALTER COLUMN avatar_upload_path DROP DEFAULT,
    ALTER COLUMN password_hash DROP DEFAULT,
    ALTER COLUMN oidc_access_token DROP DEFAULT,
    ALTER COLUMN oidc_refresh_token DROP DEFAULT,
    ALTER COLUMN oidc_token_expiry DROP DEFAULT,
    ALTER COLUMN profile_synced_at DROP DEFAULT;

ALTER TABLE user_quotas ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE upload_sessions ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE upload_chunks ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE spaces ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE spaces ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE space_members ALTER COLUMN joined_at DROP DEFAULT;
ALTER TABLE shares ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE settings ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE registered_apps ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE registered_apps ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE nook_deliveries ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE folders ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE folders ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE files ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE files ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE file_versions ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE entitlement_state ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE directory_deletion_state ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE app_objects ALTER COLUMN updated_at DROP DEFAULT;
ALTER TABLE app_objects ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE app_folders ALTER COLUMN created_at DROP DEFAULT;
ALTER TABLE activity_logs ALTER COLUMN created_at DROP DEFAULT;
