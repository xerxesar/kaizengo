/// <reference types="svelte" />
/// <reference types="vite/client" />

/** Catalog entry from GET /api/apps */
export type NavEntry = {
  id: string
  title: string
  route: string
  moduleUrl: string
}

/**
 * Contract for apps loaded into the core shell via dynamic import().
 * Framework-agnostic: Svelte/React/Vue/vanilla/static HTML all work
 * as long as they implement mount/unmount against a host element.
 */
export type SpaAppModule = {
  id?: string
  title?: string
  mount: (el: HTMLElement) => void | Promise<void>
  unmount?: (el: HTMLElement) => void
}
