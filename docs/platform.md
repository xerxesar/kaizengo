# Platform

Systems an app is built against. The layer model is [Architecture](architecture.md). Repo workflow is [Development](development/index.md).

| Guide | Use when you need to… |
|-------|------------------------|
| [Apps](apps.md) | Register an app, install and upgrade it, use the Host bag |
| [Auth, identity, and permissions](auth.md) | Sessions, login, users, orgs |
| [ACL](acl.md) | Resource ids, `security.yaml`, enforcement |
| [Capabilities](capabilities.md) | `provides` / `uses` and the identity user contract |
| [GraphQL](graphql.md) | The runtime field registry and auth-aware clients |
| [CLI](cli.md) | Web terminal, native `kaizen`, and `kaizengo` scaffolding |
| [External Go apps](grpc-plugins.md) | Domain, hooks, and handlers over go-plugin + gRPC |

Time, calendars, localization, and the settings UI are [Platform APIs](development/platform.md).

The cross-app addon design is a roadmap: [Extension platform plan](extension-platform.md).
