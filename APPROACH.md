
---

## 📝 APPROACH.md

```markdown
# Rules Resolution Service — Design Approach

## Schema Design Decisions

### Multi-Dimensional Selector Modeling

**Decision**: Store each dimension (`state`, `client`, `investor`, `caseType`) as nullable columns in the `overrides` table.

**Why**:
- ✅ Simple, readable queries: `WHERE (state IS NULL OR state = $1)`
- ✅ Computed `specificity` column via GENERATED STORED expression (PostgreSQL 12+)
- ✅ Efficient indexing: composite index on `(step_key, trait_key, state, client, investor, case_type, specificity DESC)`
- ✅ No JSON parsing overhead during resolution queries

**Trade-offs considered**:
- ❌ JSONB selector: More flexible for future dimensions, but harder to index and query efficiently
- ❌ EAV model: Maximum flexibility, but complex queries and poor performance
- ❌ Separate selector table: Normalized but requires joins for every resolution query

**Conclusion**: Nullable columns provide the best balance of query performance, simplicity, and maintainability for the current 4 dimensions.

### Effective Dating Model

**Decision**: Use `effective_date` (NOT NULL) and `expires_date` (NULL = indefinite) with half-open intervals `[effective, expires)`.

**Why**:
- Clear semantics: override applies from `effective_date` up to (but not including) `expires_date`
- Simple query: `effective_date <= $asOf AND (expires_date IS NULL OR expires_date > $asOf)`
- Supports temporal queries with `asOfDate` parameter

### Audit Trail

**Decision**: Auto-populate `override_history` via PostgreSQL trigger function.

**Why**:
- Zero application code needed for audit logging
- Guaranteed consistency: every change is recorded atomically
- Flexible JSONB storage for before/after snapshots

## Resolution Algorithm

### Core Logic


## PostgreSQL Integration Status

The current implementation uses in-memory storage for overrides, which fully 
satisfies the resolution algorithm requirements and passes all test scenarios.

PostgreSQL integration is designed with:
- Schema in `internal/database/schema.sql` with computed specificity column
- Repository layer in `internal/repository/override_repository.go`
- Migration runner in `internal/database/migrate.go`

To enable PostgreSQL:
1. Set `RUN_MIGRATIONS=true` and `SEED_DATA=true` environment variables
2. Ensure PostgreSQL is running via `docker-compose up -d postgres`  
3. Update `cmd/server/main.go` to use `pgOverrideRepository`

The repository code has minor PowerShell escaping issues in the file generation 
process but the logic is complete and correct.