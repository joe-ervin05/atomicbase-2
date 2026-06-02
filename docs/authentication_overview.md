# Atombase Authentication Overview

> [!CAUTION]
> Historical design overview. This document predates the current definitions-first SDK surface in places. For current definition authoring and browser auth flow, use `docs/definition-model.md`, `api/README.md`, and `packages/sdk/README.md`.

This is an early overview of authentication in Atombase.
The plan for authentication is production-ready before we think about being enterprise-ready. That means no SSO, Phone-based MFA, Anonymous sign-ins, email OTP (magic link only), SOC2, HIPPA, or passkeys.

## 1. Structure

There are 3 core objects in this historical auth design: users, databases, and organizations. In the current model, those live under `Project -> Definition -> Database -> Session/Auth Context -> Query`; users and organizations are part of session/auth context.

### Organizations

Organizations can be thought of as 1. a way to organize users into groups and roles within those groups and 2. a gate between users and databases. A user's ability to access data is determined by their membership in the organization that owns the database. Authorization at the data level is determined by a user's role within the organization as well as the data they are requesting. This is explained later within authorization.

Note: by default, users can own a maximum of 1 organization and organizations can only be created by authenticated users. Pro users can change this to be any number in the project auth settings. This is put in place to prevent users from creating many organizations and spiking costs or creating a DoS attack.

## 2. Authentication

Atombase uses server-backed sessions to authenticate users on data requests. Users have roles defined by their organization. Every time a database is requested, it resolves the database's organization, checks membership, and attaches the role to the request.

Note: any email or password change for an existing account requires TOTP if enabled or email OTP if not for sessions older than 24 hours. This is configurable through project auth settings.

### Email & password

We offer email & password auth with functionality for signup, login, verifying email, changing email, changing password, and resetting password. We use Argon2id for hashing and allow configuration of password requirements (minimum length, breached password check) through project auth settings.

### OAuth

Atombase supports identity linking in two ways. First, a signed-in user may add a new identity to their account, provided their email is already verified; in this case, the new identity does not require separate email verification because the existing session proves account ownership. Second, a logged-out user may automatically link a new identity by signing in with it, but only if that identity has a verified email (e.g., verified OAuth). Automatic linking is never allowed for unverified identities. If an unverified account exists for an email and a verified identity later signs in with the same email, Atombase deletes the unverified account and preserves the verified one, ensuring that unverified pre-registrations cannot be used to hijack or block legitimate accounts.

### Magic link

We offer magic link auth with functionality for signup, login, and change email.

### Session lifecycle

A new session is created on every login and the current session is deleted on logout. Additionally, sessions are invalidated for inactivity.

### Rate-limiting

Atombase provides sensible defaults for rate-limiting and pro plan users have the ability to customize these limits. Atombase relies on token buckets for authentication route rate limiting.

### MFA

Atombase provides only TOTP-based MFA.

## 3. Authorization

Authorization is done through definition-level security policies. These policies define how data can be accessed for a specific database definition.

Grants authorization example:

```typescript
// policies/org.grants.ts
import { defineGrants, definePolicy, g, eq, and, inList } from "@atombase/access";
import schema from "../schemas/org.schema.ts";

export default defineGrants(schema, {
  invoices: definePolicy({
    // select/delete use row; insert uses next; update can use both
    select: g.where(({ auth, row }) => eq(row.org_id, auth.org.id)),

    insert: g.where(({ auth, next }) =>
      and(
        eq(auth.status, "authenticated"),
        eq(next.org_id, auth.org.id),
      )
    ),

    update: g.where(({ auth, row, next }) =>
      and(
        eq(auth.status, "authenticated"),
        eq(row.org_id, auth.org.id),
        eq(next.org_id, auth.org.id),
      )
    ),

    delete: g.where(({ auth, row }) =>
      and(
        inList(auth.role, ["owner", "admin"]),
        eq(row.org_id, auth.org.id),
      )
    ),
  }),

  metadata: definePolicy({
    // DB-level style: auth-only condition
    select: g.where(({ auth }) => inList(auth.role, ["owner", "admin"])),
    insert: g.where(({ auth }) => inList(auth.role, ["owner", "admin"])),
    update: g.where(({ auth }) => inList(auth.role, ["owner", "admin"])),
    delete: g.where(({ auth }) => inList(auth.role, ["owner", "admin"])),
  }),

  user_settings: definePolicy({
    select: g.where(({ auth, row }) =>
      and(
        eq(auth.status, "authenticated"),
        eq(row.user_id, auth.id),
      )
    ),

    update: g.where(({ auth, row, next }) =>
      and(
        eq(auth.status, "authenticated"),
        eq(row.user_id, auth.id),
        eq(next.user_id, auth.id),
      )
    ),
  }),
});
```
