import { defineConfig } from 'vitest/config';
import react from '@vitejs/plugin-react';

export default defineConfig({
  plugins: [react()],
  test: {
    environment: 'jsdom',
    setupFiles: ['./test/componentSetup.ts'],
    include: ['./test/components/**/*.test.tsx'],
    restoreMocks: true,
    clearMocks: true,
  },
});
