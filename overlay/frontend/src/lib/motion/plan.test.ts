import { describe, expect, it } from 'vitest'
import type { Board, Note } from '../../board/model'
import { asChange, BATCH, DURATION, frameOf, leaving, planChange, type Change, type ChangeEvent } from './plan'

const note = (name: string, kind: string, data: unknown, extra: Partial<Note> = {}): Note => ({
  name, tab: '', title: name, kind, icon: '', open: true, size: '1/2', rows: 0, color: 'blue', pinned: false,
  version: 'v', summary: '', data, modTime: '', linked: false, ...extra,
})
const card = (text: string, who: string[] = [], stalled = false) => ({ text, who, detail: [], stalled })
const kanban = (cols: Record<string, ReturnType<typeof card>[]>) =>
  note('work.md', 'board', { desc: [], columns: Object.entries(cols).map(([title, cards]) => ({ title, cards })) })
const checklist = (items: [string, boolean][]) =>
  note('plan.md', 'checklist', { items: items.map(([text, checked]) => ({ text, checked })), prose: [], done: 0, total: items.length })
const log = (lines: string[]) => note('build.log', 'log', { lines })
const board = (...notes: Note[]): Board => ({
  title: 't', tabs: [{ name: '', title: '', notes }], palette: null,
  settings: { theme: '', style: 'doodle', view: '', language: { choice: 'en', resolved: 'en' }, tab: '', time: '', editor: '', stale: '' },
})
const ev = (type: string, n: string, extra: Partial<ChangeEvent> = {}): ChangeEvent => ({ note: n, type, ...extra })

describe('planChange: one change, one movement', () => {
  it('a card an agent moved is carried, in its owner\'s colour; the note does not glow as well', () => {
    const prev = board(kanban({ Todo: [card('ship', ['claude'])], Doing: [] }))
    const after = kanban({ Todo: [], Doing: [card('ship', ['claude'])] })
    const p = planChange(prev, board(after), { notes: [after], events: [ev('card.moved', 'work.md', { item: 'ship', from: 'Todo', to: 'Doing' })] })
    expect(p.steps).toEqual([{ target: { part: 'card', frame: 'work.md', card: 'ship' }, effect: 'carry', delay: 0, who: 'claude' }])
    expect(p.flip).toEqual(['work.md'])
  })

  it('the person\'s own move, already drawn, only lands (04 §4.5)', () => {
    const shown = kanban({ Todo: [], Doing: [card('ship')] })
    const p = planChange(board(shown), board(shown), { notes: [shown], events: [ev('card.moved', 'work.md', { item: 'ship', from: 'Todo', to: 'Doing' })] })
    expect(p.steps.map((s) => s.effect)).toEqual(['slap'])
  })

  it('a new card drops in; a card that went leaves', () => {
    const prev = board(kanban({ Todo: [card('old')] }))
    const after = kanban({ Todo: [card('new')] })
    const p = planChange(prev, board(after), {
      notes: [after],
      events: [ev('card.added', 'work.md', { item: 'new', to: 'Todo' }), ev('card.removed', 'work.md', { item: 'old', from: 'Todo' })],
    })
    expect(p.steps.map((s) => [s.effect, s.target.part === 'card' && s.target.card, s.delay])).toEqual([
      ['drop', 'new', 0],
      ['leave', 'old', BATCH.eachGap],
    ])
  })

  it('a card that has just stalled nudges once; one that was stalled already does not', () => {
    const prev = board(kanban({ Doing: [card('a'), card('b', [], true)] }))
    const after = kanban({ Doing: [card('a', [], true), card('b', [], true)] })
    const p = planChange(prev, board(after), { notes: [after], events: [] })
    expect(p.steps).toEqual([{ target: { part: 'card', frame: 'work.md', card: 'a' }, effect: 'nudge', delay: 0 }])
  })

  it('a ticked item settles; one the person ticked (already drawn) does not move again', () => {
    const prev = board(checklist([['a', false], ['b', true]]))
    const after = checklist([['a', true], ['b', true]])
    const change: Change = { notes: [after], events: [ev('item.ticked', 'plan.md', { item: 'a' })] }
    expect(planChange(prev, board(after), change).steps.map((s) => s.effect)).toEqual(['tick'])
    expect(planChange(board(after), board(after), change).steps).toEqual([])
  })

  it('a new item unfolds, an unticked one fades back', () => {
    const prev = board(checklist([['a', true]]))
    const after = checklist([['a', false], ['b', false]])
    const p = planChange(prev, board(after), { notes: [after], events: [ev('item.unticked', 'plan.md', { item: 'a' }), ev('item.added', 'plan.md', { item: 'b' })] })
    expect(p.steps.map((s) => s.effect)).toEqual(['untick', 'unfold'])
  })

  it('new log lines push in from below, together', () => {
    const prev = board(log(['one']))
    const after = log(['one', 'two', 'three'])
    const p = planChange(prev, board(after), { notes: [after], events: [ev('log.appended', 'build.log', { item: 'two' }), ev('log.appended', 'build.log', { item: 'three' })] })
    expect(p.steps).toEqual([{ target: { part: 'lines', frame: 'build.log', count: 2 }, effect: 'push', delay: 0 }])
  })

  it('a new note drops in, and its own events add nothing', () => {
    const fresh = checklist([['a', false]])
    const p = planChange(board(), board(fresh), { notes: [fresh], events: [ev('note.created', 'plan.md'), ev('item.added', 'plan.md', { item: 'a' })] })
    expect(p.steps).toEqual([{ target: { part: 'note', frame: 'plan.md' }, effect: 'drop', delay: 0 }])
    expect(p.flip).toEqual([])
  })

  it('a note changed otherwise glows once', () => {
    const prev = board(note('readme.md', 'note', { markdown: 'a' }))
    const after = note('readme.md', 'note', { markdown: 'b' })
    const p = planChange(prev, board(after), { notes: [after], events: [ev('note.changed', 'readme.md')] })
    expect(p.steps).toEqual([{ target: { part: 'note', frame: 'readme.md' }, effect: 'glow', delay: 0 }])
  })

  it('a note whose data the window already drew (the person\'s own guess) does not glow', () => {
    const n = note('readme.md', 'note', { markdown: 'a' })
    const p = planChange(board(n), board({ ...n, modTime: 'later', version: 'w' }), { notes: [{ ...n, modTime: 'later' }], events: [] })
    expect(p.steps).toEqual([])
  })

  it('a book\'s page moves in the book\'s frame', () => {
    const page = note('book/p1.md', 'note', { markdown: 'a' })
    const book = note('book', 'note', null, { pages: [page] })
    const after = { ...page, data: { markdown: 'b' } }
    const p = planChange(board(book), board({ ...book, pages: [after] }), { notes: [after], events: [] })
    expect(p.steps[0].target).toEqual({ part: 'note', frame: 'book' })
    expect(frameOf(board(book), 'book/p1.md')).toBe('book')
  })

  it('notes not on screen do not move', () => {
    const prev = board(kanban({ Todo: [card('ship')], Doing: [] }))
    const after = kanban({ Todo: [], Doing: [card('ship')] })
    const p = planChange(prev, board(after), { notes: [after], events: [ev('card.moved', 'work.md', { item: 'ship', from: 'Todo', to: 'Doing' })] }, { visible: new Set(['other.md']) })
    expect(p).toEqual({ steps: [], flip: [] })
  })
})

