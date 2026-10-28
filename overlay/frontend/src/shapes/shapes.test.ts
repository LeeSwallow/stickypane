import { expect, test } from 'vitest'
import { render } from 'svelte/server'
import Kanban from './Kanban.svelte'
import Checklist from './Checklist.svelte'
import Grid from '../board/Grid.svelte'
import { boardContext, type BoardCtx } from '../board/context'
import { itemOrder, type Note } from '../board/model'
import { look } from '../lib/doodle/look.svelte'
import { THEMES } from '../lib/theme/palettes'

look.palette = THEMES[0]

const ctx = (over: Partial<BoardCtx> = {}): Map<unknown, unknown> => {
  const c: BoardCtx = {
    find: () => null,
    navigate: () => {},
    act: async () => {},
    arrange: async () => {},
    zoom: () => {},
    back: () => {},
    canBack: () => false,
    flip: () => {},
    pageOf: () => 0,
    changed: () => false,
    seen: () => {},
    filter: () => null,
    now: () => new Date('2026-10-28T12:00:00Z'),
    ...over,
  }
  return new Map([[boardContext, c]])
}

const base = { tab: '', icon: '', open: true, rows: 0, color: 'yellow', pinned: false, version: 'v', summary: '', modTime: '2026-10-28T10:00:00Z', linked: false }

test('board draws columns and stalled ring', () => {
  const note = {
    ...base, name: 'work.md', title: 'Work', kind: 'board', size: 'page',
    data: {
      desc: [],
      columns: [
        { title: 'Todo', cards: [{ text: 'plan', who: ['sam'], detail: [], stalled: false }] },
        { title: 'Doing', cards: [{ text: 'build grid', who: ['claude'], moved: '2026-10-28T08:00:00Z', detail: [], stalled: true }] },
        { title: 'Done', cards: [] },
      ],
    },
  } as unknown as Note
  const { body } = render(Kanban, { props: { note }, context: ctx() })
  const cols = body.match(/class="col[ "]/g) ?? []
  expect(cols).toHaveLength(3)
  expect(body).toContain('Todo')
  expect(body).toContain('Doing')
  expect(body).toMatch(/class="count[^"]*"[^>]*>1</)
  expect(body).toContain('@claude')
  expect((body.match(/class="ring/g) ?? []).length).toBe(1) // only the stalled card
  expect(body).toContain('data-stalled="true"')
})

test('a stalled card with no time says so without an empty suffix', () => {
  const note = {
    ...base, name: 'w.md', title: 'W', kind: 'board', size: 'page',
    data: { desc: [], columns: [{ title: 'Doing', cards: [{ text: 'x', who: [], detail: [], stalled: true }] }, { title: 'Done', cards: [] }] },
  } as unknown as Note
  const { body } = render(Kanban, { props: { note }, context: ctx() })
  expect(body).toMatch(/class="stalled[^"]*">Stalled</)
  expect(body).not.toMatch(/Stalled \s*</) // not "Stalled {time}" with no time
})

test('a checklist draws its items, done ones struck, with progress', () => {
  const note = {
    ...base, name: 'todo.md', title: 'Todo', kind: 'checklist', size: 'half',
    data: { items: [{ text: 'one', checked: true, done: '2026-10-28T09:00:00Z' }, { text: 'two', checked: false }], prose: [], done: 1, total: 2 },
  } as unknown as Note
  const { body } = render(Checklist, { props: { note }, context: ctx() })
  expect(body).toContain('one')
  expect(body).toMatch(/class="item[^"]*done/)
  expect(body).toMatch(/role="checkbox"[^>]*aria-checked="true"/)
  expect(body).toContain('1/2')
  // open items first, then the done ones, the most recent first (as the terminal)
  expect(body.indexOf('>two<')).toBeLessThan(body.indexOf('>one<'))
})

test('ticking sends toggle with the item\'s place in the file', () => {
  expect(itemOrder([
    { text: 'a', checked: true, done: '2026-10-28T09:00:00Z' },
    { text: 'b', checked: false },
    { text: 'c', checked: true, done: '2026-10-28T10:00:00Z' },
    { text: 'd', checked: false },
  ])).toEqual([1, 3, 2, 0])
})

test('300 notes render under 1s', () => {
  const notes = Array.from({ length: 300 }, (_, i) => ({
    ...base, name: `n${i}.md`, title: `Note ${i}`, kind: 'checklist', size: i % 5 === 0 ? '1' : '1/2',
    data: { items: [{ text: 'a', checked: false }, { text: 'b', checked: true }], prose: [], done: 1, total: 2 },
  })) as unknown as Note[]
  const t0 = performance.now()
  const { body } = render(Grid, { props: { notes, width: 1200 }, context: ctx() })
  const took = performance.now() - t0
  expect((body.match(/data-frame="/g) ?? []).length).toBe(300)
  expect(took).toBeLessThan(1000)
})
