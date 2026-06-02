# full-test

Deterministic simulation testing for the Atombase Data API.

This runner provisions a fresh, complex definition using `@atombase/definitions`, pushes it via the Atombase CLI, creates a dedicated test database, and then runs a seeded stateful simulation against that database.

## Why this exists

- Reproducible failures via fixed seeds.
- Stateful API validation (not just one-off requests).
- Easy long-running stress mode (`-loop`) for constant testing.

## What gets provisioned

Before simulation starts, `full-test` creates a temporary workspace and uses the workspace packages directly:

- invokes `packages/cli` through `pnpm --filter @atombase/cli exec atombase ...`
- generates schema files that import `@atombase/definitions`

Then it runs:

- `atombase definitions push <generated-definition-name>`
- `atombase databases create <generated-database-name> --definition <generated-definition-name>`

The generated definition is intentionally complex and includes:

- Multiple related tables (`users`, `workspaces`, `projects`, `tags`, `project_tags`, `todos`, `comments`, `attachments`, `audit_events`)
- Composite primary keys
- Foreign keys with mixed actions (`CASCADE`, `SET NULL`, `RESTRICT`)
- Generated columns
- `CHECK`, `UNIQUE`, and collations
- Secondary indexes and FTS tables

Simulation targets the `todos` table in that provisioned database.

## Run

```bash
cd full-test
go run . -api-key "$ATOMBASE_API_KEY" -token "$ATOMBASE_API_KEY"
```

## Common options

```bash
go run . -api-key "$ATOMBASE_API_KEY" -token "$ATOMBASE_API_KEY" -seed 123 -steps 2000
go run . -loop
go run . -base-url http://localhost:8080 -repo-root /path/to/atombase
go run . -keep-resources
go run . -provision=false -database existing-db -table todos
go run . -fail-on-4xx
```

## Env vars

- `ATOMBASE_BASE_URL`
- `ATOMBASE_API_KEY`
- `ATOMBASE_DATABASE`
- `ATOMBASE_TABLE`
- `ATOMBASE_TOKEN`
- `ATOMBASE_ID_COLUMN`
- `ATOMBASE_TITLE_COLUMN`
- `ATOMBASE_COMPLETED_COLUMN`
- `SIM_REPO_ROOT`
- `SIM_SEED`
- `SIM_STEPS`
- `SIM_LOOP`
- `SIM_PROVISION`
- `SIM_KEEP_RESOURCES`
- `SIM_TIMEOUT_MS`
- `SIM_FAIL_ON_4XX`

Flags always override env vars.

## Failure behavior

On mismatch or unexpected HTTP failure, the runner exits with a replay command including the exact seed.

By default, the created database and temporary workspace are deleted after the run. Use `-keep-resources` to retain them for debugging.
