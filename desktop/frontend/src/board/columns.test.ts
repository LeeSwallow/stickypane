import { describe, expect, test } from 'vitest'
import { columnSize, columnSpan, columnSteps } from '../lib/sizes'
import { beside, ColumnDrag, columnsOf, dropAt, fitColumns, moveTo, placeColumns, sizeOnColumns, stepColumns, swap, TRACKS, type Slot } from './columns'
import { spanPx, COL_GAP } from './layout'

// the table the terminal's widget.ColumnSpan is tested against
import raw from '../../../../internal/core/widget/testdata/columns.json?raw'

const golden = JSON.parse(raw) as {
  spans: Record<string, number[]>
  sizes: Record<string, string[]>
}

const n = (name: string, size = '1/2', sized = false) => ({ name, size, sized })
const at = (p: { column: number; row: number; span: number }) => [p.row, p.column, p.span]

describe('a size spans its share of the columns, as on the terminal', () => {
  test('the shared table', () => {
    for (const [size, spans] of Object.entries(golden.spans)) spans.forEach((want, i) => expect(columnSpan(size, i + 1), `${size} of ${i + 1}`).toBe(want))
    for (const [cols, sizes] of Object.entries(golden.sizes)) {
      sizes.forEach((want, k) => expect(columnSize(k + 1, Number(cols))).toBe(want))
      expect(columnSteps(Number(cols))).toEqual(sizes)
    }
  })
  test('the setting: 1 to 4, auto, and 3 for anything else', () => {
    expect([columnsOf('1'), columnsOf('4'), columnsOf('auto'), columnsOf(''), columnsOf('7'), columnsOf(undefined)]).toEqual([1, 4, 0, 3, 3, 3])
  })
  test('a narrow window draws fewer columns, never none', () => {
    expect(fitColumns(4, 1200)).toBe(4)
    expect(fitColumns(4, 1000)).toBe(3)
    expect(fitColumns(3, 500)).toBe(2)
    expect(fitColumns(3, 400)).toBe(1)
    expect(fitColumns(3, 100)).toBe(1)
  })
})

describe('the grid of columns', () => {
  test('a note without a size takes a column; a row that is not full is shared equally', () => {
    const p = placeColumns([n('a'), n('b'), n('c'), n('d'), n('e')], 3)
    expect(p.map(at)).toEqual([[1, 1, 8], [1, 9, 8], [1, 17, 8], [2, 1, 12], [2, 13, 12]])
    expect(p[0].small).toBe(true)
    expect(p[3].small).toBe(false)
  })
  test('a sized note keeps its columns and the others share the rest equally', () => {
    const p = placeColumns([n('a', '1/4', true), n('b'), n('c'), n('d', '1', true), n('e', '1/2', true)], 4)
    expect(p.map(at)).toEqual([[1, 1, 6], [1, 7, 9], [1, 16, 9], [2, 1, 24], [3, 1, 12]])
  })
  test('the header says k/n and the steps go a column at a time', () => {
    expect(sizeOnColumns(n('a', '1/2'), 3)).toBe('1/3')
    expect(sizeOnColumns(n('a', '1/2', true), 3)).toBe('2/3')
    expect(stepColumns(n('a'), 3, 1)).toBe('2/3')
    expect(stepColumns(n('a'), 3, -1)).toBeNull()
    expect(stepColumns(n('a', '1', true), 4, -1)).toBe('3/4')
    expect(stepColumns(n('a', '1', true), 4, 1)).toBeNull()
  })
  test('a dragged edge lands on the nearest column', () => {
    const width = 1200
    const col = spanPx(TRACKS / 3, width, COL_GAP, TRACKS)
    const d = new ColumnDrag(n('a'), 3, width, 100)
    expect(d.start).toBeCloseTo(col)
    expect(d.move(100 + col * 0.4).landing).toBe('1/3')
    expect(d.move(100 + col * 0.6 + COL_GAP).landing).toBe('2/3')
    expect(d.move(100 + 5 * col).landing).toBe('1')
    expect(d.release()).toBe('1')
    expect(new ColumnDrag(n('a'), 3, width, 0).release()).toBeNull()
  })
})

describe('moving a note', () => {
  const placed = placeColumns(['a', 'b', 'c', 'd', 'e'].map((x) => n(x)), 3) // a b c / d e
  test('K and J trade places with the note at the same place a row up or down', () => {
    expect(beside(placed, 'b', 1)).toBe('e')
    expect(beside(placed, 'c', 1)).toBe('e')
    expect(beside(placed, 'e', -1)).toBe('b')
    expect(beside(placed, 'a', -1)).toBeNull()
    expect(beside(placed, 'd', 1)).toBeNull()
    const order = ['a', 'b', 'c', 'd', 'e']
    expect(swap(order, 'b', 'e')).toEqual(['a', 'e', 'c', 'd', 'b'])
    expect(swap(order, 'b', 'b')).toBeNull()
  })
  test('a dragged note goes before or after the note under the pointer', () => {
    const slots: Slot[] = [
      { name: 'a', left: 0, top: 0, right: 100, bottom: 50 },
      { name: 'b', left: 110, top: 0, right: 210, bottom: 80 },
      { name: 'c', left: 0, top: 100, right: 100, bottom: 150 },
    ]
    expect(dropAt(slots, 'c', 20, 10)).toEqual({ target: 'a', after: false })
    expect(dropAt(slots, 'c', 180, 10)).toEqual({ target: 'b', after: true })
    expect(dropAt(slots, 'c', 300, 10)).toEqual({ target: 'b', after: true })
    expect(dropAt(slots, 'c', 50, 120)).toBeNull()
    expect(dropAt(slots, 'c', 50, 90)).toBeNull()
    expect(moveTo(['a', 'b', 'c'], 'c', 'a', false)).toEqual(['c', 'a', 'b'])
    expect(moveTo(['a', 'b', 'c'], 'a', 'b', true)).toEqual(['b', 'a', 'c'])
    expect(moveTo(['a', 'b', 'c'], 'a', 'a', true)).toEqual(['a', 'b', 'c'])
  })
})
