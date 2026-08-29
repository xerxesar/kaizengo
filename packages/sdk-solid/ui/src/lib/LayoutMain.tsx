import { children, type JSX, type ParentProps } from 'solid-js'

/** Marks main page content inside `<Layout>`. Renders children in the layout body. */
export function LayoutMain(props: ParentProps): JSX.Element {
  const resolved = children(() => props.children)
  return resolved() as JSX.Element
}
