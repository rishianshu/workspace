-- UCL & Nucleus Integration Schema
-- Migration: 002_nucleus_integration.sql

-- =================
-- Projects Table (synced from Nucleus)
-- =================
CREATE TABLE IF NOT EXISTS projects (
    id VARCHAR(255) PRIMARY KEY,
    nucleus_project_id VARCHAR(255) UNIQUE,
    slug VARCHAR(255) NOT NULL,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    synced_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_projects_nucleus ON projects(nucleus_project_id);
CREATE INDEX IF NOT EXISTS idx_projects_slug ON projects(slug);

-- =================
-- Endpoints Table (replicated from Nucleus)
-- =================
CREATE TABLE IF NOT EXISTS endpoints (
    id VARCHAR(255) PRIMARY KEY,
    nucleus_endpoint_id VARCHAR(255) NOT NULL UNIQUE,
    project_id VARCHAR(255) REFERENCES projects(id) ON DELETE SET NULL,
    template_id VARCHAR(255) NOT NULL,   -- e.g., 'http.jira'
    display_name VARCHAR(255) NOT NULL,
    source_system VARCHAR(100),          -- 'jira', 'github', etc.
    capabilities TEXT[],
    config JSONB,                         -- Non-sensitive config only
    synced_at TIMESTAMPTZ DEFAULT NOW(),
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_endpoints_project ON endpoints(project_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_nucleus ON endpoints(nucleus_endpoint_id);
CREATE INDEX IF NOT EXISTS idx_endpoints_template ON endpoints(template_id);

-- =================
-- Credential Store (Key Store)
-- =================
CREATE TABLE IF NOT EXISTS credential_store (
    key_token VARCHAR(255) PRIMARY KEY,  -- Opaque reference token
    owner_type VARCHAR(50) NOT NULL,     -- 'user' | 'org' | 'service'
    owner_id VARCHAR(255) NOT NULL,      -- user_id or org_id
    endpoint_id VARCHAR(255) NOT NULL,   -- Nucleus endpoint instance
    credentials JSONB NOT NULL,          -- Encrypted OAuth/API tokens
    credential_type VARCHAR(50),         -- 'oauth2' | 'api_key' | 'basic'
    scopes TEXT[],                        -- OAuth scopes granted
    expires_at TIMESTAMPTZ,
    refreshed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_creds_owner ON credential_store(owner_type, owner_id);
CREATE INDEX IF NOT EXISTS idx_creds_endpoint ON credential_store(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_creds_expires ON credential_store(expires_at);

-- =================
-- User Endpoint Bindings
-- =================
CREATE TABLE IF NOT EXISTS user_endpoint_bindings (
    id VARCHAR(255) PRIMARY KEY,
    user_id VARCHAR(255) NOT NULL,
    endpoint_id VARCHAR(255) NOT NULL REFERENCES endpoints(id) ON DELETE CASCADE,
    key_token VARCHAR(255) NOT NULL,     -- Reference to credential_store
    is_active BOOLEAN DEFAULT TRUE,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(user_id, endpoint_id)
);

CREATE INDEX IF NOT EXISTS idx_bindings_user ON user_endpoint_bindings(user_id);
CREATE INDEX IF NOT EXISTS idx_bindings_endpoint ON user_endpoint_bindings(endpoint_id);
CREATE INDEX IF NOT EXISTS idx_bindings_active ON user_endpoint_bindings(user_id, is_active) WHERE is_active = TRUE;

-- =================
-- Helper: Generate key token
-- =================
CREATE OR REPLACE FUNCTION generate_key_token()
RETURNS VARCHAR(255) AS $$
BEGIN
    RETURN 'kt_' || encode(gen_random_bytes(24), 'base64');
END;
$$ LANGUAGE plpgsql;
