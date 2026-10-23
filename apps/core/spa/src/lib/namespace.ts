export type Namespace = {
  app: string
  name: string
}

/** Parse a namespaced reference (`hellospec.greeting`, `identity.users`). */
export function parseNamespace(ref: string): Namespace {
  const dot = ref.indexOf('.')
  if (dot <= 0 || dot === ref.length - 1) {
    throw new Error(`invalid namespace ${JSON.stringify(ref)}; expected app.name`)
  }
  return { app: ref.slice(0, dot), name: ref.slice(dot + 1) }
}

export function formatNamespace(app: string, name: string): string {
  return `${app}.${name}`
}

export function isNamespaced(ref: string): boolean {
  const dot = ref.indexOf('.')
  return dot > 0 && dot < ref.length - 1
}

function pascal(s: string): string {
  return s
    .split('_')
    .map((p) => (p ? p[0].toUpperCase() + p.slice(1) : ''))
    .join('')
}

function camel(s: string): string {
  const p = pascal(s)
  return p ? p[0].toLowerCase() + p.slice(1) : ''
}

export type CqrsRef = Namespace & {
  /** GraphQL field: `hellospec.greetings` → `hellospecGreetings` */
  field: string
}

/**
 * Resolve a CQRS query/command ref to its GraphQL field name.
 * `hellospec.greetings` → hellospecGreetings; `identity.postUser` → identityPostUser.
 */
export function resolveCqrsRef(ref: string): CqrsRef {
  const { app, name } = parseNamespace(ref.trim())
  return { app, name, field: `${camel(app)}${pascal(name)}` }
}
