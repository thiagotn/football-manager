# PRD 047: Refactor Go API v2 Handlers to Hybrid Store Strategy

**Status:** Phase 1 ✅ Complete | Phases 2-4 Pending  
**Last Updated:** 2026-05-27  
**Target Completion:** Progressive (5 handlers per phase)

---

## Overview

Refactor all Go API v2 handlers from tightly-coupled `*pgxpool.Pool` dependencies to a **hybrid Store interface strategy**, enabling proper unit testing without a real database.

The hybrid strategy defines a narrow `XxxStore` interface for each handler containing only the DB methods it needs, backed by a `pgXxxStore` adapter in production and mock implementations in tests.

---

## Why This Change?

### Current State
- All handlers except `auth` directly hold `*pgxpool.Pool`
- Unit tests cannot test handler business logic without a real database
- No way to mock DB behavior (errors, edge cases) in isolated tests
- Current test coverage: **20.4%** across internal packages

### Target State
- Each handler has a scoped `XxxStore` interface
- Mock stores injected in unit tests
- Testable validation gates (422 errors, 403 permission checks, 404 handling)
- Improved test coverage for business logic paths
- Constructor signature unchanged for backward compatibility (still takes `*pgxpool.Pool`)

### Benefits
1. **Isolated Unit Testing** — validate input validation, permission checks, error handling without a DB
2. **Edge Case Coverage** — mock stores can simulate DB failures, empty results, conflicts
3. **Faster Tests** — unit tests run in milliseconds, no DB setup/teardown
4. **Cleaner Code** — explicit dependencies (Store interface) vs implicit (pool field)
5. **Test-Driven Fixes** — easier to write failing test first, then fix

---

## Phase 1: Simple Handlers (5/5 Complete ✅)

**Status:** All 5 handlers refactored and tested  
**Test Coverage Improvement:** Validation paths now unit-testable

### beta
- **Store Methods:** `InsertAndroidBetaSignup`
- **Tests Added:** `TestBeta_AndroidSignup_InvalidEmail`, `TestBeta_AndroidSignup_MissingEmail`, `TestBeta_AndroidSignup_MalformedJSON`
- **Validation Paths Tested:** Email format validation, empty email, malformed JSON
- **Coverage Impact:** Email regex and validation now testable without DB

### push  
- **Store Methods:** `UpsertPushSubscription`, `DeletePushSubscriptions`
- **Tests Scenarios:** `TestSubscribe_MissingKeys`, `TestUnsubscribe_NoAuth`
- **Validation Paths Tested:** Required fields check (endpoint, keys), auth requirement
- **Coverage Impact:** Request validation logic isolated and mockable

### ranking
- **Store Methods:** `GetTopRanking`, `GetFlopRanking`
- **Tests Scenarios:** `TestGetRanking_InvalidType`, `TestGetRanking_InvalidYear`, `TestGetRanking_MonthWithoutYear`
- **Validation Paths Tested:** Type enum validation, year range, month dependency
- **Coverage Impact:** Query parameter validation covered without DB calls

### reviews
- **Store Methods:** `GetMyReview`, `UpsertReview`, `GetReviewSummary`, `ListReviews`
- **Tests Scenarios:** `TestReviews_Admin_Forbidden`, `TestReviews_Upsert_InvalidRating`, `TestListReviews_InvalidRatingFilter`
- **Validation Paths Tested:** Admin role check, rating bounds (1-5), comment length (≤500 chars), filter validation
- **Coverage Impact:** All validation gates now unit-testable

### mcp_tokens
- **Store Methods:** `GenerateMCPToken`, `CreateMCPToken`, `ListMCPTokens`, `GetMCPToken`, `RevokeMCPToken`
- **Tests Scenarios:** `TestMCPToken_Create_MissingName`, `TestMCPToken_Create_InvalidExpiresIn`, `TestMCPToken_Revoke_InvalidUUID`
- **Validation Paths Tested:** Name required, expires_in enum ('h24'|'d7'|null), UUID parsing
- **Coverage Impact:** Token creation validation fully testable

---

## Test Coverage Summary (Phase 1)

| Handler | Type | Methods | Validation Tests | Status |
|---------|------|---------|------------------|--------|
| beta | Simple | 1 | 3 | ✅ Done |
| push | Simple | 2 | 3 | ✅ Done |
| ranking | Simple | 2 | 4 | ✅ Done |
| reviews | Simple | 4 | 4 | ✅ Done |
| mcp_tokens | Simple | 5 | 3 | ✅ Done |
| **Phase 1 Total** | | **14** | **17** | **✅ 100%** |

### What's Tested in Phase 1

✅ **Input Validation Paths**
- Empty/missing required fields → 422 Unprocessable
- Invalid formats (non-UUID, invalid email, out-of-range numbers) → 422
- Business rule violations (role checks, bounds) → 403/422

✅ **Pre-DB Gates**
- Auth requirement checks → 401 Unauthorized
- Admin role enforcement → 403 Forbidden
- Field format validation (regex, enum, range)

❌ **Not Yet Tested in Phase 1**
- DB error scenarios (connection failure, constraint violation)
- Database query behavior (empty result sets, data consistency)
- Concurrent write handling (conflicts, race conditions)
- Integration with other services (stripe, storage, LLM)

---

## Implementation Pattern

All Phase 1 handlers follow this pattern:

