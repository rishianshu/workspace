-- App Registry Schema
-- Migration: 003_app_registry.sql

-- =================
-- App Instances (shared config identity)
-- =================
CREATE TABLE IF NOT EXISTS app_instances (
    id VARCHAR(255) PRIMARY KEY,
    template_id VARCHAR(255) NOT NULL,
    instance_key VARCHAR(255) NOT NULL,
    display_name VARCHAR(255),
    config JSONB,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(template_id, instance_key)
);

CREATE INDEX IF NOT EXISTS idx_app_instances_template ON app_instances(template_id);
CREATE INDEX IF NOT EXISTS idx_app_instances_key ON app_instances(instance_key);

-- =================
-- User Apps (per-user binding + credential ref)
-- =================
CREATE TABLE IF NOT EXISTS user_apps (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    app_instance_id VARCHAR(255) NOT NULL REFERENCES app_instances(id) ON DELETE CASCADE,
    credential_ref VARCHAR(255),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, app_instance_id)
);

CREATE INDEX IF NOT EXISTS idx_user_apps_user ON user_apps(user_id);
CREATE INDEX IF NOT EXISTS idx_user_apps_instance ON user_apps(app_instance_id);

-- =================
-- Project Apps (project linkage + endpoint binding)
-- =================
CREATE TABLE IF NOT EXISTS project_apps (
    id VARCHAR(255) PRIMARY KEY,
    project_id VARCHAR(255) NOT NULL,
    user_app_id VARCHAR(255) NOT NULL REFERENCES user_apps(id) ON DELETE CASCADE,
    endpoint_id VARCHAR(255) NOT NULL,
    alias VARCHAR(255),
    is_default BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(project_id, user_app_id)
);

CREATE INDEX IF NOT EXISTS idx_project_apps_project ON project_apps(project_id);
CREATE INDEX IF NOT EXISTS idx_project_apps_user_app ON project_apps(user_app_id);
CREATE INDEX IF NOT EXISTS idx_project_apps_endpoint ON project_apps(endpoint_id);
