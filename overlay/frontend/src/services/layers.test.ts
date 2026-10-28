// The frontend's layers (RA13, ADR-0018): screens and their parts reach
// the shell only through src/services, and no code here writes a note's
// text — the shapes' Go code does (RA12).
import { describe, expect, it } from 'vitest'

// every source file of the app, as text (tests aside)
const sources = import.meta.glob(['/src/**/*.ts', '/src/**/*.svelte', '!/src/**/*.test.ts'], { query: '?raw', import: 'default', eager: true }) as Record<string, string>
const all = Object.entries(sources).map(([p, text]) => ({ path: p.replace(/^\/src\//, ''), text }))

it('reads the sources', () => {
  expect(all.some((f) => f.path === 'services/board.ts')).toBe(true)
  expect(all.length).toBeGreaterThan(50)
})

describe('layers', () => {
  it('only src/services reaches the runtime', () => {
    const out = all.filter((f) => !f.path.startsWith('services/') && /['"]@wailsio\/runtime['"]/.test(f.text)).map((f) => f.path)
    expect(out).toEqual([])
  })
  it('only src/services names the shell\'s services', () => {
    const out = all
      .filter((f) => !f.path.startsWith('services/') && f.path !== 'lib/shell.ts' && /\b(home|project|canvas)\(['"][A-Z]/.test(f.text))
      .map((f) => f.path)
    expect(out).toEqual([])
  })
})

describe('no note text is written here', () => {
  // A front matter fence, or a "type:"/"kind:" line put together in a
  // string: what a note body made in TypeScript would look like.
  const fence = /(['"`])---\\n/
  const header = /['"`][^'"`\n]*\b(type|kind):\s*\$\{/
  const joined = /\[\s*['"`]---['"`]/
  it('no front matter or note template is built in the app', () => {
    const out = all.filter((f) => fence.test(f.text) || header.test(f.text) || joined.test(f.text)).map((f) => f.path)
    expect(out).toEqual([])
  })
  it('the check would catch one', () => {
    expect(fence.test("const body = '---\\ntype: board\\n---\\n'")).toBe(true)
    expect(header.test('const h = `type: ${kind}`')).toBe(true)
    expect(joined.test("['---', 'type: x', '---'].join('\\n')")).toBe(true)
  })
})
