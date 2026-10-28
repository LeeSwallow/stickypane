// Where notes go on the two-column grid (03 §3.6) and what a size or a
// dragged height comes to (03 §3.7). Pure: the Grid component draws what
// place() says, so another layout (a free canvas) can stand in for it.

import type { Size } from './model'

export const SIZES: Size[] = ['card', 'half', 'page']

/** NARROW is the width below which the grid has one column. */
export const NARROW = 720

/** ROW_PX is the height of one terminal row on the window (13px body text at 1.55, plus spacing). */
export const ROW_PX = 22

/** MIN_ROWS is the shortest a dragged note gets. */
export const MIN_ROWS = 3

export const columnsFor = (width: number): number => (width < NARROW ? 1 : 2)

const sizeOf = (s: string): Size => (SIZES.includes(s as Size) ? (s as Size) : 'half')

/** nextSize is the size one step smaller (-1, the `-` key) or bigger (+1, `=`), stopping at the ends. */
export function nextSize(size: string, dir: -1 | 1): Size {
  const i = SIZES.indexOf(sizeOf(size)) + dir
  return SIZES[Math.max(0, Math.min(SIZES.length - 1, i))]
}

export interface Placement {
  name: string
  /** 1-based grid column and row */
  column: number
  row: number
  /** how many columns it spans */
  span: number
  /** a card: drawn small */
  small: boolean
}

/**
 * place puts the notes, in the order given (the terminal's: pinned first,
 * then the user's order, then by name), row by row: a page spans every
 * column, a half or a card one. Nothing is moved up to fill a gap, so the
 * order on the screen is the order of the board.
 */
export function place(notes: { name: string; size: string }[], columns: number): Placement[] {
  const out: Placement[] = []
  let row = 1
  let col = 1
  for (const n of notes) {
    const size = sizeOf(n.size)
    const span = size === 'page' ? columns : 1
    if (col + span - 1 > columns) {
      row++
      col = 1
    }
    out.push({ name: n.name, column: col, row, span, small: size === 'card' })
    col += span
    if (col > columns) {
      row++
      col = 1
    }
  }
  return out
}

/** rowsFromHeight is the rows a dragged body height (px) comes to. */
export const rowsFromHeight = (px: number): number => Math.max(MIN_ROWS, Math.round(px / ROW_PX))

/** heightOfRows is the body height of a note of that many rows; null (as tall as its content) for 0. */
export const heightOfRows = (rows: number): number | null => (rows > 0 ? rows * ROW_PX : null)

/** shift is a tab's order with one note moved by delta (the { } keys), or null when it cannot move. */
export function shift(order: string[], name: string, delta: -1 | 1): string[] | null {
  const i = order.indexOf(name)
  const j = i + delta
  if (i < 0 || j < 0 || j >= order.length) return null
  const out = [...order]
  ;[out[i], out[j]] = [out[j], out[i]]
  return out
}

/**
 * HeightDrag is a drag of a note's bottom edge: from the body's height and
 * the pointer's y at the start to the rows to save on release. A release
 * that comes to the same rows, or a cancelled drag, saves nothing.
 */
export class HeightDrag {
  private height: number
  private done = false
  constructor(
    private readonly start: number,
    private readonly y0: number,
    private readonly rows: number,
  ) {
    this.height = start
  }
  /** move is the height to draw while the pointer is at y */
  move(y: number): number {
    this.height = Math.max(0, this.start + y - this.y0)
    return this.height
  }
  /** release is the rows to save, or null for none */
  release(): number | null {
    if (this.done) return null
    this.done = true
    const rows = rowsFromHeight(this.height)
    return rows === this.rows ? null : rows
  }
  cancel(): null {
    this.done = true
    return null
  }
}
