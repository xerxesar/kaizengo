# Auth, identity, and permissions

KaizenGo separates **who you are** (identity), **how you prove it** (auth sessions), and **what you may do** (permissions).

```text
Login → auth (session cookie)
              |
              v
        internal/auth.Principal
              |
    +---------+---------+
    |                   |
 Identity GraphQL    permissions.Can / MustAllow
 (org, users, …)     (RBAC on resources/actions)
```

## Default credentials

On first run identity seeds an admin user; auth sets the password:

| | Default | Override |
|--|---------|----------|
| Email | `admin@kaizengo.local` | `KaizenGo_ADMIN_EMAIL` |
| Password | `changeme` | `KaizenGo_ADMIN_PASSWORD` |

Change the password in production. The server logs a warning when the default password is used.

## Session auth

Kernel package: `internal/auth`.

| Piece | Detail |
|-------|--------|
| Cookie | `kg_session` — HttpOnly, `Path=/`, SameSite=Lax |
| Alternate | `Authorization: Bearer <sessionID>` |
| Lifetime | 7 days |
| Principal | `UserID`, `OrgID`, `Email`, `Name`, `SessionID` |

`cmd/server/main.go` installs `auth.SessionMiddleware`, which validates the session via the auth app (`auth` on the host bag).

### HTTP routes (public)

Registered by `apps/auth` on Mount:

| Method | Path | Body / result |
|--------|------|----------------|
| `POST` | `/auth/login` | `{ "email", "password" }` → sets cookie + user JSON |
| `POST` | `/auth/logout` | clears session + cookie |
| `GET` | `/auth/me` | current user (includes roles when permissions is loaded) |

### Protected routes

Core wraps these with `auth.RequireAuth` (401 without a valid principal):

- `POST` / `GET` `/graphql`
- `GET` `/api/apps` (nav catalog for the shell)

`/health`, `/auth/*`, static assets, and `/app-assets/*` stay public.

### Shell

`apps/core/spa` gates the UI on `/auth/me`. Login lives in `views/Login.svelte`. SPA fetches use `credentials: 'include'` so the session cookie is sent.

```bash
# After login in the browser, or with curl:
curl -c cookies.txt -s http://localhost:8080/auth/login \
  -H 'content-type: application/json' \
  -d '{"email":"admin@kaizengo.local","password":"changeme"}'

curl -b cookies.txt -s http://localhost:8080/api/apps
curl -b cookies.txt -s http://localhost:8080/graphql \
  -H 'content-type: application/json' \
  -d '{"query":"{ me { email name roles } }"}'
```

## Auth app

`apps/auth` — credentials, sessions, `/auth/*`, and the `me` GraphQL field. Spec-driven (`engine.New`); no SPA.

| Piece | Detail |
|-------|--------|
| Module | `apps/auth/module.go` |
| Host name | `auth` |
| Capability | `auth.sessions` |
| Postgres | schema `auth` (`KaizenGo_AUTH_SCHEMA`) |
| Depends | `core`, `identity` |

## Identity app

`apps/identity` — org structure, users, and memberships. Spec-driven (`engine.New` + user email hooks); views under `apps/identity/views/`.

| Piece | Path |
|-------|------|
| Module | `apps/identity/module.go` |
| Models | `identity.models` on the host bag |
| Postgres | schema `identity` (`KaizenGo_IDENTITY_SCHEMA`) |
| SPA | Overview, Structure, Users, Settings tabs |

**Org unit types:** `business_unit`, `department`, `location`, `team`, `position`.

**GraphQL** (engine CRUD, org-scoped): `identityUsers`, `identityOrganizations`, `identityOrgUnits`, `identityMemberships`, plus create/update/delete mutations. Nest org units in the client from `identityOrgUnits`.

Demo seed includes **Acme Mining Corp** and a small org tree for UI exploration.

## Permissions app

`apps/permissions` — backend RBAC only (no SPA, no nav). Depends on `core` and `identity`.

| Piece | Detail |
|-------|--------|
| Host name | `permissions` |
| Postgres | schema `permissions` (`KaizenGo_PERMISSIONS_SCHEMA`) |
| API | `Roles`, `Can`, `MustAllow`, `AssignRole`, `SeedAdmin` |

**Roles (in-code matrix):**

| Role | Access |
|------|--------|
| `admin` | `*` / `*` |
| `member` | read org / org units / users; read `counter` |

**Resources today:** `identity.organization`, `identity.org_units`, `identity.users`, `counter`.

Engine GraphQL CRUD is permission-gated via `MustAllow`. On setup, the admin user receives the `admin` role if missing.

This matches the whitepaper split: identity supplies subjects and org context; auth proves identity; permissions evaluates access.

## Environment

| Variable | Meaning |
|----------|---------|
| `KaizenGo_POSTGRES_DSN` | PostgreSQL DSN (required for identity, auth, permissions, and event-sourced apps) |
| `KaizenGo_IDENTITY_SCHEMA` | Identity schema (default `identity`) |
| `KaizenGo_AUTH_SCHEMA` | Auth schema (default `auth`) |
| `KaizenGo_PERMISSIONS_SCHEMA` | Permissions schema (default `permissions`) |
| `KaizenGo_ADMIN_EMAIL` | Seed admin email |
| `KaizenGo_ADMIN_PASSWORD` | Seed admin password |
| `KaizenGo_APPS` | Which apps to load (include `identity` + `auth` for login; `permissions` for RBAC) |

Minimal auth stack:

```bash
KaizenGo_APPS=core,identity,auth,permissions ./bin/server
```

## Note on i18n before login

`/graphql` (including the `i18n` query) requires a session. Login copy comes from Vite-compiled `.po` catalogs, so the sign-in screen does not need GraphQL. After sign-in, `syncDocumentLocale` applies the user's settings locale.
