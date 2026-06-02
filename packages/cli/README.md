# @atombase/cli

Command-line interface for Atombase project, definition, and database management.

## Installation

```bash
npm install -D @atombase/cli
# or
pnpm add -D @atombase/cli
```

## Configuration

Create `.env` or `atombase.config.ts` in your project root:

```bash
# .env
ATOMBASE_URL=http://localhost:8080
ATOMBASE_API_KEY=your-api-key
```

Or use a config file:

```typescript
// atombase.config.ts
import { defineConfig } from "@atombase/cli";

export default defineConfig({
  url: "http://localhost:8080",
  apiKey: "your-api-key",
  schemas: "./definitions",
});
```

## Commands

### Initialize Project

```bash
npx atombase init
```

Creates `atombase.config.ts` and `definitions/` directory.

### Definitions

Manage project definitions on the server.

```bash
# List all definitions
npx atombase definitions list

# Get definition details
npx atombase definitions get <name>

# Push all local definition files to server
npx atombase definitions push

# Push a specific definition by name
npx atombase definitions push <name>

# Preview schema changes without applying
npx atombase definitions diff [file]

# View version history
npx atombase definitions history <name>

```

### Databases

Manage project databases.

```bash
# List all databases
npx atombase databases list

# Get database details
npx atombase databases get <id>

# Create a new database
npx atombase databases create <id> --definition <definition>

# Delete a database
npx atombase databases delete <id> [-f]
```

## Definition Files

Define definitions in the `definitions/` directory:

- global definitions must be in `*.global.ts`
- user definitions must be in `*.user.ts`
- organization definitions must be in `*.org.ts`

```typescript
// definitions/my-app.global.ts
import { defineGlobal, defineSchema, defineAccess, defineTable, c, allow, isNull } from "@atombase/definitions";

const schema = defineSchema({
  users: defineTable({
    id: c.integer().primaryKey(),
    name: c.text().notNull(),
    email: c.text().notNull().unique(),
    created_at: c.text().notNull().default("CURRENT_TIMESTAMP"),
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

Null checks use explicit helpers:

```typescript
users: {
  select: ({ prev }) => isNull(prev.deleted_at),
}
```

## Workflow

1. Define a definition locally in `definitions/`
2. Preview changes: `npx atombase definitions diff`
3. Push to server: `npx atombase definitions push`
4. Create databases: `npx atombase databases create acme --definition my-app`
5. Databases migrate lazily on first access

## Options

```bash
# Skip SSL certificate verification (development only)
npx atombase -k definitions list
npx atombase --insecure definitions list
```

## License

Atombase is [fair-source](https://fair.io) licensed under [FSL-1.1-MIT](../../LICENSE).
