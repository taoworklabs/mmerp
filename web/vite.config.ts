import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

const backend = 'http://localhost:8080'

export default defineConfig({
  plugins: [react()],
  // alias resolved against the project root; dedupe keeps the kit source (which has its own node_modules) on web's React
  resolve: { alias: { '@': '/src' }, dedupe: ['react', 'react-dom'] },
  // Vitest externalises node_modules, so Base UI would load the kit's own React; inline it to apply dedupe.
  test: { server: { deps: { inline: [/@base-ui/] } } },
  // The kit's fonts live under ../packages/ui.
  server: { proxy: { '/api': backend, '/healthz': backend }, fs: { allow: ['..'] } },
})
