import { existsSync, readdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import type { UserConfig } from 'vite'
import { poCatalogPlugin } from './po-plugin.ts'

export { poCatalogPlugin } from './po-plugin.ts'

/** Shared Vite config for mounted app SPAs (library build + watch). */
export function createAppViteConfig(importMetaUrl: string, mode: string): UserConfig {
  const dir = dirname(fileURLToPath(importMetaUrl))
  const repoRoot = resolve(dir, '../../..')
  const sdkSvelte = resolve(repoRoot, 'packages/sdk-svelte')
  const uiRoot = resolve(sdkSvelte, 'ui')
  const typesenseSpa = resolve(repoRoot, 'apps/typesense/spa')
  const isWatch = process.argv.includes('--watch')

  return {
    root: dir,
    plugins: [svelte(), poCatalogPlugin(repoRoot)],
    resolve: {
      dedupe: ['svelte'],
      alias: {
        '@kaizengo/sdk-svelte': sdkSvelte,
        '@kaizengo/typesense': typesenseSpa,
      },
    },
    build: {
      ...(isWatch
        ? {
            watch: {
              include: [
                `${dir}/**/*`,
                `${sdkSvelte}/**/*`,
                `${typesenseSpa}/**/*`,
              ],
            },
          }
        : {}),
      lib: {
        entry: resolve(dir, 'main.ts'),
        formats: ['es'],
        fileName: () => 'spa.js',
        cssFileName: 'spa',
      },
      outDir: resolve(dir, 'dist'),
      emptyOutDir: !isWatch,
      cssCodeSplit: false,
      minify: mode === 'production',
      sourcemap: mode !== 'production',
    },
  }
}

/** All app spa dist directories under apps/ that exist (or are expected in dev). */
export function discoverAppDistDirs(repoRoot: string): string[] {
  const appsDir = resolve(repoRoot, 'apps')
  if (!existsSync(appsDir)) return []

  return readdirSync(appsDir, { withFileTypes: true })
    .filter((entry) => entry.isDirectory())
    .map((entry) => resolve(appsDir, entry.name, 'spa', 'dist'))
    .filter((dir) => existsSync(dir))
}

/** UI package source — shell HMR when editing shared components. */
export function uiSourceDir(repoRoot: string): string {
  return resolve(repoRoot, 'packages/sdk-svelte/ui/src')
}
