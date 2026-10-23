import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'
import tailwindcss from '@tailwindcss/vite'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'
import { poCatalogPlugin } from './vite/po-plugin.ts'

const spaRoot = dirname(fileURLToPath(import.meta.url))
const repoRoot = resolve(spaRoot, '../../..')

export default defineConfig({
  base: '/app/',
  plugins: [react(), tailwindcss(), poCatalogPlugin(repoRoot)],
  resolve: {
    dedupe: ['react', 'react-dom'],
    alias: {
      '@': resolve(spaRoot, 'src'),
      '@apps': resolve(repoRoot, 'apps'),
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
      '/web': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
