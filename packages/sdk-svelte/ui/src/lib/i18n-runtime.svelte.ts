/// <reference path="../../../spa-config/virtual-kaizengo-i18n.d.ts" />
import { catalogs, localeDirs } from 'virtual:kaizengo-i18n'
import { applyLocale, type TextDirection, type Translator } from './i18n'

let locale = $state(readDocumentLocale())

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
}

export function bundledTranslator(): Translator {
  return {
    locale,
    dir: dirOf(locale),
    messages: { ...(catalogs.en ?? {}), ...(catalogs[locale] ?? {}) },
    t,
  }
}