```go
// 1. Define Store interface (only methods this handler uses)
type XxxStore interface {
    Method1(ctx context.Context, ...) error
    Method2(ctx context.Context, ...) (*Type, error)
}

// 2. Create pgXxx adapter (wraps real pool)
type pgXxxStore struct {
    pool *pgxpool.Pool
}
func (s *pgXxxStore) Method1(ctx context.Context, ...) error {
    return db.Method1(ctx, s.pool, ...)
}

// 3. Export handler struct with Store field
type XxxHandler struct {
    Store XxxStore
}

// 4. Constructor unchanged for callers
func NewXxxHandler(pool *pgxpool.Pool) *XxxHandler {
    return &XxxHandler{Store: &pgXxxStore{pool: pool}}
}

// 5. Unit tests inject mock stores
type mockXxxStore struct {
    method1Fn func(...) error
}
func (m *mockXxxStore) Method1(...) error {
    return m.method1Fn(...)
}
```

---

## Phases 2-4 (Planned)

### Phase 2: Medium Handlers (5 handlers)
- **finance** (10 store methods)
- **invites** (12 store methods)
- **votes** (13 store methods)
- **webhooks** (4 store methods + stripe interface)
- **subscriptions** (3 store methods + stripe interface)

**Estimated Impact:** 40+ additional unit tests, validation of group limits, permission gates, vote workflows

### Phase 3: Large Handlers (2 handlers)
- **matches** (16 store methods) — largest handler
- **groups** (25 store methods) — most complex business logic

**Estimated Impact:** 60+ unit tests covering match lifecycle, group member management, plan constraints

### Phase 4: Raw SQL Extraction (3 handlers)
- **players** (need SQL wrappers for stats queries)
- **chat** (need SQL wrappers for user listing)
- **admin** (need SQL wrappers for stats, metrics)

**Estimated Impact:** 30+ new db.* functions, 50+ tests covering aggregations, analytics, reporting

---

## Acceptance Criteria (Phase 1) ✅

- [x] All 5 handlers refactored with Store interfaces
- [x] `pgXxxStore` adapters created and wired in constructors
- [x] Constructor signatures unchanged (backward compatible)
- [x] Unit tests written for validation paths (pre-DB gates)
- [x] All validation tests passing
- [x] Mock stores pattern established for reuse in Phases 2-4
- [x] `postJSON` helper added to test infrastructure
- [x] `phase3_test.go` updated with mock stores

---

## Files Modified

### Handlers (refactored)
- `internal/handlers/beta.go` — added BetaStore interface + pgBetaStore adapter
- `internal/handlers/push.go` — added PushStore interface + pgPushStore adapter
- `internal/handlers/ranking.go` — added RankingStore interface + pgRankingStore adapter
- `internal/handlers/reviews.go` — added ReviewStore interface + pgReviewStore adapter
- `internal/handlers/mcp_tokens.go` — added MCPTokenStore interface + pgMCPTokenStore adapter

### Tests (updated)
- `tests/unit/phase3_test.go` — added mock stores, replaced nil pools with mocks
- `tests/unit/helpers_test.go` — added `postJSON` helper for test reuse
- `tests/unit/auth_test.go` — removed duplicate `postJSON`, streamlined imports

### Unchanged
- `internal/server/router.go` — no changes (constructors still take `*pgxpool.Pool`)
- All other handlers — pending Phases 2-4

---

## Coverage Metrics

### Pre-Phase 1
```
github.com/thiagotn/football-manager/football-api-go/internal/handlers    0.0%
github.com/thiagotn/football-manager/football-api-go/internal/middleware   ~90%
github.com/thiagotn/football-manager/football-api-go/internal/services    0-100% (varies)
Total: 20.4%
```

### Post-Phase 1
✅ **Validation paths now testable** in 5 handlers (beta, push, ranking, reviews, mcp_tokens)

**Note:** Coverage % may not show large increase because:
1. Handler code with mock stores still exercises validation logic (good)
2. DB-touching code (db.* calls) not executed in unit tests (expected)
3. Full coverage increase comes in Phases 2-4 when more complex logic is tested

**Measurable Improvement:** 17 new unit tests covering pre-DB gates, validation logic, permission checks — all critical for reliability.

---

## Next Actions

### Immediate (Phase 2)
1. Refactor finance handler (GetFinancePeriod, UpdatePayment logic)
2. Refactor invites handler (AddMember, AcceptInvite complex flows)
3. Refactor votes handler (voting permission, result calculation)
4. Extract stripe.VerifyWebhookSignature to StripeVerifier interface
5. Implement Phase 2 tests

### Medium Term (Phase 3)
1. Refactor matches handler (16 methods, match lifecycle)
2. Refactor groups handler (25 methods, group operations)
3. Heavy test coverage for business rule enforcement

### Long Term (Phase 4)
1. Wrap raw `pool.Query` calls in `internal/db/` layer
2. Refactor players, chat, admin with wrapped queries
3. Full end-to-end test coverage

---

## Rollback Plan

If any phase encounters issues:
1. All constructors remain `NewXxxHandler(pool *pgxpool.Pool)`
2. Caller code (router.go) requires **zero changes**
3. Can revert individual handler by removing Store interface and reverting to direct pool usage
4. Tests can be disabled without affecting production code

---

## Success Metrics

- [ ] Phase 1: 100% of 5 handlers refactored + 17 validation tests passing ✅
- [ ] Phase 2: 100% of 5 handlers refactored + 40+ tests
- [ ] Phase 3: 100% of 2 handlers refactored + 60+ tests
- [ ] Phase 4: 100% of 3 handlers refactored + 50+ tests
- [ ] **Final:** All 16 handlers using Store pattern, 167+ unit tests, internal handler coverage >40%

---

## Notes

- **Backward Compatibility:** Fully maintained — no changes to API routes, constructors, or public interfaces
- **Incremental Adoption:** Can deploy Phase 1, then Phase 2, etc. without waiting for all phases
- **Test-Driven Culture:** Mock stores make it natural to write validation tests before implementation
- **Future Maintainability:** New handlers added after Phase 4 can follow same Store pattern from day one

