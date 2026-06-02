import { AtombaseError } from "./AtombaseError.js";
import type {
  AtombaseResponse,
  CreateOrganizationInviteOptions,
  CreateOrganizationMemberOptions,
  CreateOrganizationOptions,
  CreateUserDatabaseOptions,
  MagicLinkCompleteResponse,
  MagicLinkStartOptions,
  MagicLinkStartResponse,
  Organization,
  OrganizationContext,
  OrganizationInvite,
  OrganizationMember,
  TransferOrganizationOwnershipOptions,
  UpdateOrganizationMemberOptions,
  UpdateOrganizationOptions,
  User,
} from "./types.js";

type AuthMode = "none" | "sessionOrService";

type SharedClientOptions = {
  baseUrl: string;
  apiKey?: string;
  sessionToken?: string;
  headers: Record<string, string>;
  fetch: typeof fetch;
};

class AuthRequestClient {
  protected readonly baseUrl: string;
  protected readonly apiKey?: string;
  protected readonly sessionToken?: string;
  protected readonly headers: Record<string, string>;
  protected readonly fetchFn: typeof fetch;

  constructor(options: SharedClientOptions) {
    this.baseUrl = options.baseUrl;
    this.apiKey = options.apiKey;
    this.sessionToken = options.sessionToken;
    this.headers = options.headers;
    this.fetchFn = options.fetch;
  }

  protected buildHeaders(authMode: AuthMode): Record<string, string> {
    const headers: Record<string, string> = {
      "Content-Type": "application/json",
      ...this.headers,
    };

    if (authMode === "none") {
      return headers;
    }

    if (this.sessionToken) {
      headers.Authorization = `Bearer ${this.sessionToken}`;
      return headers;
    }

    if (this.apiKey) {
      headers.Authorization = `Bearer service.${this.apiKey}`;
    }

    return headers;
  }

  protected async request<T>(method: string, path: string, body?: unknown, authMode: AuthMode = "sessionOrService"): Promise<AtombaseResponse<T>> {
    try {
      const response = await this.fetchFn(`${this.baseUrl}${path}`, {
        method,
        headers: this.buildHeaders(authMode),
        body: body ? JSON.stringify(body) : undefined,
      });

      if (!response.ok) {
        const errorBody = await response.json().catch(() => ({}));
        return { data: null, error: AtombaseError.fromResponse(errorBody, response.status) };
      }

      if (response.status === 204) {
        return { data: undefined as T, error: null };
      }

      const text = await response.text();
      return { data: (text ? JSON.parse(text) : null) as T, error: null };
    } catch (err) {
      return { data: null, error: AtombaseError.networkError(err) };
    }
  }
}

export class OrganizationAuthClient extends AuthRequestClient {
  list(): Promise<AtombaseResponse<Organization[]>> {
    return this.request("GET", "/auth/orgs");
  }

  create(options: CreateOrganizationOptions): Promise<AtombaseResponse<Organization>> {
    return this.request("POST", "/auth/orgs", options);
  }

  get(orgId: string): Promise<AtombaseResponse<OrganizationContext>> {
    return this.request("GET", `/auth/orgs/${encodeURIComponent(orgId)}`);
  }

  listMembers(orgId: string): Promise<AtombaseResponse<OrganizationMember[]>> {
    return this.request("GET", `/auth/orgs/${encodeURIComponent(orgId)}/members`);
  }

  addMember(orgId: string, options: CreateOrganizationMemberOptions): Promise<AtombaseResponse<OrganizationMember>> {
    return this.request("POST", `/auth/orgs/${encodeURIComponent(orgId)}/members`, options);
  }

  updateMember(orgId: string, userId: string, options: UpdateOrganizationMemberOptions): Promise<AtombaseResponse<OrganizationMember>> {
    return this.request("PATCH", `/auth/orgs/${encodeURIComponent(orgId)}/members/${encodeURIComponent(userId)}`, options);
  }

  removeMember(orgId: string, userId: string): Promise<AtombaseResponse<void>> {
    return this.request("DELETE", `/auth/orgs/${encodeURIComponent(orgId)}/members/${encodeURIComponent(userId)}`);
  }

  listInvites(orgId: string): Promise<AtombaseResponse<OrganizationInvite[]>> {
    return this.request("GET", `/auth/orgs/${encodeURIComponent(orgId)}/invites`);
  }

  createInvite(orgId: string, options: CreateOrganizationInviteOptions): Promise<AtombaseResponse<OrganizationInvite>> {
    return this.request("POST", `/auth/orgs/${encodeURIComponent(orgId)}/invites`, options);
  }

  deleteInvite(orgId: string, inviteId: string): Promise<AtombaseResponse<void>> {
    return this.request("DELETE", `/auth/orgs/${encodeURIComponent(orgId)}/invites/${encodeURIComponent(inviteId)}`);
  }

  acceptInvite(orgId: string, inviteId: string): Promise<AtombaseResponse<OrganizationMember>> {
    return this.request("POST", `/auth/orgs/${encodeURIComponent(orgId)}/invites/${encodeURIComponent(inviteId)}/accept`);
  }

  update(orgId: string, options: UpdateOrganizationOptions): Promise<AtombaseResponse<Organization>> {
    return this.request("PATCH", `/auth/orgs/${encodeURIComponent(orgId)}`, options);
  }

  delete(orgId: string): Promise<AtombaseResponse<void>> {
    return this.request("DELETE", `/auth/orgs/${encodeURIComponent(orgId)}`);
  }

  transferOwnership(orgId: string, options: TransferOrganizationOwnershipOptions): Promise<AtombaseResponse<Organization>> {
    return this.request("POST", `/auth/orgs/${encodeURIComponent(orgId)}/transfer-ownership`, options);
  }
}

export class AuthClient extends AuthRequestClient {
  readonly orgs: OrganizationAuthClient;

  constructor(options: SharedClientOptions) {
    super(options);
    this.orgs = new OrganizationAuthClient(options);
  }

  startMagicLink(options: MagicLinkStartOptions): Promise<AtombaseResponse<MagicLinkStartResponse>> {
    return this.request("POST", "/auth/magic-link/start", options, "none");
  }

  completeMagicLink(token: string): Promise<AtombaseResponse<MagicLinkCompleteResponse>> {
    return this.request("GET", `/auth/magic-link/complete?token=${encodeURIComponent(token)}`, undefined, "none");
  }

  signOut(): Promise<AtombaseResponse<void>> {
    return this.request("POST", "/auth/signout");
  }

  me(): Promise<AtombaseResponse<User>> {
    return this.request("GET", "/auth/me");
  }

  createDatabase(options: CreateUserDatabaseOptions): Promise<AtombaseResponse<{
    id: string;
    definitionId: number;
    definitionName: string;
    definitionType: string;
    definitionVersion: number;
  }>> {
    return this.request("POST", "/auth/me/database", options);
  }
}
