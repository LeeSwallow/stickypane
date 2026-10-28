import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vitest/config'

// Component tests opt into jsdom with `// @vitest-environment jsdom`; the
// "browser" condition makes Svelte resolve to its client runtime there.
export default defineConfig({
  plugins: [svelte()],
  resolve: { conditions: ['browser'] },
  test: { environment: 'node', include: ['src/**/*.test.ts'] },
})
