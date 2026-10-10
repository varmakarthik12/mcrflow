import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

const controlPort = process.env.CONTROL || process.env.MCRFLOW_CONTROL_PORT || '3081'
const backendTarget = process.env.MCRFLOW_CONTROL_URL || `http://localhost:${controlPort}`

export default defineConfig({
  plugins: [react()],
  server: {
    port: 3080,
    host: true,
    proxy: {
      '/api': {
        target: backendTarget,
        changeOrigin: true,
      },
      '/hls': {
        target: backendTarget,
        changeOrigin: true,
      },
      '/epg': {
        target: backendTarget,
        changeOrigin: true,
      },
      '/media': {
        target: backendTarget,
        changeOrigin: true,
      },
      '/data': {
        target: backendTarget,
        changeOrigin: true,
      },
      '/logos': {
        target: backendTarget,
        changeOrigin: true,
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  }
})
