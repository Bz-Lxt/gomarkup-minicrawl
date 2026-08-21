import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'

export default defineConfig({
  plugins: [vue()],
  server: {
    host: true,
    port: 5174,
    proxy: {
      '/api': 'http://127.0.0.1:18421',
      '/fixture': 'http://127.0.0.1:18421',
    },
  },
})
