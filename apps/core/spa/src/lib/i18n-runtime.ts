/// <reference path="../../vite/virtual-kaizengo-i18n.d.ts" />
import { catalogs, localeDirs } from 'virtual:kaizengo-i18n'
import { applyLocale, type TextDirection, type Translator } from './i18n'

let locale = readDocumentLocale()
const listeners = new Set<() => void>()

function readDocumentLocale(): string {
  if (typeof document === 'undefined') return 'en'
  return document.documentElement.lang?.trim() || 'en'
}

function dirOf(id: string): TextDirection {
  return localeDirs[id] === 'rtl' ? 'rtl' : 'ltr'
}

function lookup(key: string): string {
  return catalogs[locale]?.[key] ?? catalogs.en?.[key] ?? key
}

function emit() {
  for (const l of listeners) l()
}

export function subscribeI18n(listener: () => void): () => void {
  listeners.add(listener)
  return () => listeners.delete(listener)
}

export function getI18nLocaleSnapshot(): string {
  return locale
}

/** Translate a catalog key from the Vite-compiled .po catalogs. */
export function t(key: string, ...args: Array<string | number>): string {
  let s = lookup(key)
  for (const a of args) {
    s = s.replace(/%[sd]/, String(a))
  }
  return s
}

export function getI18nLocale(): string {
  return locale
}

export function setI18nLocale(next: string) {
  const id = next.trim() || 'en'
  locale = id
  applyLocale(id, dirOf(id))
  emit()
}

export function bundledTranslator(): Translator {
  const loc = locale
  return {
    locale: loc,
    dir: dirOf(loc),
    messages: { ...(catalogs.en ?? {}), ...(catalogs[loc] ?? {}) },
    t,
  }
}
