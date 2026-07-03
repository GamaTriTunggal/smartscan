import { defineConfig } from 'vite'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [vue()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
    extensions: ['.mjs', '.js', '.ts', '.jsx', '.tsx', '.json', '.vue'],
  },
  server: {
    port: 3000,
    host: true,
    allowedHosts: ['smartscan_frontend', 'localhost'],
    proxy: {
      '/api': {
        target: 'http://smartscan_backend:8080',
        changeOrigin: true
      },
      '/uploads': {
        target: 'http://smartscan_backend:8080',
        changeOrigin: true
      },
      // Proxy scan redirect to backend (for session-based geo tracking)
      // Simple prefix match - QR codes are Base58 (21-22 chars), won't conflict with /src/
      '/s/': {
        target: 'http://smartscan_backend:8080',
        changeOrigin: true
      }
    }
  },
})
