import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { resolve } from 'path'

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': resolve(__dirname, 'src')
    }
  },
  server: {
    // 开发时前端单独端口；正式使用只开 go run main.go，访问 8026 即可（8022 被系统代理服务占用）
    port: 8022,
    proxy: {
      '/api': {
        target: 'http://localhost:8022',
        changeOrigin: true
      },
      '/v1': {
        target: 'http://localhost:8022',
        changeOrigin: true
      },
      '/pool': {
        target: 'http://localhost:8022',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true
  }
})
