import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { poCatalogPlugin } from '../../../packages/sdk-svelte/spa-config/app-vite.ts'

const spaRoot = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(spaRoot, '../../..')

export default defineConfig({
  base: '/app/',
  plugins: [svelte(), tailwindcss(), poCatalogPlugin(repoRoot)],
  resolve: {
    dedupe: ['svelte'],
    alias: {
      '@apps': resolve(repoRoot, 'apps'),
      '@kaizengo/sdk-svelte': resolve(repoRoot, 'packages/sdk-svelte'),
    },
  },
  server: {
    port: 5173,
    open: '/app/',
    fs: {
      allow: [repoRoot],
    },
    proxy: {
      '/graphql': { target: 'http://localhost:8080', changeOrigin: true },
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/auth': { target: 'http://localhost:8080', changeOrigin: true },
      '/static': { target: 'http://localhost:8080', changeOrigin: true },
      '/favicon.ico': { target: 'http://localhost:8080/static/icon.ico', changeOrigin: true },
      '/health': { target: 'http://localhost:8080', changeOrigin: true },
      '/playground': { target: 'http://localhost:8080', changeOrigin: true },
      '/apps': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
