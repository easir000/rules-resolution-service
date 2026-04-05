-- Migration: 001_create_schema.sql
-- Core schema for Rules Resolution Service

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Workflow steps (canonical definitions)
CREATE TABLE steps (
    key VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    position INTEGER NOT NULL UNIQUE CHECK (position > 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Default values for each step/trait (specificity 0)
CREATE TABLE defaults (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    step_key VARCHAR(50) NOT NULL REFERENCES steps(key) ON DELETE CASCADE,
    trait_key VARCHAR(50) NOT NULL,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(step_key, trait_key)
);

-- Multi-dimensional override records
CREATE TABLE overrides (
    id VARCHAR(20) PRIMARY KEY,
    
    -- Target: which step/trait this override modifies
    step_key VARCHAR(50) NOT NULL REFERENCES steps(key) ON DELETE CASCADE,
    trait_key VARCHAR(50) NOT NULL,
    
    -- Selector dimensions (NULL = wildcard)
    state VARCHAR(2) NULL,
    client VARCHAR(50) NULL,
    investor VARCHAR(50) NULL,
    case_type VARCHAR(30) NULL,
    
    -- Computed specificity: count of non-null selector dimensions
    specificity INTEGER GENERATED ALWAYS AS (
        CASE WHEN state IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN client IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN investor IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN case_type IS NOT NULL THEN 1 ELSE 0 END
    ) STORED,
    
    -- Value to apply when this override wins
    value JSONB NOT NULL,
    
    -- Effective dating
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expires_date DATE NULL,
    
    -- Lifecycle status
    status VARCHAR(20) NOT NULL DEFAULT 'draft' 
        CHECK (status IN ('draft', 'active', 'archived')),
    
    -- Metadata
    description TEXT,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Index for efficient resolution queries
CREATE INDEX idx_overrides_resolve 
ON overrides (step_key, trait_key, state, client, investor, case_type, specificity DESC, effective_date DESC)
WHERE status = 'active';

-- Index for conflict detection
CREATE INDEX idx_overrides_conflict 
ON overrides (step_key, trait_key, specificity, state, client, investor, case_type)
WHERE status IN ('active', 'draft');

-- Audit trail for all override changes
CREATE TABLE override_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    override_id VARCHAR(20) NOT NULL REFERENCES overrides(id) ON DELETE CASCADE,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    changed_by VARCHAR(100) NOT NULL,
    change_type VARCHAR(20) NOT NULL 
        CHECK (change_type IN ('create', 'update', 'status_change', 'delete')),
    before_state JSONB,
    after_state JSONB,
    summary TEXT
);

-- Auto-update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER update_steps_updated_at BEFORE UPDATE ON steps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_defaults_updated_at BEFORE UPDATE ON defaults FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER update_overrides_updated_at BEFORE UPDATE ON overrides FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- Auto-log override changes to history
CREATE OR REPLACE FUNCTION log_override_change()
RETURNS TRIGGER AS $$
DECLARE
    change_type VARCHAR(20);
    before_json JSONB;
    after_json JSONB;
    summary TEXT;
BEGIN
    IF TG_OP = 'INSERT' THEN
        change_type := 'create';
        before_json := NULL;
        after_json := to_jsonb(NEW);
        summary := format('Created override %s', NEW.id);
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.status IS DISTINCT FROM NEW.status THEN
            change_type := 'status_change';
            summary := format('Status: %s → %s', OLD.status, NEW.status);
        ELSE
            change_type := 'update';
            summary := 'Override updated';
        END IF;
        before_json := to_jsonb(OLD);
        after_json := to_jsonb(NEW);
    ELSIF TG_OP = 'DELETE' THEN
        change_type := 'delete';
        before_json := to_jsonb(OLD);
        after_json := NULL;
        summary := format('Deleted override %s', OLD.id);
    END IF;
    
    INSERT INTO override_history (override_id, changed_by, change_type, before_state, after_state, summary)
    VALUES (COALESCE(NEW.id, OLD.id), COALESCE(NEW.updated_by, OLD.created_by, 'system'), change_type, before_json, after_json, summary);
    
    RETURN COALESCE(NEW, OLD);
END;
$$ language 'plpgsql';

CREATE TRIGGER trg_override_audit
AFTER INSERT OR UPDATE OR DELETE ON overrides
FOR EACH ROW EXECUTE FUNCTION log_override_change();