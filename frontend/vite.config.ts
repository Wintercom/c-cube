import { fileURLToPath, URL } from 'node:url'
import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import vueJsx from '@vitejs/plugin-vue-jsx'

export default defineConfig({
  server: {
    port: 5000, // 将端口改为 5000
    strictPort: true, // 端口被占用时直接退出，不尝试其他端口:cite[1]
    proxy: {
      "/api": {
        target: "http://192.168.31.223:8080/",
        changeOrigin: true
      }
    }
  },
  plugins: [
    vue(),
    vueJsx(),
  ],
  resolve: {
    alias: {
      '@': fileURLToPath(new URL('./src', import.meta.url))
    },
  }
})
