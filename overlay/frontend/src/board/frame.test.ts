import { expect, test } from 'vitest'
import { render } from 'svelte/server'
import { act, toggleAction, type BoardApi } from './actions'
import { boardContext, type BoardCtx } from './context'
import { frameOf, keyAction } from './keys'
import { HeightDrag, ROW_PX } from './layout'
import type { Note } from './model'
import NoteFrame from './NoteFrame.svelte'
import { look } from '../lib/doodle/look.svelte'
import { THEMES } from '../lib/theme/palettes'

look.palette = THEMES[0]

const base = { tab: '', icon: '', open: true, rows: 0, color: 'yellow', pinned: false, version: 'v1', summary: '', modTime: '2026-10-28T10:00:00Z', linked: false, size: 'half' }

const ctx = (over: Partial<BoardCtx> = {}) => {
  const c: BoardCtx = {
    find: () => null, navigate: () => {}, act: async () => {}, arrange: async () => {}, zoom: () => {}, back: () => {},
    canBack: () => false, flip: () => {}, pageOf: () => 0, changed: () => false, seen: () => {}, filter: () => null,
    now: () => new Date('2026-10-28T12:00:00Z'), ...over,
  }
  return new Map<unknown, unknown>([[boardContext, c]])
}

// a stand-in for an element inside the DOM: closest() walks its ancestors
type El = { attrs: Record<string, string>; parent: El | null }
const el = (attrs: Record<string, string>, parent: El | null = null): El => ({ attrs, parent })
const asDom = (e: El): Element =>
  ({
    closest(sel: string) {
      const attr = sel.slice(1, -1).replace(/^data-/, '')
      for (let x: El | null = e; x; x = x.parent) if (attr in x.attrs) return { dataset: { [attr]: x.attrs[attr] } }
      return null
    },
  }) as unknown as Element

test('keys act on the frame around a focused link, not on the note it links to', () => {
  const frame = el({ frame: 'host.md' })
  const link = el({ note: 'plan' }, el({}, frame))
  expect(frameOf(asDom(link))).toBe('host.md')
  expect(frameOf(asDom(el({ note: 'plan' })))).toBeNull()
  expect(frameOf(null)).toBeNull()
  const ctx = { zoomed: false, canBack: false }
  expect(keyAction({ key: '-' }, ctx)).toEqual({ type: 'size', dir: -1 })
  expect(keyAction({ key: '=' }, ctx)).toEqual({ type: 'size', dir: 1 })
  expect(keyAction({ key: '[' }, ctx)).toEqual({ type: 'page', delta: -1 })
  expect(keyAction({ key: ']' }, ctx)).toEqual({ type: 'page', delta: 1 })
})

test('a frame carries data-frame; an embed link inside it carries only data-note', () => {
  const host = { ...base, name: 'host.md', title: 'Host', kind: 'note', data: { markdown: '![[plan]]' } } as unknown as Note
  const plan = { ...base, name: 'plan.md', title: 'Plan', kind: 'note', data: { markdown: 'p' } } as unknown as Note
  const { body } = render(NoteFrame, { props: { note: host }, context: ctx({ find: (l) => (l === 'plan' ? plan : null) }) })
  expect(body.match(/data-frame="/g)).toHaveLength(1)
  expect(body).toContain('data-frame="host.md"')
  expect(body).toContain('data-note="plan"')
  expect(body).not.toMatch(/data-note="host.md"/)
})

test('left and right turn the pages of a book, and move focus otherwise', () => {
  expect(keyAction({ key: 'ArrowRight' }, { zoomed: true, canBack: true, book: true })).toEqual({ type: 'page', delta: 1 })
  expect(keyAction({ key: 'ArrowLeft' }, { zoomed: false, canBack: false, book: true })).toEqual({ type: 'page', delta: -1 })
  expect(keyAction({ key: 'ArrowDown' }, { zoomed: false, canBack: false, book: true })).toEqual({ type: 'focus', delta: 1 })
  expect(keyAction({ key: 'ArrowRight' }, { zoomed: false, canBack: false, book: false })).toEqual({ type: 'focus', delta: 1 })
})

test('a note that cannot be read offers no rename, size, pin or colour', () => {
  const bad = { ...base, name: 'bad.md', title: 'Bad', kind: 'note', version: '', data: null, error: 'invalid front matter' } as unknown as Note
  const { body } = render(NoteFrame, { props: { note: bad }, context: ctx() })
  expect(body).toContain('invalid front matter')
  expect(body).not.toMatch(/aria-label="(Narrower|Wider|Width [^"]*|Pin first|Unpin|Colour)"/)
  expect(body).not.toContain('Double-click to rename')
  expect(body).not.toMatch(/class="edge\b/)
  expect(body).toMatch(/aria-label="Close"/) // it can still be put away
})

test('a note says its width as a fraction of the board; old names and cards read as one', () => {
  const at = (size: string, width: number) =>
    render(NoteFrame, { props: { note: { ...base, size, name: 'a.md', title: 'A', kind: 'note', data: { markdown: '' } } as unknown as Note, width }, context: ctx() }).body
  expect(at('1/3', 1200)).toMatch(/aria-label="Width 1\/3"/)
  expect(at('half', 1200)).toMatch(/aria-label="Width 1\/2"/)
  expect(at('page', 1200)).toMatch(/aria-label="Width 1"/)
  expect(at('card', 960)).toMatch(/aria-label="Width 1\/3"/) // the narrowest a card fits in
  expect(at('1/2', 1200)).toMatch(/aria-label="Narrower"/)
  expect(at('1/2', 1200)).toMatch(/class="edge side\b/) // the right edge drags the width
})

test('the bottom edge gives rows on release, nothing when cancelled or unchanged', () => {
  const d = new HeightDrag(ROW_PX * 5, 100, 5)
  expect(d.move(100 + ROW_PX * 3)).toBe(ROW_PX * 8)
  expect(d.release()).toBe(8)
  const same = new HeightDrag(ROW_PX * 5, 100, 5)
  same.move(103)
  expect(same.release()).toBeNull()
  const gone = new HeightDrag(ROW_PX * 5, 100, 5)
  gone.move(400)
  expect(gone.cancel()).toBeNull()
  expect(gone.release()).toBeNull() // a cancelled drag never saves
  const up = new HeightDrag(ROW_PX * 5, 100, 0)
  expect(up.move(-1000)).toBe(0)
  expect(up.release()).toBe(3) // never shorter than three rows
})

test('ticking an item sends note.act toggle with its place and the version', async () => {
  const sent: unknown[] = []
  const api = { act: async (...a: unknown[]) => void sent.push(a) } as unknown as BoardApi
  const note = { ...base, name: 'plan.md', title: 'Plan', kind: 'checklist', data: null } as unknown as Note
  let refreshed = 0
  await act(api, note, toggleAction(2), () => {}, async () => void refreshed++)
  expect(sent).toEqual([['plan.md', 'v1', { name: 'toggle', args: { i: '2' } }]])
  expect(refreshed).toBe(1)
})

test('a failed act is told, and the board is read again', async () => {
  const api = { act: async () => Promise.reject(new Error('stale')) } as unknown as BoardApi
  const note = { ...base, name: 'plan.md', title: 'Plan', kind: 'checklist', data: null } as unknown as Note
  const told: string[] = []
  let refreshed = 0
  await expect(act(api, note, toggleAction(0), (m) => told.push(m), async () => void refreshed++)).resolves.toBeUndefined()
  expect(told).toEqual(['Error: stale'])
  expect(refreshed).toBe(1)
})
