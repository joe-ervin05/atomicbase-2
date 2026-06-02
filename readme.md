# Atombase

**Launch your SaaS without rebuilding the backend.**

Atombase is the SaaS-native backend for a definition-driven database architecture, built on distributed SQLite.

> [!CAUTION]
> **Prototype software - not complete.**
> Atombase is still under active development and is not production-ready. Expect missing features, API changes, and rough edges.

## What is Atombase?

Atombase helps you run many isolated databases from versioned definitions while still feeling like you are working with a single backend.

- **Projects**: run one backend with one primary metadata database.
- **Definitions**: version database schemas, access policies, and provisioning rules in code.
- **Databases**: create isolated SQLite/Turso databases from definitions.
- **Session/Auth Context**: resolve service, anonymous, user, and organization access.
- **Queries**: run policy-aware data operations over HTTP or the SDK.
- **Managed Migrations**: keep database schemas in sync through definition versions.
- **Authentication**: built-in magic-link auth and session-backed browser access.
- **Storage**: coming soon.
- **AI**: coming soon.

## Prototype Status

| Component        | Status       |
| ---------------- | ------------ |
| Data API         | Alpha        |
| TypeScript SDK   | Alpha        |
| Platform API     | Experimental |
| Definitions      | Experimental |
| CLI              | Alpha        |
| Authentication   | Alpha        |
| AI               | In progress  |
| File Storage     | Planned      |
| Realtime         | Planned      |
| Dashboard        | Planned      |

This repository currently represents a working prototype, not a finished product.

## Quick Start

```bash
cd api
```

### 1) Set environment variables

```ini
TURSO_API_KEY="your-turso-key"
TURSO_ORGANIZATION="your-turso-org"

ATOMBASE_CORS_ORIGINS="http://localhost:3000,http://localhost:5173"
ATOMBASE_API_KEY="your-api-key"
```

### 2) Start the API

```bash
make run
```

By default the server runs at `http://localhost:8080`.

### 3) Install the SDK and definitions package

```bash
npm install @atombase/sdk @atombase/definitions
```

### 4) Initialize project config

```bash
npx atombase init
```

### 5) Define and push a definition

```typescript
import { defineGlobal, defineSchema, defineAccess, defineTable, c, allow } from "@atombase/definitions";

const schema = defineSchema({
  users: defineTable({
    id: c.integer().primaryKey(),
    name: c.text().notNull(),
    email: c.text().notNull().unique(),
  }),
});

export default defineGlobal({
  schema,
  access: defineAccess(schema, {
    users: {
      select: allow(),
      insert: allow(),
      update: allow(),
      delete: allow(),
    },
  }),
});
```

```bash
npx atombase definitions push
```

### 6) Create a database

```typescript
import { createClient } from "@atombase/sdk";

const client = createClient({
  url: "http://localhost:8080",
  apiKey: "your-api-key",
});

await client.databases.create({ id: "acme-corp", definition: "my-app" });
```

### 7) Query data

```typescript
import { eq } from "@atombase/sdk";

const acme = client.database("acme-corp");

await acme.from("users").insert({ name: "Alice", email: "alice@example.com" });
const { data } = await acme.from("users").select();
await acme.from("users").update({ name: "Alicia" }).where(eq("id", 1));
await acme.from("users").delete().where(eq("id", 1));
```

### 8) Browser auth and user databases

For browser apps, the intended path is:

1. start magic-link auth
2. user clicks the emailed link to your app callback route
3. app callback reads the `token` query param and calls `completeMagicLink`
4. store the returned session token client-side
5. restore that token on app boot
6. create a session-backed SDK client
7. call `client.auth.me()`
8. if `databaseId` is missing, self-provision with `client.auth.createDatabase({ definition })`
9. use `client.database()` with no `Database` header to access the current user's database directly from the browser

```typescript
import { createClient } from "@atombase/sdk";

const baseClient = createClient({ url: "http://localhost:8080" });
const tokenFromEmail = new URL(window.location.href).searchParams.get("token");
if (!tokenFromEmail) throw new Error("Missing magic-link token");

const completed = await baseClient.auth.completeMagicLink(tokenFromEmail);
if (completed.error) throw completed.error;

localStorage.setItem("atombase.session", completed.data.token);

const client = baseClient.withSession(completed.data.token);
const me = await client.auth.me();
if (me.error) throw me.error;

if (!me.data.databaseId) {
  const provisioned = await client.auth.createDatabase({ definition: "todo-app" });
  if (provisioned.error) throw provisioned.error;
}

const todoDb = client.database();
```

## Core Model

Atombase uses one vocabulary everywhere:

```text
Project -> Definition -> Database -> Session/Auth Context -> Query
```

See [docs/core-model.md](./docs/core-model.md) for the canonical noun model. Historical `global`, `user`, and `organization` terms are definition scopes, not separate top-level abstractions.

## Key Ideas

- **Database isolation by default**: each concrete app, user, or organization boundary gets its own database.
- **Definitions keep systems aligned**: define once, roll databases forward with migrations.
- **Strict versions + lazy sync**: out-of-date databases are synchronized when accessed.
- **Simple operational model**: single Go service with a focused API surface.

## Roadmap (incomplete)

- Enterprise authentication (orgs, RBAC, SSO, RLS)
- Storage APIs and object workflows
- AI APIs with database-scoped context
- Dashboard and improved operator tooling

## Examples

- [react-todo](./examples/react-todo) - Next.js todo example using magic-link auth, organization databases, and direct SDK data access

## Contributing

See [CONTRIBUTING.md](./CONTRIBUTING.md).

## License

Atombase is [fair-source](https://fair.io) licensed under [FSL-1.1-MIT](./LICENSE). You can use, modify, and self-host the software freely for your own applications. The only restriction is offering Atombase as a competing hosted service. The license converts to MIT after two years.
