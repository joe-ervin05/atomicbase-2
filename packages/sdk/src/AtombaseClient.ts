import { AtombaseQueryBuilder } from "./AtombaseQueryBuilder.js";
import { AtombaseBuilder } from "./AtombaseBuilder.js";
import { AtombaseError } from "./AtombaseError.js";
import { AuthClient, OrganizationAuthClient } from "./AuthClient.js";
import { DatabasesClient } from "./DatabasesClient.js";
import { DefinitionsClient } from "./DefinitionsClient.js";
import type { AtombaseClientOptions, AtombaseBatchResponse } from "./types.js";

/**
 * Database-scoped client for operations.
 * Created by calling `client.database()` for the caller's own user database,
 * or `client.database('db-id')` for an explicit database.
 *
 * @example
 * ```ts
 * const databaseClient = client.database('acme-corp')
 *
 * // Query with fluent filters
 * const { data, error } = await databaseClient
 *   .from('users')
 *   .select('id', 'name')
 *   .eq('status', 'active')
 *   .limit(10)
 *
 * // Insert data
 * const { data } = await databaseClient
 *   .from('users')
 *   .insert({ name: 'Alice', email: 'alice@example.com' })
 * ```
 */
export class DatabaseClient {
  readonly baseUrl: string;
  readonly apiKey?: string;
  readonly sessionToken?: string;
  readonly headers: Record<string, string>;
  readonly databaseId?: string;
  private readonly fetchFn: typeof fetch;

  constructor(options: AtombaseClientOptions & { databaseId?: string }) {
    this.baseUrl = options.url.replace(/\/$/, "");
    this.apiKey = options.apiKey;
    this.sessionToken = options.sessionToken;
    this.headers = options.headers ?? {};
    this.databaseId = options.databaseId;
    this.fetchFn = options.fetch ?? globalThis.fetch.bind(globalThis);
  }

  /**
   * Start a query on a table.
   *
   * @example
   * ```ts
   * const { data } = await databaseClient.from('users').select()
   * ```
   */
  from<T = Record<string, unknown>>(table: string): AtombaseQueryBuilder<T> {
    return new AtombaseQueryBuilder<T>({
      table,
      baseUrl: this.baseUrl,
      apiKey: this.apiKey,
      sessionToken: this.sessionToken,
      fetch: this.fetchFn,
      headers: {
        ...this.headers,
        ...(this.databaseId ? { Database: this.databaseId } : {}),
      },
    });
  }

  /**
   * Execute multiple operations in a single atomic transaction.
   * All operations succeed or all fail together.
   *
   * @example
   * ```ts
   * const { data, error } = await databaseClient.batch([
   *   databaseClient.from('users').insert({ name: 'Alice' }),
   *   databaseClient.from('users').insert({ name: 'Bob' }),
   *   databaseClient.from('counters').update({ count: 2 }).eq('id', 1),
   * ])
   * ```
   */
  async batch<T extends unknown[] = unknown[]>(
    queries: AtombaseBuilder<unknown>[]
  ): Promise<AtombaseBatchResponse<T>> {
    const operations = queries.map((q) => q.toBatchOperation());

    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...this.headers,
      ...(this.databaseId ? { Database: this.databaseId } : {}),
    };

    if (this.sessionToken) {
      headers["Authorization"] = `Bearer ${this.sessionToken}`;
    } else if (this.apiKey) {
      headers["Authorization"] = `Bearer service.${this.apiKey}`;
    }

    try {
      const response = await this.fetchFn(`${this.baseUrl}/data/batch`, {
        method: "POST",
        headers,
        body: JSON.stringify({ operations }),
      });

      if (!response.ok) {
        const errorBody = await response.json().catch(() => ({}));
        const error = AtombaseError.fromResponse(errorBody, response.status);
        return { data: null, error };
      }

      const rawData = await response.json();
      const rawResults = rawData.results as unknown[];

      // Apply client-side post-processing based on resultMode
      const processedResults = rawResults.map((result, index) => {
        const op = operations[index];
        if (!op) return result;

        switch (op.resultMode) {
          case "count": {
            const r = result as { count?: number };
            return r.count ?? 0;
          }

          case "withCount":
            return result;

          case "single": {
            const data = result as unknown[];
            if (!data || data.length === 0) {
              return { __error: "NOT_FOUND", message: "No rows returned" };
            }
            if (data.length > 1) {
              return { __error: "MULTIPLE_ROWS", message: "Multiple rows returned" };
            }
            return data[0];
          }

          case "maybeSingle": {
            const data = result as unknown[];
            return data?.[0] ?? null;
          }

          default:
            return result;
        }
      });

      return { data: { results: processedResults as T }, error: null };
    } catch (err) {
      const error = AtombaseError.networkError(err);
      return { data: null, error };
    }
  }
}

