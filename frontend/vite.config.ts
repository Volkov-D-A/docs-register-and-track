import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // Wails desktop budget: route-level chunks are accepted up to 1.6 MB
    // because the heavy statistics page includes charting libraries.
    chunkSizeWarningLimit: 1600,
  },
})
