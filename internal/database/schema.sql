-- internal/database/schema.sql
-- Rules Resolution Service - PostgreSQL Schema

CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- =============================================================================
-- TABLE: steps
-- =============================================================================
CREATE TABLE IF NOT EXISTS steps (
    key VARCHAR(50) PRIMARY KEY,
    name VARCHAR(100) NOT NULL,
    description TEXT,
    position INTEGER NOT NULL UNIQUE CHECK (position > 0),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- =============================================================================
-- TABLE: defaults
-- =============================================================================
CREATE TABLE IF NOT EXISTS defaults (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    step_key VARCHAR(50) NOT NULL REFERENCES steps(key) ON DELETE CASCADE,
    trait_key VARCHAR(50) NOT NULL,
    value JSONB NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW(),
    UNIQUE(step_key, trait_key)
);

-- =============================================================================
-- TABLE: overrides
-- =============================================================================
CREATE TABLE IF NOT EXISTS overrides (
    id VARCHAR(20) PRIMARY KEY,
    step_key VARCHAR(50) NOT NULL REFERENCES steps(key) ON DELETE CASCADE,
    trait_key VARCHAR(50) NOT NULL,
    
    -- Selector dimensions (NULL = wildcard)
    state VARCHAR(2) NULL,
    client VARCHAR(50) NULL,
    investor VARCHAR(50) NULL,
    case_type VARCHAR(30) NULL,
    
    -- Computed specificity (0-4)
    specificity INTEGER GENERATED ALWAYS AS (
        CASE WHEN state IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN client IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN investor IS NOT NULL THEN 1 ELSE 0 END +
        CASE WHEN case_type IS NOT NULL THEN 1 ELSE 0 END
    ) STORED,
    
    value JSONB NOT NULL,
    effective_date DATE NOT NULL DEFAULT CURRENT_DATE,
    expires_date DATE NULL,
    
    status VARCHAR(20) NOT NULL DEFAULT 'draft' 
        CHECK (status IN ('draft', 'active', 'archived')),
    
    description TEXT,
    created_by VARCHAR(100) NOT NULL,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_by VARCHAR(100),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

-- Indexes for efficient queries
CREATE INDEX IF NOT EXISTS idx_overrides_resolve 
ON overrides (step_key, trait_key, state, client, investor, case_type, specificity DESC, effective_date DESC)
WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_overrides_conflict 
ON overrides (step_key, trait_key, specificity, state, client, investor, case_type)
WHERE status IN ('active', 'draft');

CREATE INDEX IF NOT EXISTS idx_overrides_list 
ON overrides (status, step_key, trait_key, created_at DESC);

-- =============================================================================
-- TABLE: override_history (Audit Trail)
-- =============================================================================
CREATE TABLE IF NOT EXISTS override_history (
    id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    override_id VARCHAR(20) NOT NULL REFERENCES overrides(id) ON DELETE CASCADE,
    changed_at TIMESTAMPTZ DEFAULT NOW(),
    changed_by VARCHAR(100) NOT NULL,
    change_type VARCHAR(20) NOT NULL CHECK (change_type IN ('create', 'update', 'status_change', 'delete')),
    before_state JSONB,
    after_state JSONB,
    summary TEXT
);

CREATE INDEX IF NOT EXISTS idx_override_history_lookup 
ON override_history (override_id, changed_at DESC);

-- =============================================================================
-- TRIGGERS
-- =============================================================================
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ language 'plpgsql';

CREATE TRIGGER trg_steps_updated_at BEFORE UPDATE ON steps FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_defaults_updated_at BEFORE UPDATE ON defaults FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
CREATE TRIGGER trg_overrides_updated_at BEFORE UPDATE ON overrides FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

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
        summary := format('Created override %s for %s.%s', NEW.id, NEW.step_key, NEW.trait_key);
    ELSIF TG_OP = 'UPDATE' THEN
        IF OLD.status IS DISTINCT FROM NEW.status THEN
            change_type := 'status_change';
            summary := format('Status: %s → %s for override %s', OLD.status, NEW.status, NEW.id);
        ELSE
            change_type := 'update';
            summary := format('Updated override %s', NEW.id);
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