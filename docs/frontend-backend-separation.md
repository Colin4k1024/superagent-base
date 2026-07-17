# Frontend/Backend Repository Separation Plan

## Current State

- Frontend (`web/`) and backend (`backend/`) live in the same repository
- `web/node_modules/` has 20,000+ files committed to git
- Frontend and backend are independently buildable (separate Dockerfiles)
- API contract is implicit (HTTP handlers in `backend/api/handler/`)

## Problems

1. **Repository bloat**: `node_modules/` inflates clone size by ~200MB
2. **CI coupling**: Frontend changes trigger backend CI and vice versa
3. **Developer experience**: Frontend devs must clone the entire Go backend
4. **Release coupling**: Frontend and backend must be released together

## Separation Strategy

### Phase 1: API Contract Extraction (Low Risk)

1. Generate OpenAPI spec from backend HTTP handlers
2. Create `api-spec/` directory with:
   - `openapi.yaml` — machine-readable API contract
   - `CHANGELOG.md` — API breaking change log
3. Add CI check: frontend API client must match spec

### Phase 2: Frontend Extraction (Medium Risk)

1. Create `superagent-frontend` repository
2. Move `web/` contents (excluding `node_modules/`)
3. Add `web/package-lock.json` to the new repo
4. Update Docker build to pull from separate repo
5. Add API client generation from OpenAPI spec

### Phase 3: Monorepo Cleanup (Low Risk)

1. Remove `web/` from main repository
2. Remove `node_modules/` from git history (BFG filter)
3. Update CI pipelines
4. Update documentation

## Recommended Approach

For now, the simplest improvement is to:

1. Add `web/node_modules/` to `.gitignore` (if not already)
2. Remove `node_modules/` from git tracking: `git rm -r --cached web/node_modules/`
3. Add a `web/.env.example` with API base URL configuration
4. Document the API contract in `docs/api/`

This gives 80% of the benefit with minimal risk.
