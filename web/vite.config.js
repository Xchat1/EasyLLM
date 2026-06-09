import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

const devHost = process.env.VITE_DEV_HOST || '127.0.0.1'
const devPort = Number(process.env.VITE_DEV_PORT || 5180)
const apiTarget = process.env.VITE_API_TARGET || 'http://localhost:8022'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    // Keep Vite on a separate port so API proxy traffic can reach the Go backend.
    host: devHost,
    port: devPort,
    strictPort: false,
    proxy: {
      '/api': {
        target: apiTarget,
        changeOrigin: true
      },
      '/v1': {
        target: apiTarget,
        changeOrigin: true
      },
      '/pool': {
        target: apiTarget,
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
