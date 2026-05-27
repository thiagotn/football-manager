# API Comparison Skill: `/api-compare`

## Overview

The `/api-compare` skill compares Go API v2 (`football-api-go`) endpoint implementations against their Python API v1 (`football-api`) equivalents to identify missing business rules, validation gaps, and behavioral differences.

## Installation

The skill is located at `.claude/skills/api-compare/SKILL.md` in your local Claude Code environment. If not present, it will be created automatically by the setup process.

To manually create it:

```bash
mkdir -p .claude/skills/api-compare
cp docs/API_COMPARISON_SKILL.md/.claude/skills/api-compare/SKILL.md .claude/skills/api-compare/
```

Or copy the content from the SKILL.md section below into `.claude/skills/api-compare/SKILL.md`.

## Usage

### Basic Syntax

```bash
/api-compare /matches/{matchID}/teams
/api-compare GET /groups/{groupID}/members  
/api-compare POST /players/me
/api-compare curl 'http://localhost:8080/api/v2/matches/123/teams' -H "..."
```

### Examples

**Compare teams endpoint:**
```
/api-compare /matches/{matchID}/teams
```
Produces: detailed analysis of GET and POST team endpoints, identifying 4+ gaps between v1 and v2.

**Compare with method specified:**
```
/api-compare POST /invites
```

**From curl (auto-extracts path):**
```
/api-compare curl 'http://localhost:8080/api/v2/groups/abc123/members' -H 'Authorization: Bearer ...'
```

## Output Format

The skill generates a markdown report with:

1. **Source Files** — exact locations in both APIs
2. **✅ Parities Confirmed** — matching behavior
3. **⚠️ Gaps Identified** — table of differences
4. **📋 Detailed Explanations** — code snippets and fix suggestions
5. **Priority Classification** — P0 (breaking), P1 (behavioral), P2 (UX)

Example:

```
## Comparison: POST /matches/{matchID}/teams

### ⚠️ Gaps Identified

| # | Category | Python v1 | Go v2 | Impact |
|---|----------|-----------|-------|--------|
| 1 | Validation | Pre-checks minimum player count with descriptive error | No pre-check, fails in service with generic error | **High** |
| 2 | Join Type | INNER JOIN (excludes non-members) | LEFT JOIN (includes with defaults) | **Medium** |
| 3 | Response | Global + group nickname coalesced in draw | Only global nickname in response | **Medium** |
| 4 | 404 Handling | Returns 404 if match not found | Returns 200 with empty arrays | **Medium** |
```

## Known Gaps (from `/api-compare /matches/{matchID}/teams`)

1. **Minimum Player Validation** — v1 pre-checks before draw; v2 only fails in service with generic error
2. **GET Match Existence Check** — v1 returns 404; v2 returns 200 with empty arrays
3. **Confirmed Players JOIN** — v1 uses INNER JOIN (excludes non-group-members); v2 uses LEFT JOIN (includes with defaults)
4. **Nickname Coalesce** — v1 returns group nickname in POST response; v2 returns global nickname only

## How It Works

The skill follows a 6-step process:

1. **Parse input** — extract method, path, domain
2. **Locate files** — find handlers, repos, services in both APIs
3. **Read source** — load all relevant implementation code
4. **Analyze 6 dimensions** — auth, validation, logic, queries, response, errors
5. **Generate report** — structured markdown with gaps and fixes
6. **Summarize fixes** — actionable recommendations with priority

## Technical Details

### Supported Dimensions

- **Authentication & Authorization** — role checks, group admin, ownership
- **Input Validation** — required fields, ranges, business rules
- **Business Logic** — flow, calculations, conditionals
- **Database Queries** — JOINs, filters, coalesces, fetch strategy
- **Response Structure** — fields, status codes, field naming
- **Error Handling** — 404, 403, 409, 422, descriptive messages

### File Mappings

| Layer | Go v2 | Python v1 |
|-------|-------|-----------|
| Handler | `internal/handlers/{domain}.go` | `app/api/v1/routers/{domain}.py` |
| Database | `internal/db/{domain}.go` | `app/db/repositories/{domain}_repo.py` |
| Service | `internal/services/{domain}*.go` | `app/services/{domain}*.py` |
| Schema | — | `app/schemas/{domain}.py` |

## Skill File Location

```
.claude/skills/api-compare/SKILL.md
```

This file is not committed to git (it's in `.gitignore`). If you clone the repo fresh, you'll need to create it using the instructions above.

## Contributing

To improve the skill:
1. Edit `.claude/skills/api-compare/SKILL.md`
2. Test with `/api-compare [endpoint]`
3. Document findings in this file
4. Share new patterns or gaps discovered

## See Also

- `football-api/CLAUDE.md` — v1 API documentation
- `football-api-go/` — v2 source code (Go)
