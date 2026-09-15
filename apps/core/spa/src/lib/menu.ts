import type { MenuItem } from './types'

/** Flatten a menu tree to leaf items (those with view/route, or nodes without children). */
export function flattenMenuItems(items: MenuItem[]): MenuItem[] {
  const out: MenuItem[] = []
  for (const item of items) {
    if (item.children?.length) {
      out.push(...flattenMenuItems(item.children))
      continue
    }
    out.push(item)
  }
  return out
}

/** First leaf menu that has a view (for initial selection). */
export function firstMenuLeaf(items: MenuItem[]): MenuItem | undefined {
  for (const item of flattenMenuItems(items)) {
    if (item.view) return item
  }
  return flattenMenuItems(items)[0]
}
