import { describe, expect, test } from 'vitest'
import { columnsFor, heightOfRows, nextSize, place, ROW_PX, rowsFromHeight, shift } from './layout'
import { withArrange, type Board, type Note } from './model'
import { setSize, type BoardApi } from './actions'

const note = (name: string, size = 'half', rows = 0): Note =>
  ({ name, tab: '', title: name, kind: 'note', icon: '', open: true, size, rows, color: 'yellow', pinned: false,
     version: 'v', summary: '', data: { markdown: '' }, modTime: '2026-10-28T10:00:00Z', linked: false }) as Note

describe('the grid places notes as 03 §3.6 says', () => {
  test('page takes two columns, half and card one, in the order given', () => {
    const got = place([note('a', 'page'), note('b', 'half'), note('c', 'card'), note('d', 'half')], 2)
    expect(got.map((p) => [p.name, p.column, p.row, p.span])).toEqual([
      ['a', 1, 1, 2],
      ['b', 1, 2, 1],
      ['c', 2, 2, 1],
      ['d', 1, 3, 1],
    ])
  })
  test('a page after a lone half starts its own row; the order is never changed', () => {
    const got = place([note('a', 'half'), note('b', 'page'), note('c', 'half')], 2)
    expect(got.map((p) => [p.name, p.column, p.row, p.span])).toEqual([
      ['a', 1, 1, 1],
      ['b', 1, 2, 2],
      ['c', 1, 3, 1],
    ])
  })
  test('one column below 720px: every note spans one', () => {
    expect(columnsFor(719)).toBe(1)
    expect(columnsFor(720)).toBe(2)
    const got = place([note('a', 'page'), note('b', 'half')], 1)
    expect(got.map((p) => [p.column, p.row, p.span])).toEqual([[1, 1, 1], [1, 2, 1]])
  })
  test('a card is marked small', () => {
    expect(place([note('a', 'card')], 2)[0].small).toBe(true)
    expect(place([note('a', 'half')], 2)[0].small).toBe(false)
  })
})

describe('sizes and rows', () => {
  test('- and = step through card, half, page and stop at the ends', () => {
    expect(nextSize('half', -1)).toBe('card')
    expect(nextSize('card', -1)).toBe('card')
    expect(nextSize('half', 1)).toBe('page')
    expect(nextSize('page', 1)).toBe('page')
    expect(nextSize('', 1)).toBe('page') // unknown reads as half
  })
  test('a dragged height becomes rows, and rows a height', () => {
    expect(rowsFromHeight(ROW_PX * 12)).toBe(12)
    expect(rowsFromHeight(ROW_PX * 12 + ROW_PX * 0.4)).toBe(12)
    expect(rowsFromHeight(ROW_PX * 12 + ROW_PX * 0.6)).toBe(13)
    expect(rowsFromHeight(5)).toBe(3) // never shorter than three rows
    expect(heightOfRows(10)).toBe(ROW_PX * 10)
    expect(heightOfRows(0)).toBeNull() // 0: as tall as the content
  })
})

test('size button sends note.arrange and the grid reflows', async () => {
  const board: Board = {
    title: 't',
    tabs: [{ name: '', title: '', notes: [note('a', 'half'), note('b', 'half')] }],
    settings: {} as Board['settings'],
    palette: {},
  }
  const sent: unknown[] = []
  const api = { arrange: async (p: unknown) => void sent.push(p) } as unknown as BoardApi
  let shown = board
  await setSize(api, 'a', 'page', (patch) => (shown = patch(shown)))
  expect(sent).toEqual([{ note: 'a', size: 'page' }])
  // the window redraws at once, before the bridge's board.changed
  expect(place(shown.tabs[0].notes, 2).map((p) => [p.name, p.span, p.row])).toEqual([['a', 2, 1], ['b', 1, 2]])
  expect(board.tabs[0].notes[0].size).toBe('half') // the old board is not changed in place
  expect(withArrange(board, 'b', { rows: 7 }).tabs[0].notes[1].rows).toBe(7)
})

test('{ and } move a note one place in its tab, and not past the ends', () => {
  expect(shift(['a', 'b', 'c'], 'b', -1)).toEqual(['b', 'a', 'c'])
  expect(shift(['a', 'b', 'c'], 'b', 1)).toEqual(['a', 'c', 'b'])
  expect(shift(['a', 'b', 'c'], 'a', -1)).toBeNull()
  expect(shift(['a', 'b', 'c'], 'c', 1)).toBeNull()
  expect(shift(['a'], 'x', 1)).toBeNull()
})
