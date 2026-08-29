/// <reference types="vite/client" />

declare module 'virtual:kaizengo-i18n' {
  export const catalogs: Record<string, Record<string, string>>
  export const localeDirs: Record<string, 'ltr' | 'rtl'>
}

declare module '*.tsx' {
  import type { Component } from 'solid-js'
  const component: Component
  export default component
}
