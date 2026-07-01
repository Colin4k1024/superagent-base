/// <reference types="vitest" />
import { defineConfig, loadEnv } from 'vite'
import react from '@vitejs/plugin-react'
import path from 'path'

// https://vitejs.dev/config/
export default defineConfig(({ mode }) => {
  const env = loadEnv(mode, process.cwd(), '')
  const apiBase = env.VITE_API_BASE || 'http://localhost:8888'

  return {
    plugins: [react()],
    resolve: {
      alias: {
        '@': path.resolve(__dirname, './src'),
      },
    },
    server: {
      host: '0.0.0.0',
      port: parseInt(env.PORT || '3500'),
      proxy: {
        '/api': {
          target: apiBase,
          changeOrigin: true,
        },
        '/grpc': {
          target: apiBase.replace(/:\d+$/, ':50051'),
          changeOrigin: true,
        },
        '/metrics': {
          target: apiBase,
          changeOrigin: true,
        },
      },
    },
    build: {
      rollupOptions: {
        output: {
          manualChunks: {
            'vendor': ['react', 'react-dom', 'react-router-dom'],
            'monaco': ['@monaco-editor/react'],
            'xyflow': ['@xyflow/react'],
            'query': ['@tanstack/react-query', 'zustand'],
          },
        },
      },
    },
    test: {
      globals: true,
      environment: 'jsdom',
      setupFiles: ['./src/test/setup.ts'],
      exclude: ['e2e/**', 'node_modules/**'],
      css: true,
    },
  }
})
