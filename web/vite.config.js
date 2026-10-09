import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3080,
    host: true,
    proxy: {
      '/api': {
        target: 'http://localhost:3081',
        changeOrigin: true,
      },
      '/hls': {
        target: 'http://localhost:3081',
        changeOrigin: true,
      },
      '/epg': {
        target: 'http://localhost:3081',
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  }
})
