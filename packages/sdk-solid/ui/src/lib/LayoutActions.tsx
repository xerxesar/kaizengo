import { children, onCleanup, onMount, type JSX, type ParentProps } from 'solid-js'
import { getLayoutRegistry } from './layout-context'
import type { LayoutSlot } from './layout-context'

export function LayoutActions(props: ParentProps): null {
  const registry = getLayoutRegistry()
  const resolved = children(() => props.children)

  const actionsSlot: LayoutSlot = () => resolved() as JSX.Element

  onMount(() => {
    registry.setActions(actionsSlot)
    onCleanup(() => registry.setActions(null))
  })

  return null
}
