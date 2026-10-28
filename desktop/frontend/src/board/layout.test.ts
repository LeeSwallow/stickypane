import { describe, expect, test } from 'vitest'
import {
  colsOf, columnsFor, FRACTIONS, heightOfRows, normalSize, place, ROW_PX, rowsFromHeight, shift, sizeShown, snapSize, spanOf, spanPx, stepSize, WidthDrag,
} from './layout'
import { withArrange, type Board, type Note } from './model'
import { setSize, type BoardApi } from './actions'

const note = (name: string, size = '1/2', rows = 0): Note =>
  ({ name, tab: '', title: name, kind: 'note', icon: '', open: true, size, rows, color: 'yellow', pinned: false,
     version: 'v', summary: '', data: { markdown: '' }, modTime: '2026-10-28T10:00:00Z', linked: false }) as Note

describe('sizes are fractions of the board, as on the terminal (03 §3.6–3.7)', () => {
  test('each fraction takes its share of the 12 columns', () => {
    expect(FRACTIONS.map((f) => colsOf(f))).toEqual([3, 4, 6, 8, 9, 12])
  })
  test('old names are read as fractions; nonsense is the whole width', () => {
    expect(normalSize('page')).toBe('1')
    expect(normalSize('half')).toBe('1/2')
    expect(normalSize(' 1/3 ')).toBe('1/3')
    expect(normalSize('Card')).toBe('card')
    expect(normalSize('2/5')).toBeNull()
    expect(sizeShown('huge', 1200)).toBe('1')
    expect(sizeShown('', 1200)).toBe('1')
  })
  test('a card is the narrowest fraction a card (300px) fits in', () => {
    expect(sizeShown('card', 1200)).toBe('1/4') // 300
    expect(sizeShown('card', 960)).toBe('1/3') // 240 < 300 <= 320
    expect(sizeShown('card', 500)).toBe('2/3')
    expect(sizeShown('card', 200)).toBe('1')
  })
  test('a fraction narrower than 240px takes the next one that is not', () => {
    expect(spanOf('1/4', 1200)).toBe(3)
    expect(spanOf('1/4', 960)).toBe(3) // 240: just enough
    expect(spanOf('1/4', 800)).toBe(4) // 200 → 1/3 (266)
    expect(spanOf('1/2', 400)).toBe(8) // 200 → 2/3 (266)
    expect(spanOf('1/4', 100)).toBe(12)
  })
  test('- and = step through the fractions and stop at the ends', () => {
    expect(stepSize('1/4', 1200, 1)).toBe('1/3')
    expect(stepSize('1/2', 1200, 1)).toBe('2/3')
    expect(stepSize('3/4', 1200, 1)).toBe('1')
    expect(stepSize('1', 1200, 1)).toBe('1')
    expect(stepSize('1', 1200, -1)).toBe('3/4')
    expect(stepSize('1/4', 1200, -1)).toBe('1/4')
    expect(stepSize('half', 1200, 1)).toBe('2/3')
    expect(stepSize('page', 1200, -1)).toBe('3/4')
    expect(stepSize('card', 960, 1)).toBe('1/2') // a card is 1/3 here
    expect(stepSize('card', 960, -1)).toBe('1/4')
    expect(stepSize('', 1200, -1)).toBe('3/4') // unknown is the whole width
  })
  test('a span is its columns and the gaps between them', () => {
    // 1200px, 12 columns, 11 gaps of 16px: a column is (1200 - 176) / 12
    expect(spanPx(12, 1200)).toBe(1200)
    expect(spanPx(6, 1200)).toBeCloseTo(592)
    expect(spanPx(3, 1200)).toBeCloseTo(288)
    expect(spanPx(1, 1200)).toBeCloseTo(1024 / 12)
  })
  test('a dragged right edge changes the size only past the middle of two steps', () => {
    // on 1200px, gaps counted, the steps are 288 389 592 795 896 1200
    expect(snapSize('1/2', 592, 1200)).toBe('1/2')
    expect(snapSize('1/2', 492, 1200)).toBe('1/2') // the middle of 1/3 and 1/2 is 490.7
    expect(snapSize('1/2', 489, 1200)).toBe('1/3')
    expect(snapSize('1/2', 694, 1200)).toBe('2/3') // the middle of 1/2 and 2/3 is 693.3
    expect(snapSize('1/2', 692, 1200)).toBe('1/2')
    expect(snapSize('1/2', 5000, 1200)).toBe('1')
    expect(snapSize('1/2', 0, 1200)).toBe('1/4')
    expect(snapSize('card', 601, 1200)).toBe('1/2')
    expect(snapSize('1', 1049, 1200)).toBe('1') // the middle of 3/4 and 1 is 1048
    expect(snapSize('1', 1047, 1200)).toBe('3/4')
  })
  test('a width drag starts from the width the note asks for, not the width its row stretched it to', () => {
    // a lone 1/4 is drawn across the whole row, but asks for 288px
    expect(place([note('a', '1/4')], 1200)[0].span).toBe(12)
    const d = new WidthDrag('1/4', 1200, 1000)
    expect(d.start).toBeCloseTo(288)
    expect(d.move(1010).landing).toBe('1/4') // a nudge changes nothing
    expect(d.release()).toBeNull()
    const grow = new WidthDrag('1/4', 1200, 1000)
    expect(grow.move(1060).landing).toBe('1/3') // past 338.7, the middle of 1/4 and 1/3
    expect(grow.release()).toBe('1/3')
    const gone = new WidthDrag('1/4', 1200, 1000)
    gone.move(1600)
    expect(gone.cancel()).toBeNull()
    expect(gone.release()).toBeNull()
  })
})

