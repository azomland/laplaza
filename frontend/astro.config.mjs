import { defineConfig } from 'astro/config'

export default defineConfig({
  outDir: './dist',
  build: { format: 'directory' },
  server: { port: 4321 },
  vite: {
    server: {
      proxy: {
        '/api': 'http://localhost:8080',
        '/ws': {
          target: 'ws://localhost:8080',
          ws: true,
        },
      },
    },
  },
})
