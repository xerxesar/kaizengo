import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const dir = dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  root: dir,
  plugins: [svelte()],
  build: {
    lib: {
      entry: resolve(dir, 'main.ts'),
      formats: ['es'],
      fileName: () => 'spa.js',
      cssFileName: 'spa',
    },
    outDir: resolve(dir, 'dist'),
    emptyOutDir: true,
    cssCodeSplit: false,
  },
})
