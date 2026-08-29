const handlers = new Map<string, () => void>()

/** Register a named handler for keymap actions (`custom:<name>`). Returns an unregister fn. */
export function registerKeymapAction(name: string, handler: () => void): () => void {
  handlers.set(name, handler)
  return () => {
    if (handlers.get(name) === handler) handlers.delete(name)
  }
}

export function runCustomAction(name: string): boolean {
  const fn = handlers.get(name)
  if (!fn) return false
  fn()
  return true
}