describe('planChange: many at once (04 §4.4)', () => {
  it('up to three are played each in turn, 60 ms apart', () => {
    const prev = board(checklist([['a', false], ['b', false], ['c', false]]))
    const after = checklist([['a', true], ['b', true], ['c', true]])
    const p = planChange(prev, board(after), { notes: [after], events: ['a', 'b', 'c'].map((i) => ev('item.ticked', 'plan.md', { item: i })) })
    expect(p.steps.map((s) => s.delay)).toEqual([0, 60, 120])
  })

  it('more: only card moves, 30 ms apart, at most six, the rest only slide', () => {
    const names = ['a', 'b', 'c', 'd', 'e', 'f', 'g', 'h']
    const prev = board(kanban({ Todo: names.map((n) => card(n)), Doing: [] }))
    const after = kanban({ Todo: [], Doing: [...names.map((n) => card(n)), card('new')] })
    const events = [...names.map((n) => ev('card.moved', 'work.md', { item: n, from: 'Todo', to: 'Doing' })), ev('card.added', 'work.md', { item: 'new', to: 'Doing' })]
    const p = planChange(prev, board(after), { notes: [after], events })
    expect(p.steps.map((s) => s.effect)).toEqual(Array(BATCH.moves).fill('carry'))
    expect(p.steps.map((s) => s.delay)).toEqual([0, 30, 60, 90, 120, 150])
    expect(p.flip).toEqual(['work.md'])
  })

  it('nothing ends after the cap', () => {
    const many = Array.from({ length: 40 }, (_, i) => `c${i}`)
    const prev = board(kanban({ Todo: many.map((n) => card(n)), Doing: [] }))
    const after = kanban({ Todo: [], Doing: many.map((n) => card(n)) })
    const p = planChange(prev, board(after), { notes: [after], events: many.map((n) => ev('card.moved', 'work.md', { item: n, from: 'Todo', to: 'Doing' })) })
    for (const s of p.steps) expect(s.delay + DURATION[s.effect]).toBeLessThanOrEqual(BATCH.cap)
  })
})

describe('leaving and asChange', () => {
  it('copies the notes and cards that go, not a book that only lost a page', () => {
    const page = note('book/p1.md', 'note', null)
    const prev = board(note('gone.md', 'note', null), note('book', 'note', null, { pages: [page] }), kanban({ Todo: [card('x')] }))
    expect(
      leaving(prev, { removed: ['gone.md', 'book/p1.md'], events: [ev('card.removed', 'work.md', { item: 'x' })] }),
    ).toEqual([
      { part: 'note', frame: 'gone.md' },
      { part: 'card', frame: 'work.md', card: 'x' },
    ])
  })

  it('reads a notification\'s params and ignores what is not one', () => {
    expect(asChange(null)).toEqual({ notes: [], removed: [], events: [] })
    expect(asChange({ events: [{ type: 'item.ticked', note: 'a.md' }, { nope: 1 }], removed: ['b.md', 3] })).toEqual({
      notes: [],
      removed: ['b.md'],
      events: [{ type: 'item.ticked', note: 'a.md' }],
    })
  })
})
