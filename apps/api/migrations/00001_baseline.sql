-- +goose Up
-- baseline: the production schema as GORM AutoMigrate and schemas.MigrateWithIssuer built it, captured 2026-11-13 with pg_dump --schema-only --no-owner --no-privileges

CREATE TABLE activity_logs (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    event_type text NOT NULL,
    resource_type text NOT NULL,
    resource_id bigint,
    resource_name text,
    metadata text,
    created_at timestamp with time zone
);

CREATE SEQUENCE activity_logs_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE activity_logs_id_seq OWNED BY activity_logs.id;

CREATE TABLE antenne_deliveries (
    id bigint NOT NULL,
    event_type text NOT NULL,
    payload text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts bigint DEFAULT 0 NOT NULL,
    next_retry_at timestamp with time zone,
    response_code bigint,
    response_body text,
    error_message text,
    latency_ms bigint,
    created_at timestamp with time zone,
    delivered_at timestamp with time zone
);

CREATE SEQUENCE antenne_deliveries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE antenne_deliveries_id_seq OWNED BY antenne_deliveries.id;

CREATE TABLE api_token_scopes (
    session_id bigint NOT NULL,
    app text DEFAULT ''::text NOT NULL,
    space_facile_ids jsonb DEFAULT '[]'::jsonb NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    provision boolean DEFAULT false NOT NULL
);

CREATE TABLE app_folders (
    id bigint NOT NULL,
    user_id bigint NOT NULL,
    app_id bigint NOT NULL,
    created_at timestamp with time zone
);

CREATE SEQUENCE app_folders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE app_folders_id_seq OWNED BY app_folders.id;

