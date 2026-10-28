import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vitest/config'

// Tests run in node: a component can be rendered on the server
// (svelte/server), which needs no DOM. Component tests that need one opt
// into jsdom with `// @vitest-environment jsdom`; the "browser" condition
// makes Svelte resolve to its client runtime there.
export default defineConfig({
  plugins: [svelte()],
  resolve: { conditions: ['browser'] },
  test: { environment: 'node', include: ['src/**/*.test.ts'] },
})
