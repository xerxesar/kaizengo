import { existsSync, readdirSync, readFileSync } from 'node:fs'
import { join } from 'node:path'
import type { Plugin } from 'vite'
import { parsePO } from './po.ts'

const VIRTUAL_ID = 'virtual:kaizengo-i18n'
const RESOLVED_ID = '\0' + VIRTUAL_ID

const RTL = new Set(['fa', 'ar', 'he', 'ur', 'ps'])

export function poCatalogPlugin(repoRoot: string): Plugin {
  return {
    name: 'kaizengo-po-catalog',
    resolveId(id) {
      if (id === VIRTUAL_ID) return RESOLVED_ID
    },
    load(id) {
      if (id !== RESOLVED_ID) return
      const files = findPoFiles(repoRoot)
      for (const file of files) this.addWatchFile(file)
      const { catalogs, localeDirs } = loadCatalogs(files)
      return `export const catalogs = ${JSON.stringify(catalogs)};
export const localeDirs = ${JSON.stringify(localeDirs)};
`
    },
    handleHotUpdate({ file, server }) {
      if (!file.endsWith('.po')) return
      const mod = server.moduleGraph.getModuleById(RESOLVED_ID)
      if (!mod) return
      server.moduleGraph.invalidateModule(mod)
      server.ws.send({ type: 'full-reload' })
      return []
    },
  }
}

export function findPoFiles(repoRoot: string): string[] {
  const out: string[] = []
  const platform = join(repoRoot, 'internal/platform/i18n/locale')
  collectPo(platform, out)
  const appsDir = join(repoRoot, 'apps')
  if (!existsSync(appsDir)) return out
  for (const entry of readdirSync(appsDir, { withFileTypes: true })) {
    if (!entry.isDirectory()) continue
    collectPo(join(appsDir, entry.name, 'locale'), out)
  }
  return out.sort()
}

function collectPo(dir: string, out: string[]) {
  if (!existsSync(dir)) return
  for (const name of readdirSync(dir)) {
    if (name.endsWith('.po')) out.push(join(dir, name))
  }
}

export function loadCatalogs(files: string[]): {
  catalogs: Record<string, Record<string, string>>
  localeDirs: Record<string, 'ltr' | 'rtl'>
} {
  const catalogs: Record<string, Record<string, string>> = {}
  const localeDirs: Record<string, 'ltr' | 'rtl'> = {}
  for (const file of files) {
    const locale = file.replace(/\\/g, '/').split('/').pop()!.replace(/\.po$/, '')
    const msgs = parsePO(readFileSync(file, 'utf8'))
    if (!catalogs[locale]) catalogs[locale] = {}
    Object.assign(catalogs[locale], msgs)
    localeDirs[locale] = RTL.has(locale) ? 'rtl' : 'ltr'
  }
  return { catalogs, localeDirs }
}