/**
 * Atombase client for multi-database operations.
 * Use `.database()` to get a database-scoped client for querying.
 * Use `.definitions` and `.databases` for project-scoped actions.
 * Use `.auth` and `.orgs` for session/auth-context actions.
 *
 * @example
 * ```ts
 * import { createClient } from '@atombase/sdk'
 *
 * const client = createClient({
 *   url: 'http://localhost:8080',
 *   apiKey: 'your-api-key',
 * })
 *
 * // Manage databases
 * const { data: database } = await client.databases.create({
 *   id: 'acme-corp',
 *   definition: 'my-app'
 * })
 *
 * // Get a database-scoped client for operations
 * const acme = client.database('acme-corp')
 *
 * // Query the database
 * const { data, error } = await acme
 *   .from('users')
 *   .select('id', 'name')
 *   .eq('status', 'active')
 *   .limit(10)
 * ```
 */
export class AtombaseClient {
  readonly baseUrl: string;
  readonly apiKey?: string;
  readonly sessionToken?: string;
  readonly headers: Record<string, string>;
  private readonly fetchFn: typeof fetch;

  /**
   * Client for managing project databases.
   *
   * @example
   * ```ts
   * // List all databases
   * const { data: databases } = await client.databases.list()
   *
   * // Create a new database
   * const { data: database } = await client.databases.create({
   *   id: 'acme-corp',
   *   definition: 'my-app'
   * })
   *
   * // Get database details
   * const { data: database } = await client.databases.get('acme-corp')
   *
   * // Delete a database
   * await client.databases.delete('acme-corp')
   * ```
   */
  readonly databases: DatabasesClient;
  readonly definitions: DefinitionsClient;
  readonly auth: AuthClient;
  /**
   * Client for organization membership and org management auth routes.
   * Retained as a convenience alias for `client.auth.orgs`.
   */
  readonly orgs: OrganizationAuthClient;

  constructor(options: AtombaseClientOptions) {
    this.baseUrl = options.url.replace(/\/$/, "");
    this.apiKey = options.apiKey;
    this.sessionToken = options.sessionToken;
    this.headers = options.headers ?? {};
    this.fetchFn = options.fetch ?? globalThis.fetch.bind(globalThis);

    // Initialize databases client
    this.databases = new DatabasesClient({
      baseUrl: this.baseUrl,
      apiKey: this.apiKey,
      headers: this.headers,
      fetch: this.fetchFn,
    });
    this.definitions = new DefinitionsClient({
      baseUrl: this.baseUrl,
      apiKey: this.apiKey,
      headers: this.headers,
      fetch: this.fetchFn,
    });
    this.auth = new AuthClient({
      baseUrl: this.baseUrl,
      apiKey: this.apiKey,
      sessionToken: this.sessionToken,
      headers: this.headers,
      fetch: this.fetchFn,
    });
    this.orgs = this.auth.orgs;
  }

  /**
   * Create a database-scoped client for operations.
   *
   * @example
   * ```ts
   * const databaseClient = client.database('acme-corp')
   * const { data } = await databaseClient.from('users').select()
   * ```
   */
  database(databaseTarget?: string): DatabaseClient {
    return new DatabaseClient({
      url: this.baseUrl,
      apiKey: this.apiKey,
      sessionToken: this.sessionToken,
      fetch: this.fetchFn,
      headers: this.headers,
      databaseId: databaseTarget,
    });
  }

  withSession(sessionToken: string): AtombaseClient {
    return new AtombaseClient({
      url: this.baseUrl,
      apiKey: this.apiKey,
      sessionToken,
      fetch: this.fetchFn,
      headers: this.headers,
    });
  }
}

/**
 * Create an Atombase client.
 *
 * @example
 * ```ts
 * const client = createClient({
 *   url: 'http://localhost:8080',
 *   apiKey: 'your-api-key',
 * })
 *
 * // Get a database client and query
 * const { data } = await client.database('my-database').from('users').select()
 * ```
 */
export function createClient(options: AtombaseClientOptions): AtombaseClient {
  return new AtombaseClient(options);
}
