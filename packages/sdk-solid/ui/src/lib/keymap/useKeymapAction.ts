import { onCleanup, onMount } from 'solid-js'
import { registerKeymapAction } from './registry'

/** Register a `custom:<name>` keymap handler for the lifetime of the component. */
export function useKeymapAction(name: string, handler: () => void) {
  onMount(() => {
    const off = registerKeymapAction(name, handler)
    onCleanup(off)
  })
}
