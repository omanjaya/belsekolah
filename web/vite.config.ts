import { defineConfig } from 'vitest/config'
import { svelte } from '@sveltejs/vite-plugin-svelte'
import tailwindcss from '@tailwindcss/vite'

export default defineConfig({
  plugins: [tailwindcss(), svelte()],
  build: { outDir: '../internal/webui/dist', emptyOutDir: true },
  // BEL_API: alamat backend untuk `npm run dev` (compose.dev.yaml memakai http://api:8080).
  server: { proxy: { '/api': process.env.BEL_API ?? 'http://127.0.0.1:8080' } },
  test: { include: ['src/**/*.test.ts'] },
})
