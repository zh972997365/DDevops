import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import { fileURLToPath, URL } from 'node:url'

// 构建产物输出到 ../internal/webui/dist，由 Go 通过 go:embed 内嵌进二进制，
// 运行时前端完全由内嵌的静态资源提供（见 internal/router 与 internal/webui）。
export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url)),
    },
  },
  server: {
    port: 5173,
    proxy: {
      '/api': 'http://localhost:8081',
    },
  },
  build: {
    outDir: '../internal/webui/dist',
    // 每次构建前清空产物目录，避免上一版带 hash 的旧文件长期堆积。
    emptyOutDir: true,
  },
})
