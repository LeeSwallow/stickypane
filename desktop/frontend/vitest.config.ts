import { svelte } from '@sveltejs/vite-plugin-svelte'
import { defineConfig } from 'vitest/config'

// Every run reads dates in one zone, the one the fixtures were recorded in
// (desktop/scripts/shape-fixtures.sh sets it too): a chat's "day" and a
// note's time name the same day on a laptop in Seoul and on CI in UTC.
// Set here, before the test workers start, so they inherit it.
process.env.TZ = 'Asia/Seoul'

// Tests run in node: a component can be rendered on the server
// (svelte/server, as the board and the design system's tests do), which
// needs no DOM. Component tests that need one opt into jsdom with
// `// @vitest-environment jsdom`; the "browser" condition makes Svelte
// resolve to its client runtime there.
export default defineConfig({
  plugins: [svelte()],
  resolve: { conditions: ['browser'] },
  test: { environment: 'node', include: ['src/**/*.test.ts'] },
})
