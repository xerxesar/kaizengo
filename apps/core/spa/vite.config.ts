import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

export default defineConfig({
  base: '/app/',
  plugins: [svelte()],
  server: {
    port: 5173,
    open: '/app/',
    proxy: {
      '/graphql': { target: 'http://localhost:8080', changeOrigin: true },
      '/api': { target: 'http://localhost:8080', changeOrigin: true },
      '/app-assets': { target: 'http://localhost:8080', changeOrigin: true },
      '/static': { target: 'http://localhost:8080', changeOrigin: true },
      '/favicon.ico': { target: 'http://localhost:8080/static/icon.ico', changeOrigin: true },
      '/health': { target: 'http://localhost:8080', changeOrigin: true },
      '/playground': { target: 'http://localhost:8080', changeOrigin: true },
      '/apps': { target: 'http://localhost:8080', changeOrigin: true },
    },
  },
})
