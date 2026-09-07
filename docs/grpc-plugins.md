# Out-of-process extensions (gRPC + go-plugin)

**Goal:** Let apps and addons talk across process boundaries with **typed capability contracts**, without replacing GraphQL for the SPA or in-process extension hooks.

**Transport:** [gRPC](https://grpc.io/) over [HashiCorp go-plugin](https://github.com/hashicorp/go-plugin) (subprocess plugins, net/rpc or gRPC handshake, reattach, negotiated protocol versions).

This is a **roadmap**. Same-process apps (`blank-import` + `host.Provide` / `Lookup`) remain the default until this lands.

---

## Why another lane

KaizenGo already has three coupling styles:

| Lane | Audience | Mechanism today |
|------|----------|-----------------|
| North–south | Browser / Solid SDK | GraphQL `/graphql` |
| In-process east–west | Other Go packages in the same binary | `host.Provide` / `Lookup`, `extension.Register` |
| Declared contracts | Startup validation | `provides` / `uses` in `app.yaml` |

`Provide` / `Lookup` only works inside one process. GraphQL is a poor east–west protocol (query strings between services, flat schema, session cookie auth). When an addon must run **out of process** (separate binary, different lifecycle, crash isolation, optional remote host later), we need a real RPC surface.

**gRPC + go-plugin** is that surface. GraphQL stays the SPA / public composition API.

```text
SPA ──GraphQL──► host (core)
                   │
                   ├─ engine CRUD / ACL (local)
                   ├─ extension.Run (in-process hooks)
                   └─ go-plugin clients ──gRPC──► plugin processes
                                                   (capability servers)
```

---

## Design principles

1. **Capabilities stay the names** — `identity.users`, `platform.search.backend`, etc. remain the stable IDs in `provides` / `uses`. gRPC is how a *process* implements or consumes them; GraphQL is how the *browser* consumes a subset.
2. **One capability ≈ one protobuf service** (versioned), e.g. `kaizengo.identity.users.v1.Users`.
3. **GraphQL resolvers and plugin servers share domain logic** — adapters only; do not treat GraphQL as the source of truth for cross-app calls.
4. **UI never speaks gRPC** — Solid keeps `fetch('/graphql', …)` and SDK clients.
5. **Hooks stay in-process** — `model.*.afterCreate`, ordered multi-handler chains. A plugin that needs to react to mutations either:
   - registers an in-process shim that RPCs into the plugin, or
   - implements a **capability** the host calls from a thin hook (preferred for search/audit-style drivers).
6. **Fail at startup** — missing `uses`, failed plugin handshake, or protocol mismatch aborts load with a clear error (same spirit as capability validation today).
7. **Contracts over internals** — plugins never import another app’s `service` / `store` packages; only protobuf + host SDK.

See also [extension-platform.md](extension-platform.md) and [capabilities.md](capabilities.md).

---

## Why HashiCorp go-plugin

| Need | go-plugin fit |
|------|----------------|
| Spawn addon as child process | Host launches plugin binary; bidirectional gRPC |
| Version negotiation | Built-in protocol / MagicCookie handshake |
| Crash isolation | Plugin death doesn’t take down the host (host can restart or fail the capability) |
| Local-first ops | No separate service mesh required for v1 |
| Typed API | gRPC + protobuf; go-plugin’s `GRPCPlugin` interface |
| Same codebase languages | Plugins are Go binaries for v1 (aligned with “Go only” extension non-goal) |

**Not** using go-plugin for: browser traffic, GraphQL federation, or replacing `extension.Run` for same-binary addons.

---

## Capability mapping

| Capability (yaml) | GraphQL (north–south) | gRPC (east–west) |
|-------------------|----------------------|------------------|
| `identity.users` | `identityUsers`, `identityUser`, … | `Users.List`, `Users.Get`, … |
| `platform.search.backend` | `search` (host field) | `SearchBackend.Query` / `Upsert` / `Delete` |
| `permissions.rbac` | (indirect via engine ACL) | `Authorizer.Can` / `MustAllow` (optional; may stay in-proc longer) |
| `appman.apps` | `apps`, `installApp`, … | optional; install lifecycle may stay host-local |

Rules of thumb:

- Publish **capability RPCs other apps need**, not a 1:1 mirror of every engine CRUD field.
- Spec model CRUD remains GraphQL-facing inside the host; plugins call host APIs only through documented capability services (or stay in-process via hooks).

---

## Host responsibilities

Planned host-side pieces (names indicative):

1. **Plugin manager** — discover plugin binaries (config / `KaizenGo_PLUGINS` / appman-installed paths), start via go-plugin, hold clients for process lifetime.
2. **Capability broker** — resolve `uses: [identity.users]` to either:
   - in-process `Provide` implementation, or
   - gRPC client stub from a loaded plugin that `provides` that capability.
3. **Shared protobuf / SDK** — `packages/sdk-go/proto/…` (or `api/proto`) + generated Go; plugins and host depend on the same module.
4. **Auth context propagation** — pass org/user/session (or signed internal token) on gRPC metadata for ACL-aware capability calls; do not reuse browser cookies as the plugin protocol.
5. **Health / shutdown** — graceful stop on host exit; optional restart policy for soft failures.

Apps that remain blank-imported keep working unchanged. Plugins are an **additional** load path.

---

## Plugin binary shape (sketch)

```text
apps/typesense/          # or plugins/typesense/
  app.yaml               # provides: [platform.search.backend]
  cmd/plugin/main.go     # go-plugin Serve + gRPC server
  …
```

```go
// illustrative — not implemented yet
plugin.Serve(&plugin.ServeConfig{
	HandshakeConfig: kaizenHandshake,
	Plugins: map[string]plugin.Plugin{
		"search": &SearchGRPCPlugin{Impl: typesenseBackend},
	},
	GRPCServer: plugin.DefaultGRPCServer,
})
```

Host dials, dispenses the gRPC client, registers it as the provider for `platform.search.backend` before capability validation completes.

---

## Relationship to extension points

| Concern | Stay | Move / add |
|---------|------|------------|
| Ordered `beforeCreate` / `afterCreate` chains | `extension.Run` in-process | — |
| Wildcard audit / search indexing | In-proc handler **or** thin handler → gRPC capability | Plugin implements `SearchBackend` / `AuditSink` |
| `host.Provide("permissions")` | Fine for same binary | Broker prefers plugin stub when provider is out-of-proc |
| Solid `UserPicker` | GraphQL / SDK | Unchanged |

Do **not** make every hook a gRPC round-trip. Prefer: host emits extension point → one registered driver calls plugin once.

---

## Phased plan

### Phase A — Protobuf + in-tree dual serve (foundation)

**Deliverables**

- [ ] Proto definitions for 1–2 pilot capabilities (`platform.search.backend`, optionally `identity.users` read APIs)
- [ ] Codegen in CI / `make proto`
- [ ] Domain service interface implemented once; GraphQL resolver and a **local** gRPC server both call it (same process, no go-plugin yet)
- [ ] Doc: capability ↔ RPC method table in [capabilities.md](capabilities.md)

**Exit criteria:** identity or search callable via generated gRPC client against an in-process server; SPA still on GraphQL only.

---

### Phase B — go-plugin host + one out-of-process addon

**Deliverables**

- [ ] `internal/plugin` (or `internal/module/plugin`): handshake, dispense, lifecycle
- [ ] Config to launch plugin binaries; map each to `provides`
- [ ] Extract Typesense (or a minimal echo/search stub) to a plugin binary implementing `platform.search.backend`
- [ ] Startup: handshake failure or missing `uses` → clear fatal error
- [ ] Integration test: host + plugin process, query via GraphQL `search` still works

**Exit criteria:** host binary does not link Typesense driver code; search works through go-plugin gRPC.

---

### Phase C — Capability broker replaces ad hoc Lookup for cross-app reads

**Deliverables**

- [ ] Broker API: `MustCapability[T](host, "identity.users")` resolving in-proc or plugin
- [ ] Migrate selected `Lookup` call sites that cross app boundaries
- [ ] ACL / org metadata on gRPC context
- [ ] Appman (or install path) can register plugin artifacts alongside in-tree apps

**Exit criteria:** Notes (or equivalent) can consume `identity.users` via broker without importing identity packages; works with identity in-process **or** as plugin (pilot).

---

### Phase D — Hardening (later)

**Deliverables**

- [ ] Protocol version bumps / `identity.users/v2` story aligned with [extension-platform open questions](extension-platform.md#open-questions)
- [ ] Crash restart policy, metrics, deadline defaults
- [ ] Optional: plugin checksum / signed artifacts for installed addons
- [ ] CLI scaffold: `kaizengo new-app … --type plugin`

---

## Non-goals (explicit)

- Replacing GraphQL with gRPC for the SPA
- Exposing go-plugin endpoints on the public HTTP port
- Cross-app SQL / shared DB schemas via plugins
- Non-Go plugin languages in v1 (go-plugin can do other languages later; not required)
- Automatic GraphQL federation from plugin schemas
- Making every `extension.Register` handler a remote call

---

## Risks & mitigations

| Risk | Mitigation |
|------|------------|
| Latency vs in-proc Lookup | Keep hot paths in-process; plugins for isolation / optional addons first |
| Duplicate GraphQL + proto drift | Single domain service; CI check resolvers/RPCs against shared types |
| Auth bugs on metadata | Explicit internal credential; never trust raw browser cookies in plugins |
| Ops complexity (many child processes) | Start with one pilot plugin; document process model in workflow docs |
| Handshake / version skew | go-plugin protocol version + capability semver in yaml (`/v2` when needed) |

---

## Success metrics

1. At least one real addon (search or audit) runs as a **separate binary** via go-plugin and satisfies a `provides` entry.
2. SPA and `@kaizengo/sdk-solid/*` require **no** gRPC or plugin awareness.
3. `uses` / `provides` validation still fails fast whether the provider is blank-imported or a plugin.
4. Docs alone are enough to scaffold a Go plugin that the host loads.

---

## Open questions

1. **Discovery:** env list vs appman-installed plugin dir vs both?
2. **Identity as plugin:** worth the boundary, or keep identity always in-host and only plugin “sidecar” capabilities (search, audit, notify)?
3. **Reattach / parent death:** needed for `make dev` hot restart, or always respawn plugins with the host?
4. **Multi-host later:** does go-plugin local subprocess remain the only mode, or do we later allow dialing a remote gRPC address for the same protos?

Track decisions here as ADR-style bullets when resolved.

---

## Related

- [extension-platform.md](extension-platform.md) — capabilities, hooks, SDK roadmap
- [capabilities.md](capabilities.md) — `provides` / `uses` catalog
- [graphql.md](graphql.md) — north–south API (unchanged by this plan)
- [apps.md](apps.md) — in-process lifecycle and Host bag
- [Go SDK](internals/go-sdk.md) — extension points and services
