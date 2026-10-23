import { useEffect } from 'react'
import { registerKeymapAction } from './registry'

/** Register a `custom:<name>` keymap handler for the lifetime of the component. */
export function useKeymapAction(name: string, handler: () => void) {
  useEffect(() => registerKeymapAction(name, handler), [name, handler])
}
