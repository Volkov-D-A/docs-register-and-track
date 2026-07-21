import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

// https://vitejs.dev/config/
export default defineConfig({
  plugins: [react()],
  build: {
    // The large chart runtime is checked separately after the build. This
    // warning remains a safeguard for other unexpectedly large chunks.
    chunkSizeWarningLimit: 1600,
  },
})
