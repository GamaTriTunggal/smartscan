import { defineConfig } from 'vitest/config'
import vue from '@vitejs/plugin-vue'
import path from 'path'
import { fileURLToPath } from 'url'

const __dirname = path.dirname(fileURLToPath(import.meta.url))

export default defineConfig({
  plugins: [vue()],
  test: {
    globals: true,
    environment: 'happy-dom',
    setupFiles: ['./tests/setup.js'],
    include: ['src/**/*.{test,spec}.{js,ts}', 'tests/**/*.{test,spec}.{js,ts}'],
    exclude: ['node_modules', 'dist'],
    coverage: {
      provider: 'v8',
      reporter: ['text', 'json', 'html'],
      exclude: [
        'node_modules/',
        'tests/',
        '**/*.d.ts',
        '**/*.config.{js,ts}',
        '**/main.js',
      ],
    },
  },
  resolve: {
    alias: [
      { find: 'driver.js/dist/driver.css', replacement: path.resolve(__dirname, './tests/mocks/driver.mock.js') },
      { find: 'driver.js', replacement: path.resolve(__dirname, './tests/mocks/driver.mock.js') },
      { find: '@', replacement: path.resolve(__dirname, './src') },
    ],
  },
})