CREATE TABLE app_objects (
    id bigint NOT NULL,
    folder_id bigint NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    mime_type text,
    size bigint,
    hash text,
    bucket_key text NOT NULL,
    source_file_id bigint,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE app_objects_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE app_objects_id_seq OWNED BY app_objects.id;

CREATE TABLE directory_deletion_state (
    id bigint NOT NULL,
    state bytea NOT NULL,
    updated_at timestamp with time zone
);

CREATE SEQUENCE directory_deletion_state_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE directory_deletion_state_id_seq OWNED BY directory_deletion_state.id;

CREATE TABLE entitlement_state (
    id bigint NOT NULL,
    document text NOT NULL,
    updated_at timestamp with time zone
);

CREATE SEQUENCE entitlement_state_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE entitlement_state_id_seq OWNED BY entitlement_state.id;

CREATE TABLE file_versions (
    id bigint NOT NULL,
    file_id bigint NOT NULL,
    version bigint NOT NULL,
    bucket_key text NOT NULL,
    hash text,
    size bigint,
    created_by bigint NOT NULL,
    created_at timestamp with time zone,
    bucket text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE file_versions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE file_versions_id_seq OWNED BY file_versions.id;

CREATE TABLE files (
    id bigint NOT NULL,
    facile_id text NOT NULL,
    name text NOT NULL,
    mime_type text,
    size bigint,
    bucket_key text NOT NULL,
    folder_id bigint,
    origin_app text,
    linked_to text,
    uploaded_by bigint NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    deleted_at timestamp with time zone,
    hash text,
    space_id bigint,
    bucket text DEFAULT ''::text NOT NULL
);

CREATE SEQUENCE files_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE files_id_seq OWNED BY files.id;

CREATE TABLE folders (
    id bigint NOT NULL,
    facile_id text NOT NULL,
    name text NOT NULL,
    parent_id bigint,
    owner_id bigint NOT NULL,
    created_at timestamp with time zone,
    deleted_at timestamp with time zone,
    updated_at timestamp with time zone,
    space_id bigint
);

CREATE SEQUENCE folders_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE folders_id_seq OWNED BY folders.id;

CREATE TABLE mcp_actions (
    id text NOT NULL,
    user_id text NOT NULL,
    kind text NOT NULL,
    target text NOT NULL,
    digest text NOT NULL,
    summary text DEFAULT ''::text NOT NULL,
    client_id text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    resolved_at timestamp with time zone,
    outcome text DEFAULT ''::text NOT NULL
);

CREATE TABLE nook_deliveries (
    id bigint NOT NULL,
    event_type text NOT NULL,
    payload text NOT NULL,
    status text DEFAULT 'pending'::text NOT NULL,
    attempts bigint DEFAULT 0 NOT NULL,
    next_retry_at timestamp with time zone,
    response_code bigint,
    response_body text,
    error_message text,
    latency_ms bigint,
    created_at timestamp with time zone,
    delivered_at timestamp with time zone
);

CREATE SEQUENCE nook_deliveries_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE nook_deliveries_id_seq OWNED BY nook_deliveries.id;

CREATE TABLE porte_identities (
    user_id bigint NOT NULL,
    provider text NOT NULL,
    subject text NOT NULL,
    password_hash text DEFAULT ''::text NOT NULL,
    access_token text DEFAULT ''::text NOT NULL,
    refresh_token text DEFAULT ''::text NOT NULL,
    token_expiry timestamp with time zone,
    roles jsonb,
    roles_synced_at timestamp with time zone,
    synced_at timestamp with time zone,
    created_at timestamp with time zone DEFAULT now()
);

CREATE TABLE porte_login_codes (
    code_hash text NOT NULL,
    user_id bigint NOT NULL,
    expires_at timestamp with time zone NOT NULL,
    consumed_at timestamp with time zone,
    provider text DEFAULT ''::text NOT NULL,
    subject text DEFAULT ''::text NOT NULL
);

CREATE TABLE porte_sessions (
    id bigint NOT NULL,
    token_hash text NOT NULL,
    user_id bigint NOT NULL,
    label text DEFAULT ''::text NOT NULL,
    created_at timestamp with time zone DEFAULT now() NOT NULL,
    last_used_at timestamp with time zone DEFAULT now() NOT NULL,
    expires_at timestamp with time zone
);

CREATE SEQUENCE porte_sessions_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE porte_sessions_id_seq OWNED BY porte_sessions.id;

CREATE TABLE registered_apps (
    id bigint NOT NULL,
    key text NOT NULL,
    name text NOT NULL,
    client_id text NOT NULL,
    active boolean DEFAULT true NOT NULL,
    created_at timestamp with time zone,
    updated_at timestamp with time zone
);

CREATE SEQUENCE registered_apps_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE registered_apps_id_seq OWNED BY registered_apps.id;

CREATE TABLE settings (
    key text NOT NULL,
    value text,
    updated_at timestamp with time zone
);

CREATE TABLE shares (
    id bigint NOT NULL,
    token text NOT NULL,
    file_id bigint,
    folder_id bigint,
    shared_by bigint NOT NULL,
    permission text DEFAULT 'view'::text NOT NULL,
    expires_at timestamp with time zone,
    created_at timestamp with time zone,
    space_id bigint
);

CREATE SEQUENCE shares_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE shares_id_seq OWNED BY shares.id;

CREATE TABLE space_members (
    id bigint NOT NULL,
    space_id bigint NOT NULL,
    user_id bigint NOT NULL,
    role text DEFAULT 'member'::text NOT NULL,
    joined_at timestamp with time zone
);

CREATE SEQUENCE space_members_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE space_members_id_seq OWNED BY space_members.id;

CREATE TABLE spaces (
    id bigint NOT NULL,
    facile_id text NOT NULL,
    name text NOT NULL,
    description text,
    created_at timestamp with time zone,
    updated_at timestamp with time zone,
    provisioned_by text DEFAULT ''::text NOT NULL,
    registre_space_id text,
    charged_to bigint,
    billing_offer_to bigint,
    billing_offer_end timestamp with time zone
);

CREATE SEQUENCE spaces_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE spaces_id_seq OWNED BY spaces.id;

CREATE TABLE tombstones (
    id bigint NOT NULL,
    resource_type text NOT NULL,
    resource_id bigint NOT NULL,
    facile_id text NOT NULL,
    name text,
    user_id bigint NOT NULL,
    space_id bigint,
    deleted_at timestamp with time zone NOT NULL
);

CREATE SEQUENCE tombstones_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE tombstones_id_seq OWNED BY tombstones.id;

CREATE TABLE upload_chunks (
    id bigint NOT NULL,
    session_id text NOT NULL,
    part_number bigint NOT NULL,
    bucket_key text NOT NULL,
    size bigint,
    hash text,
    created_at timestamp with time zone
);

CREATE SEQUENCE upload_chunks_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE upload_chunks_id_seq OWNED BY upload_chunks.id;

CREATE TABLE upload_sessions (
    id text NOT NULL,
    file_name text NOT NULL,
    mime_type text,
    folder_id bigint,
    origin_app text,
    user_id bigint NOT NULL,
    total_size bigint,
    status text DEFAULT 'pending'::text NOT NULL,
    expires_at timestamp with time zone,
    created_at timestamp with time zone,
    space_id bigint,
    linked_to text
);

CREATE TABLE user_quotas (
    user_id bigint NOT NULL,
    storage_used bigint DEFAULT 0 NOT NULL,
    storage_limit bigint DEFAULT 0 NOT NULL,
    updated_at timestamp with time zone
);

CREATE SEQUENCE user_quotas_user_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE user_quotas_user_id_seq OWNED BY user_quotas.user_id;

CREATE TABLE users (
    id bigint NOT NULL,
    email text,
    name text,
    avatar_url text,
    color text,
    password_hash text,
    created_at timestamp with time zone,
    is_admin boolean DEFAULT false,
    avatar_source text,
    oidc_picture_url text,
    oidc_access_token text,
    oidc_refresh_token text,
    oidc_token_expiry timestamp with time zone,
    profile_synced_at timestamp with time zone,
    oidc_subject text,
    avatar_upload_path text,
    locale text DEFAULT 'en'::text NOT NULL
);

CREATE SEQUENCE users_id_seq
    START WITH 1
    INCREMENT BY 1
    NO MINVALUE
    NO MAXVALUE
    CACHE 1;

ALTER SEQUENCE users_id_seq OWNED BY users.id;

ALTER TABLE ONLY activity_logs ALTER COLUMN id SET DEFAULT nextval('activity_logs_id_seq'::regclass);

ALTER TABLE ONLY antenne_deliveries ALTER COLUMN id SET DEFAULT nextval('antenne_deliveries_id_seq'::regclass);

ALTER TABLE ONLY app_folders ALTER COLUMN id SET DEFAULT nextval('app_folders_id_seq'::regclass);

ALTER TABLE ONLY app_objects ALTER COLUMN id SET DEFAULT nextval('app_objects_id_seq'::regclass);

ALTER TABLE ONLY directory_deletion_state ALTER COLUMN id SET DEFAULT nextval('directory_deletion_state_id_seq'::regclass);

ALTER TABLE ONLY entitlement_state ALTER COLUMN id SET DEFAULT nextval('entitlement_state_id_seq'::regclass);

ALTER TABLE ONLY file_versions ALTER COLUMN id SET DEFAULT nextval('file_versions_id_seq'::regclass);

ALTER TABLE ONLY files ALTER COLUMN id SET DEFAULT nextval('files_id_seq'::regclass);

ALTER TABLE ONLY folders ALTER COLUMN id SET DEFAULT nextval('folders_id_seq'::regclass);

ALTER TABLE ONLY nook_deliveries ALTER COLUMN id SET DEFAULT nextval('nook_deliveries_id_seq'::regclass);

ALTER TABLE ONLY porte_sessions ALTER COLUMN id SET DEFAULT nextval('porte_sessions_id_seq'::regclass);

ALTER TABLE ONLY registered_apps ALTER COLUMN id SET DEFAULT nextval('registered_apps_id_seq'::regclass);

ALTER TABLE ONLY shares ALTER COLUMN id SET DEFAULT nextval('shares_id_seq'::regclass);

ALTER TABLE ONLY space_members ALTER COLUMN id SET DEFAULT nextval('space_members_id_seq'::regclass);

ALTER TABLE ONLY spaces ALTER COLUMN id SET DEFAULT nextval('spaces_id_seq'::regclass);

ALTER TABLE ONLY tombstones ALTER COLUMN id SET DEFAULT nextval('tombstones_id_seq'::regclass);

ALTER TABLE ONLY upload_chunks ALTER COLUMN id SET DEFAULT nextval('upload_chunks_id_seq'::regclass);

ALTER TABLE ONLY user_quotas ALTER COLUMN user_id SET DEFAULT nextval('user_quotas_user_id_seq'::regclass);

ALTER TABLE ONLY users ALTER COLUMN id SET DEFAULT nextval('users_id_seq'::regclass);

ALTER TABLE ONLY activity_logs
    ADD CONSTRAINT activity_logs_pkey PRIMARY KEY (id);

ALTER TABLE ONLY antenne_deliveries
    ADD CONSTRAINT antenne_deliveries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY api_token_scopes
    ADD CONSTRAINT api_token_scopes_pkey PRIMARY KEY (session_id);

ALTER TABLE ONLY app_folders
    ADD CONSTRAINT app_folders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY app_objects
    ADD CONSTRAINT app_objects_pkey PRIMARY KEY (id);

ALTER TABLE ONLY directory_deletion_state
    ADD CONSTRAINT directory_deletion_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY entitlement_state
    ADD CONSTRAINT entitlement_state_pkey PRIMARY KEY (id);

ALTER TABLE ONLY file_versions
    ADD CONSTRAINT file_versions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY files
    ADD CONSTRAINT files_pkey PRIMARY KEY (id);

ALTER TABLE ONLY folders
    ADD CONSTRAINT folders_pkey PRIMARY KEY (id);

ALTER TABLE ONLY mcp_actions
    ADD CONSTRAINT mcp_actions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY nook_deliveries
    ADD CONSTRAINT nook_deliveries_pkey PRIMARY KEY (id);

ALTER TABLE ONLY porte_identities
    ADD CONSTRAINT porte_identities_pkey PRIMARY KEY (provider, subject);

ALTER TABLE ONLY porte_login_codes
    ADD CONSTRAINT porte_login_codes_pkey PRIMARY KEY (code_hash);

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_token_hash_key UNIQUE (token_hash);

ALTER TABLE ONLY registered_apps
    ADD CONSTRAINT registered_apps_pkey PRIMARY KEY (id);

ALTER TABLE ONLY settings
    ADD CONSTRAINT settings_pkey PRIMARY KEY (key);

ALTER TABLE ONLY shares
    ADD CONSTRAINT shares_pkey PRIMARY KEY (id);

ALTER TABLE ONLY space_members
    ADD CONSTRAINT space_members_pkey PRIMARY KEY (id);

ALTER TABLE ONLY spaces
    ADD CONSTRAINT spaces_pkey PRIMARY KEY (id);

ALTER TABLE ONLY tombstones
    ADD CONSTRAINT tombstones_pkey PRIMARY KEY (id);

ALTER TABLE ONLY upload_chunks
    ADD CONSTRAINT upload_chunks_pkey PRIMARY KEY (id);

ALTER TABLE ONLY upload_sessions
    ADD CONSTRAINT upload_sessions_pkey PRIMARY KEY (id);

ALTER TABLE ONLY user_quotas
    ADD CONSTRAINT user_quotas_pkey PRIMARY KEY (user_id);

ALTER TABLE ONLY users
    ADD CONSTRAINT users_pkey PRIMARY KEY (id);

CREATE INDEX idx_activity_logs_created_at ON activity_logs USING btree (created_at);

CREATE INDEX idx_activity_logs_event_type ON activity_logs USING btree (event_type);

CREATE INDEX idx_activity_logs_resource_type ON activity_logs USING btree (resource_type);

CREATE INDEX idx_activity_logs_user_id ON activity_logs USING btree (user_id);

CREATE INDEX idx_antenne_deliveries_event_type ON antenne_deliveries USING btree (event_type);

CREATE INDEX idx_antenne_status_retry ON antenne_deliveries USING btree (status, next_retry_at);

CREATE UNIQUE INDEX idx_app_folder_user_app ON app_folders USING btree (user_id, app_id);

CREATE INDEX idx_app_folders_app_id ON app_folders USING btree (app_id);

CREATE UNIQUE INDEX idx_app_object_folder_key ON app_objects USING btree (folder_id, key);

CREATE UNIQUE INDEX idx_app_objects_source_file_id ON app_objects USING btree (source_file_id);

CREATE UNIQUE INDEX idx_chunk_session_part ON upload_chunks USING btree (session_id, part_number);

CREATE INDEX idx_file_versions_file_id ON file_versions USING btree (file_id);

CREATE INDEX idx_files_deleted_at ON files USING btree (deleted_at);

CREATE UNIQUE INDEX idx_files_facile_id ON files USING btree (facile_id);

CREATE INDEX idx_files_folder_id ON files USING btree (folder_id);

CREATE INDEX idx_files_hash ON files USING btree (hash);

CREATE INDEX idx_files_linked_to ON files USING btree (linked_to);

CREATE INDEX idx_files_space_id ON files USING btree (space_id);

CREATE INDEX idx_folders_deleted_at ON folders USING btree (deleted_at);

CREATE UNIQUE INDEX idx_folders_facile_id ON folders USING btree (facile_id);

CREATE INDEX idx_folders_parent_id ON folders USING btree (parent_id);

CREATE INDEX idx_folders_space_id ON folders USING btree (space_id);

CREATE INDEX idx_nook_deliveries_event_type ON nook_deliveries USING btree (event_type);

CREATE INDEX idx_nook_status_retry ON nook_deliveries USING btree (status, next_retry_at);

CREATE UNIQUE INDEX idx_registered_apps_client_id ON registered_apps USING btree (client_id);

CREATE UNIQUE INDEX idx_registered_apps_key ON registered_apps USING btree (key);

CREATE INDEX idx_shares_file_id ON shares USING btree (file_id);

CREATE INDEX idx_shares_folder_id ON shares USING btree (folder_id);

CREATE INDEX idx_shares_space_id ON shares USING btree (space_id);

CREATE UNIQUE INDEX idx_shares_token ON shares USING btree (token);

CREATE UNIQUE INDEX idx_space_user ON space_members USING btree (space_id, user_id);

CREATE INDEX idx_spaces_charged_to ON spaces USING btree (charged_to);

CREATE UNIQUE INDEX idx_spaces_facile_id ON spaces USING btree (facile_id);

CREATE UNIQUE INDEX idx_spaces_registre_space_id ON spaces USING btree (registre_space_id);

CREATE INDEX idx_tombstones_deleted_at ON tombstones USING btree (deleted_at);

CREATE INDEX idx_tombstones_facile_id ON tombstones USING btree (facile_id);

CREATE INDEX idx_tombstones_resource_id ON tombstones USING btree (resource_id);

CREATE INDEX idx_tombstones_resource_type ON tombstones USING btree (resource_type);

CREATE INDEX idx_tombstones_space_id ON tombstones USING btree (space_id);

CREATE INDEX idx_tombstones_user_id ON tombstones USING btree (user_id);

CREATE INDEX idx_upload_chunks_session_id ON upload_chunks USING btree (session_id);

CREATE INDEX idx_upload_sessions_user_id ON upload_sessions USING btree (user_id);

CREATE UNIQUE INDEX idx_users_email ON users USING btree (email);

CREATE UNIQUE INDEX idx_users_o_id_c_subject ON users USING btree (oidc_subject);

CREATE INDEX mcp_actions_open_idx ON mcp_actions USING btree (user_id, created_at DESC) WHERE (resolved_at IS NULL);

CREATE INDEX porte_identities_user_idx ON porte_identities USING btree (user_id);

CREATE INDEX porte_sessions_expiry_idx ON porte_sessions USING btree (expires_at) WHERE (expires_at IS NOT NULL);

CREATE INDEX porte_sessions_user_idx ON porte_sessions USING btree (user_id);

ALTER TABLE ONLY api_token_scopes
    ADD CONSTRAINT api_token_scopes_session_id_fkey FOREIGN KEY (session_id) REFERENCES porte_sessions(id) ON DELETE CASCADE;

ALTER TABLE ONLY activity_logs
    ADD CONSTRAINT fk_activity_logs_user FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY app_folders
    ADD CONSTRAINT fk_app_folders_app FOREIGN KEY (app_id) REFERENCES registered_apps(id) ON DELETE RESTRICT;

ALTER TABLE ONLY app_folders
    ADD CONSTRAINT fk_app_folders_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY app_objects
    ADD CONSTRAINT fk_app_objects_folder FOREIGN KEY (folder_id) REFERENCES app_folders(id) ON DELETE CASCADE;

ALTER TABLE ONLY file_versions
    ADD CONSTRAINT fk_file_versions_file FOREIGN KEY (file_id) REFERENCES files(id);

ALTER TABLE ONLY file_versions
    ADD CONSTRAINT fk_file_versions_user FOREIGN KEY (created_by) REFERENCES users(id);

ALTER TABLE ONLY files
    ADD CONSTRAINT fk_files_folder FOREIGN KEY (folder_id) REFERENCES folders(id);

ALTER TABLE ONLY files
    ADD CONSTRAINT fk_files_space FOREIGN KEY (space_id) REFERENCES spaces(id);

ALTER TABLE ONLY files
    ADD CONSTRAINT fk_files_user FOREIGN KEY (uploaded_by) REFERENCES users(id);

ALTER TABLE ONLY folders
    ADD CONSTRAINT fk_folders_parent FOREIGN KEY (parent_id) REFERENCES folders(id);

ALTER TABLE ONLY folders
    ADD CONSTRAINT fk_folders_space FOREIGN KEY (space_id) REFERENCES spaces(id);

ALTER TABLE ONLY folders
    ADD CONSTRAINT fk_folders_user FOREIGN KEY (owner_id) REFERENCES users(id);

ALTER TABLE ONLY shares
    ADD CONSTRAINT fk_shares_file FOREIGN KEY (file_id) REFERENCES files(id);

ALTER TABLE ONLY shares
    ADD CONSTRAINT fk_shares_folder FOREIGN KEY (folder_id) REFERENCES folders(id);

ALTER TABLE ONLY shares
    ADD CONSTRAINT fk_shares_owner FOREIGN KEY (shared_by) REFERENCES users(id);

ALTER TABLE ONLY shares
    ADD CONSTRAINT fk_shares_space FOREIGN KEY (space_id) REFERENCES spaces(id);

ALTER TABLE ONLY space_members
    ADD CONSTRAINT fk_space_members_space FOREIGN KEY (space_id) REFERENCES spaces(id) ON DELETE CASCADE;

ALTER TABLE ONLY space_members
    ADD CONSTRAINT fk_space_members_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY user_quotas
    ADD CONSTRAINT fk_user_quotas_user FOREIGN KEY (user_id) REFERENCES users(id);

ALTER TABLE ONLY porte_identities
    ADD CONSTRAINT porte_identities_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_login_codes
    ADD CONSTRAINT porte_login_codes_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

ALTER TABLE ONLY porte_sessions
    ADD CONSTRAINT porte_sessions_user_id_fkey FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE;

-- +goose Down
SELECT 'baseline migration is not reversible' WHERE false;
