#!/bin/bash
# Setup script to initialize Claude Code skills for this project
# Run: bash scripts/setup-skills.sh

set -e

PROJECT_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
SKILLS_DIR="$PROJECT_ROOT/.claude/skills"
API_COMPARE_SKILL="$SKILLS_DIR/api-compare/SKILL.md"

echo "Setting up Claude Code skills..."
echo "Project root: $PROJECT_ROOT"

# Create api-compare skill
if [ ! -d "$SKILLS_DIR/api-compare" ]; then
  echo "📁 Creating api-compare skill directory..."
  mkdir -p "$SKILLS_DIR/api-compare"
else
  echo "✓ api-compare directory exists"
fi

if [ ! -f "$API_COMPARE_SKILL" ]; then
  echo "📝 Creating api-compare SKILL.md..."
  cat > "$API_COMPARE_SKILL" << 'EOF'
---
name: api-compare
description: Compares a Go API v2 endpoint implementation against the Python API v1 equivalent, identifying missing business rules, validation gaps, and behavioral differences.
allowed-tools: Read, Bash
---

# API Comparison Skill: v1 (Python) vs v2 (Go)

## Purpose
Analyze an API endpoint implementation in both FastAPI (v1) and Go (v2) to identify gaps in business rules, validation, error handling, and behavior.

## Input Format
The skill accepts `$ARGUMENTS` in any of these formats:
- **Path only**: `/matches/{matchID}/teams`
- **With method**: `GET /groups/{groupID}/members` or `POST /players/me`
- **Full curl**: `curl 'http://localhost:8080/api/v2/matches/123/teams' -H "Auth: Bearer ..."` (path extracted automatically)

## Execution Steps

### Step 1: Parse Input and Identify Domain

Extract:
- **Method** (GET, POST, PATCH, DELETE) — default GET if not specified
- **Path** — extract `/api/v2/...` and remove UUIDs/IDs to find the pattern
- **Domain** — first resource in path (matches, groups, players, teams, etc.)

### Step 2: Locate Source Files in Both APIs

**Go v2** — search in `football-api-go/`:
- Handler: `internal/handlers/{domain}.go` — look for function matching the HTTP method
- Database queries: `internal/db/{domain}.go` or `internal/db/queries.go`
- Services: `internal/services/{domain}*.go`
- Models: `internal/models/` or inline in handlers

**Python v1** — search in `football-api/`:
- Router: `app/api/v1/routers/{domain}.py`
- Repository: `app/db/repositories/{domain}_repo.py`
- Service: `app/services/{domain}*.py` (if exists)
- Schema: `app/schemas/{domain}.py` (request/response types)
- Model: `app/models/{domain}.py`

If files are not found by pattern, use `grep -r` to search for the endpoint path string (e.g., `grep -r "matches.*teams" football-api-go/`).

### Step 3: Read All Relevant Source Code

For the identified endpoint, read:
1. Complete handler function (both versions)
2. All called repository/service functions
3. Database query logic (raw SQL or ORM)
4. Schema/type definitions (request, response)
5. Error handling and status code assignments

### Step 4: Comparative Analysis Across 6 Dimensions

Analyze both implementations and document differences in:

#### 1. **Authentication & Authorization**
- Is auth required? Which middleware applies?
- Role checks (admin, group admin, owner)?
- Any differences in who can access the endpoint?

#### 2. **Input Validation**
- Required fields?
- Data type validation?
- Range/format validation?
- Business rule validation (e.g., "minimum 10 players to draw teams")?

#### 3. **Business Logic**
- Main flow and decision points
- Calculations or transformations
- Sequence of operations
- Any conditional logic?

#### 4. **Database Queries**
- SQL/ORM structure
- JOIN types (INNER vs LEFT) — affects which records are included
- COALESCE logic (fallback field values)
- Filtering, grouping, ordering
- Does it fetch fresh data after write, or return in-memory?

#### 5. **Response Structure**
- Which fields are returned?
- HTTP status codes (200, 201, 400, 403, 404, 409)?
- Any inconsistencies in field naming or structure?

#### 6. **Error Handling**
- Does it validate resource exists before returning 404?
- Descriptive error messages vs generic ones?
- Which error scenarios are explicitly caught?

### Step 5: Generate Report

Output markdown with this structure:

```
## Comparison: [METHOD] [PATH]

### Source Files
**Go v2:**
- `football-api-go/internal/handlers/{domain}.go` — function name
- `football-api-go/internal/db/{domain}.go` — function name
- ...

**Python v1:**
- `football-api/app/api/v1/routers/{domain}.py` — function name
- `football-api/app/db/repositories/{domain}_repo.py` — function name
- ...

### ✅ Parities Confirmed
- Authentication: Both check [X]
- Error 404: Both return 404 if resource not found
- ...

### ⚠️ Gaps Identified

| # | Category | Python v1 | Go v2 | Impact |
|---|----------|-----------|-------|--------|
| 1 | Validation | Validates min 10 players before draw | No pre-check, fails in service layer with generic error | **High** — v2 gives poor error message |
| 2 | Auth | ... | ... | ... |
| 3 | Join Type | INNER JOIN on group_members (excludes non-members) | LEFT JOIN (includes with defaults) | **Medium** — v2 has different behavior for guest players |

### 📋 Detailed Explanations

#### Gap #1: Minimum Player Validation
**Python v1** (`app/api/v1/routers/teams.py`, lines X–Y):
```python
min_needed = (match.players_per_team + 1) * 2
if len(confirmed) < min_needed:
    raise ValidationError(f"Need at least {min_needed} players...")
```

**Go v2** (`football-api-go/internal/handlers/teams.go`, lines X–Y):
```go
// No pre-check; service only fails if nTeams < 2
teams, err := services.BuildTeams(...)
```

**Fix suggestion:** Add validation in Go handler before calling service, with descriptive error message.

---
```

### Step 6: Summarize Actionable Fixes

If gaps are found, list each one with:
- Where to fix (file and line range)
- What to add/change
- Code snippet example (if straightforward)
- Priority (P0 = breaking bug, P1 = behavioral difference, P2 = minor UX)

## Notes

- **Line numbers** — use Read tool to locate exact lines and include them in the report
- **Code snippets** — show 3-5 lines of context for each gap
- **If endpoint not found** — report that the files don't exist and suggest manual comparison
- **Field name mismatches** — always note if response fields differ between v1 and v2
- **Ignore trivial differences** — language idioms (Go vs Python) are not gaps; focus on business logic and behavior
EOF
  echo "✓ api-compare SKILL.md created"
else
  echo "✓ api-compare SKILL.md already exists"
fi

echo ""
echo "✅ Skills setup complete!"
echo ""
echo "Available skills:"
echo "  - /api-compare [endpoint]  — Compare Go v2 vs Python v1 implementations"
echo ""
echo "Documentation: docs/API_COMPARISON_SKILL.md"
