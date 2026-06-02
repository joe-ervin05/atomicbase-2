import { test } from "node:test";
import assert from "node:assert/strict";

import { createClient, eq } from "../dist/index.js";

const BASE_URL = process.env.ATOMBASE_CONTRACT_BASE_URL ?? "http://127.0.0.1:8080";
const API_KEY = process.env.ATOMBASE_API_KEY;
const RUN_CONTRACT = process.env.ATOMBASE_CONTRACT === "1";

const skipReason = RUN_CONTRACT
  ? null
  : "Set ATOMBASE_CONTRACT=1 to run live API/SDK contract tests";

async function assertHealthy() {
  const response = await fetch(`${BASE_URL}/health`);
  assert.equal(response.status, 200, "API must be reachable before running contract tests");
}

function buildDefinition(definitionName) {
  return {
    name: definitionName,
    type: "global",
    schema: {
      tables: [
        {
          name: "contacts",
          pk: ["id"],
          columns: {
            id: { name: "id", type: "INTEGER" },
            name: { name: "name", type: "TEXT", notNull: true },
            email: { name: "email", type: "TEXT", notNull: true, unique: true },
            status: { name: "status", type: "TEXT", default: "active" },
          },
        },
      ],
    },
    access: {
      contacts: {
        select: { field: "auth.status", op: "eq", value: "anonymous" },
        insert: { field: "auth.status", op: "eq", value: "anonymous" },
        update: { field: "auth.status", op: "eq", value: "anonymous" },
        delete: { field: "auth.status", op: "eq", value: "anonymous" },
      },
    },
  };
}

test("SDK <-> API contract: core database data flows", { skip: skipReason ?? false }, async (t) => {
  await assertHealthy();

  const suffix = Date.now();
  const definitionName = `sdk-contract-${suffix}`;
  const databaseName = `sdk-contract-database-${suffix}`;

  const client = createClient({
    url: BASE_URL,
    ...(API_KEY ? { apiKey: API_KEY } : {}),
  });

  let definitionCreated = false;
  let databaseCreated = false;

  try {
    const definitionCreate = await client.definitions.create(buildDefinition(definitionName));
    assert.equal(definitionCreate.error, null, `definition creation failed: ${definitionCreate.error?.message}`);
    definitionCreated = true;

    const databaseCreate = await client.databases.create({
      id: databaseName,
      definition: definitionName,
    });
    assert.equal(databaseCreate.error, null, `database creation failed: ${databaseCreate.error?.message}`);
    databaseCreated = true;

    const database = client.database(databaseName);

    await t.test("insert + select + filter flow", async () => {
      const first = await database.from("contacts").insert({
        id: 1,
        name: "Alice",
        email: "alice.contract@example.com",
      });
      assert.equal(first.error, null, first.error?.message);

      const second = await database.from("contacts").insert({
        id: 2,
        name: "Bob",
        email: "bob.contract@example.com",
        status: "inactive",
      });
      assert.equal(second.error, null, second.error?.message);

      const filtered = await database.from("contacts").select("id", "name", "status").where(eq("id", 2)).single();
      assert.equal(filtered.error, null, filtered.error?.message);
      assert.equal(filtered.data?.name, "Bob");
      assert.equal(filtered.data?.status, "inactive");
    });

    await t.test("count and withCount contracts", async () => {
      const countResult = await database.from("contacts").select().count();
      assert.equal(countResult.error, null, countResult.error?.message);
      assert.equal(countResult.data, 2);

      const withCountResult = await database.from("contacts").select("id").limit(1).withCount();
      assert.equal(withCountResult.error, null, withCountResult.error?.message);
      assert.equal(withCountResult.data?.length, 1);
      assert.equal(withCountResult.count, 2);
    });

    await t.test("update + delete flow", async () => {
      const updateResult = await database
        .from("contacts")
        .update({ status: "archived" })
        .where(eq("id", 1));
      assert.equal(updateResult.error, null, updateResult.error?.message);
      assert.equal(updateResult.data?.rows_affected, 1);

      const verifyUpdate = await database
        .from("contacts")
        .select("status")
        .where(eq("id", 1))
        .single();
      assert.equal(verifyUpdate.error, null, verifyUpdate.error?.message);
      assert.equal(verifyUpdate.data?.status, "archived");

      const deleteResult = await database.from("contacts").delete().where(eq("id", 2));
      assert.equal(deleteResult.error, null, deleteResult.error?.message);
      assert.equal(deleteResult.data?.rows_affected, 1);

      const remaining = await database.from("contacts").select().count();
      assert.equal(remaining.error, null, remaining.error?.message);
      assert.equal(remaining.data, 1);
    });

    await t.test("batch atomic rollback contract", async () => {
      const before = await database.from("contacts").select().count();
      assert.equal(before.error, null, before.error?.message);

      const batchResult = await database.batch([
        database.from("contacts").insert({
          id: 10,
          name: "Should Rollback",
          email: "rollback.contract@example.com",
        }),
        database.from("contacts").insert({
          id: 11,
          name: "Duplicate Email",
          email: "alice.contract@example.com",
        }),
      ]);
      assert.notEqual(batchResult.error, null, "batch should fail to verify transaction rollback");

      const after = await database.from("contacts").select().count();
      assert.equal(after.error, null, after.error?.message);
      assert.equal(after.data, before.data, "row count should be unchanged after failed batch");
    });

    await t.test("error code contract for unsafe update", async () => {
      const unsafeUpdate = await database.from("contacts").update({ status: "bad" });
      assert.notEqual(unsafeUpdate.error, null, "update without where should fail");
      assert.equal(unsafeUpdate.error?.code, "MISSING_WHERE_CLAUSE");
    });
  } finally {
    if (databaseCreated) {
      const deleteDatabase = await client.databases.delete(databaseName);
      if (deleteDatabase.error && deleteDatabase.error.code !== "DATABASE_NOT_FOUND") {
        throw new Error(`failed to clean up database ${databaseName}: ${deleteDatabase.error.message}`);
      }
    }
    if (definitionCreated) {
      // Definition delete endpoint does not exist yet; leave cleanup to disposable test names.
    }
  }
});
