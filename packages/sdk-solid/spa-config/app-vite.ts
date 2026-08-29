import { existsSync, readdirSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import type { UserConfig } from 'vite'
import solid from 'vite-plugin-solid'
import { poCatalogPlugin } from './po-plugin.ts'

export { poCatalogPlugin } from './po-plugin.ts'

/** Shared Vite config for mounted app SPAs (library build + watch). */
export function createAppViteConfig(importMetaUrl: string, mode: string): UserConfig {
  const dir = dirname(fileURLToPath(importMetaUrl))
  const repoRoot = resolve(dir, '../../..')
  const sdkSolid = resolve(repoRoot, 'packages/sdk-solid')
  const isWatch = process.argv.includes('--watch')

  return {
    root: dir,
    plugins: [solid(), poCatalogPlugin(repoRoot)],
    resolve: {
      alias: {
        '@kaizengo/sdk-solid': sdkSolid,
      },
    },
    build: {
      ...(isWatch
        ? {
            watch: {
              include: [`${dir}/**/*`, `${sdkSolid}/**/*`],
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
  return resolve(repoRoot, 'packages/sdk-solid/ui/src')
}
