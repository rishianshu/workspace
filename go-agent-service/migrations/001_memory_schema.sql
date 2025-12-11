-- ADK Memory System Schema
-- Run: psql -U postgres -d agent -f 001_memory_schema.sql

-- Enable pgvector extension
CREATE EXTENSION IF NOT EXISTS vector;

-- =================
-- Sessions Table (Short-term state)
-- =================
CREATE TABLE IF NOT EXISTS sessions (
    id VARCHAR(255) PRIMARY KEY,
    conversation_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255),
    summary TEXT DEFAULT '',
    state JSONB DEFAULT '{}',
    turn_count INTEGER DEFAULT 0,
    last_activity TIMESTAMP WITH TIME ZONE DEFAULT NOW(),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_sessions_conversation ON sessions(conversation_id);
CREATE INDEX idx_sessions_last_activity ON sessions(last_activity);

-- =================
-- Turns Table (Episodic memory)
-- =================
CREATE TABLE IF NOT EXISTS turns (
    id VARCHAR(255) PRIMARY KEY,
    session_id VARCHAR(255) NOT NULL REFERENCES sessions(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,  -- 'user' or 'assistant'
    content TEXT NOT NULL,
    summary TEXT DEFAULT '',
    embedding vector(768),      -- Gemini text-embedding-004 dimension
    compressed BOOLEAN DEFAULT FALSE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_turns_session ON turns(session_id);
CREATE INDEX idx_turns_created ON turns(created_at);

-- Vector similarity index (IVFFlat for fast approximate search)
CREATE INDEX idx_turns_embedding ON turns 
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- =================
-- Facts Table (Semantic memory)
-- =================
CREATE TABLE IF NOT EXISTS facts (
    id VARCHAR(255) PRIMARY KEY,
    entity_id VARCHAR(255) NOT NULL,
    session_id VARCHAR(255) REFERENCES sessions(id) ON DELETE SET NULL,
    type VARCHAR(100) NOT NULL,  -- 'resolved', 'mentioned', 'acted_on', 'created'
    content TEXT NOT NULL,
    source VARCHAR(100) NOT NULL,  -- 'jira', 'github', 'agent', 'user'
    embedding vector(768),
    created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
);

CREATE INDEX idx_facts_entity ON facts(entity_id);
CREATE INDEX idx_facts_session ON facts(session_id);
CREATE INDEX idx_facts_type ON facts(type);

-- Vector similarity index for facts
CREATE INDEX idx_facts_embedding ON facts 
    USING ivfflat (embedding vector_cosine_ops)
    WITH (lists = 100);

-- =================
-- Helper Functions
-- =================

-- Function to search turns by semantic similarity
CREATE OR REPLACE FUNCTION search_turns_by_embedding(
    p_session_id VARCHAR(255),
    p_embedding vector(768),
    p_limit INTEGER DEFAULT 5
)
RETURNS TABLE (
    id VARCHAR(255),
    role VARCHAR(50),
    content TEXT,
    summary TEXT,
    similarity FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        t.id,
        t.role,
        t.content,
        t.summary,
        1 - (t.embedding <=> p_embedding) AS similarity
    FROM turns t
    WHERE t.session_id = p_session_id
      AND t.embedding IS NOT NULL
    ORDER BY t.embedding <=> p_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;

-- Function to search facts by semantic similarity
CREATE OR REPLACE FUNCTION search_facts_by_embedding(
    p_embedding vector(768),
    p_limit INTEGER DEFAULT 5
)
RETURNS TABLE (
    id VARCHAR(255),
    entity_id VARCHAR(255),
    type VARCHAR(100),
    content TEXT,
    source VARCHAR(100),
    similarity FLOAT
) AS $$
BEGIN
    RETURN QUERY
    SELECT 
        f.id,
        f.entity_id,
        f.type,
        f.content,
        f.source,
        1 - (f.embedding <=> p_embedding) AS similarity
    FROM facts f
    WHERE f.embedding IS NOT NULL
    ORDER BY f.embedding <=> p_embedding
    LIMIT p_limit;
END;
$$ LANGUAGE plpgsql;
