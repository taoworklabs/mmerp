import react from '@vitejs/plugin-react'
import { defineConfig } from 'vite'

const backend = 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  resolve: { alias: { '@': '/src' } }, // resolved against the project root
  server: { proxy: { '/api': backend, '/healthz': backend } },
})
