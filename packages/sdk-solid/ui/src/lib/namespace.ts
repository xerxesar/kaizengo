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
