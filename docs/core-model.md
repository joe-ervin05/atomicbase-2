# Atombase Core Model

Atombase uses one product vocabulary across docs, CLI commands, SDK surfaces, API routes, and storage:

```text
Project -> Definition -> Database -> Session/Auth Context -> Query
```

## Project

A project is the running Atombase backend plus its primary metadata database, configuration, service key, and application databases.

Project-scoped surfaces include:

- service configuration and environment variables
- CLI configuration in `atombase.config.ts`
- platform routes for project-owned definitions and databases
- SDK clients created with `createClient({ url, apiKey })` or `createClient({ url, sessionToken })`

## Definition

A definition is the versioned contract for a database. It contains:

- schema
- access policies
- provisioning rules
- organization membership and management rules when the definition uses organization scope

Definition scope is a property of the definition, not a separate top-level abstraction:

- `global`: project-managed databases
- `user`: one self-provisioned database per authenticated user
- `organization`: one database per organization, with database-local membership

Definitions are the canonical abstraction.

## Database

A database is a concrete SQLite or Turso database created from a definition.

Every database records:

- its definition
- the definition version it is currently running
- routing metadata used by the API and SDK

Database routes and SDK methods should describe concrete database instances.

## Session/Auth Context

Session/auth context describes the caller for a request.

It includes:

- anonymous, service, or session-backed caller mode
- authenticated user identity when present
- organization role when the target database uses organization scope

Organizations, users, sessions, and service keys belong to this layer. They exist to resolve auth context for a database and query.

## Query

A query is the data operation against one resolved database under one resolved auth context.

Queries flow through:

- database routing
- definition version checks and lazy migration
- access policy compilation
- SQL execution

The query surface is `client.database(...).from(table)` in the SDK and `/data/query/{table}` in the HTTP API.
