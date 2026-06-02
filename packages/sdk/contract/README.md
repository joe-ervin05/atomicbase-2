# SDK Contract Tests

These tests validate real API/SDK behavior together (not mocks).

## What is covered

- Definition creation through the SDK
- Database creation through the SDK
- CRUD flows through `client.database("<database-id>").from(...)`
- Count behavior (`count`, `withCount`)
- Batch atomicity (rollback when one operation fails)
- Error code contracts (`MISSING_WHERE_CLAUSE`)

## Requirements

1. Atombase API is running.
2. API is configured for database creation (Turso environment on the API side).
3. If platform auth is enabled, set `ATOMBASE_API_KEY`.

## Run

From repo root:

```bash
ATOMBASE_CONTRACT=1 pnpm test:contract:sdk
```

Optional base URL override:

```bash
ATOMBASE_CONTRACT=1 ATOMBASE_CONTRACT_BASE_URL=http://localhost:8080 pnpm test:contract:sdk
```

If `ATOMBASE_CONTRACT` is not set to `1`, tests are skipped intentionally.

## Notes

- These tests currently exercise the service-key platform path and direct database-id routing.
- Session-backed auth flows and `client.database()` user-self routing are not covered by this contract suite yet.