describe('the grid places notes on 12 columns, in order, and fills each row (03 §3.6)', () => {
  test('notes sit side by side until the next does not fit; nothing is moved up', () => {
    const got = place([note('a', '1'), note('b', '1/2'), note('c', '1/4'), note('d', '1/2')], 1200)
    expect(got.map((p) => [p.name, p.column, p.row, p.span])).toEqual([
      ['a', 1, 1, 12],
      ['b', 1, 2, 8], // 6 + 3 stretched to 12 in proportion
      ['c', 9, 2, 4],
      ['d', 1, 3, 12], // alone on its row, it fills it
    ])
  })
  test('the old names place as their fractions', () => {
    const got = place([note('a', 'half'), note('b', 'page'), note('c', 'half'), note('d', 'half')], 1200)
    expect(got.map((p) => [p.name, p.column, p.row, p.span])).toEqual([
      ['a', 1, 1, 12],
      ['b', 1, 2, 12],
      ['c', 1, 3, 6],
      ['d', 7, 3, 6],
    ])
  })
  test('thirds and quarters share a row; rounding goes to who lost most', () => {
    const got = place([note('a', '1/3'), note('b', '1/3'), note('c', '1/3'), note('d', '1/4'), note('e', '1/4'), note('f', '1/3')], 1200)
    expect(got.map((p) => [p.name, p.column, p.row, p.span])).toEqual([
      ['a', 1, 1, 4],
      ['b', 5, 1, 4],
      ['c', 9, 1, 4],
      // 3 3 4 → 10, stretched to 3.6 3.6 4.8: the two columns left over go
      // to f (lost .8) and then to d (the first of the two that lost .6)
      ['d', 1, 2, 4],
      ['e', 5, 2, 3],
      ['f', 8, 2, 5],
    ])
  })
  test('one column below 720px: every note spans it, one per row', () => {
    expect(columnsFor(719)).toBe(1)
    expect(columnsFor(720)).toBe(12)
    const got = place([note('a', '1/4'), note('b', '1/2')], 600)
    expect(got.map((p) => [p.column, p.row, p.span])).toEqual([[1, 1, 1], [1, 2, 1]])
  })
  test('a note drawn narrow is marked small; one its row stretched wide is not', () => {
    const four = place([note('a', 'card'), note('b', '1/4'), note('c', '1/4'), note('d', '1/4')], 1200)
    expect(four.map((p) => p.small)).toEqual([true, true, true, true])
    expect(place([note('a', '1/4')], 1200)[0].small).toBe(false) // alone, it fills the row
    expect(place([note('a', '1/2'), note('b', '1/2')], 1200)[0].small).toBe(false)
  })
})

describe('rows', () => {
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
    tabs: [{ name: '', title: '', notes: [note('a', '1/2'), note('b', '1/2')] }],
    settings: {} as Board['settings'],
    palette: {},
  }
  const sent: unknown[] = []
  const api = { arrange: async (p: unknown) => void sent.push(p) } as unknown as BoardApi
  let shown = board
  expect(place(board.tabs[0].notes, 1200).map((p) => [p.name, p.span, p.row])).toEqual([['a', 6, 1], ['b', 6, 1]])
  await setSize(api, 'a', '1', (patch) => (shown = patch(shown)))
  expect(sent).toEqual([{ note: 'a', size: '1' }])
  // the window redraws at once, before the bridge's board.changed
  expect(place(shown.tabs[0].notes, 1200).map((p) => [p.name, p.span, p.row])).toEqual([['a', 12, 1], ['b', 12, 2]])
  expect(board.tabs[0].notes[0].size).toBe('1/2') // the old board is not changed in place
  expect(withArrange(board, 'b', { rows: 7 }).tabs[0].notes[1].rows).toBe(7)
})

test('{ and } move a note one place in its tab, and not past the ends', () => {
  expect(shift(['a', 'b', 'c'], 'b', -1)).toEqual(['b', 'a', 'c'])
  expect(shift(['a', 'b', 'c'], 'b', 1)).toEqual(['a', 'c', 'b'])
  expect(shift(['a', 'b', 'c'], 'a', -1)).toBeNull()
  expect(shift(['a', 'b', 'c'], 'c', 1)).toBeNull()
  expect(shift(['a'], 'x', 1)).toBeNull()
})
