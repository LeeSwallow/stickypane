// A board of columns (the `columns` setting, 1 to 4): the grid is that many
// equal columns, as internal/core/layout.Grid on the terminal. A note
// without a size of its own takes one column; a note with one spans
// columnSpan of them (lib/sizes.ts). The notes of a row share what it
// leaves equally, so a row that is not full has no gap. The grid is drawn
// on TRACKS tracks so every share is whole. Also here: moving a note in
// the order, by keys (a place, a row) and by a drag on the grid.

import { columnSize, columnSpan, MAX_COLUMNS, type Fraction } from '../lib/sizes'
import { COL_GAP, columnsFor, MIN_PX, rows, spanPx, type Placement } from './layout'

/** TRACKS is the number of tracks of a grid of columns: 24 holds 1–4 columns and the halves and thirds a row is shared into. */
export const TRACKS = 24

/** DEFAULT_COLUMNS is the board's columns when the setting says nothing. */
export const DEFAULT_COLUMNS = 3

type Sized = { size: string; sized?: boolean }

/** columnsOf reads the board's `columns` setting: 1–4, 0 for auto (the sizes decide, as before), the default for anything else. */
export function columnsOf(setting: string | undefined | null): number {
  if (setting === 'auto') return 0
  const n = Number(setting)
  return Number.isInteger(n) && n >= 1 && n <= MAX_COLUMNS ? n : DEFAULT_COLUMNS
}

/**
 * fitColumns is how many of the n chosen columns a grid width px wide
 * draws: n, or fewer when a column would be narrower than MIN_PX, never
 * under one. A safeguard for a narrow window, not a layout of its own:
 * nothing is written.
 */
export const fitColumns = (n: number, width: number): number => Math.max(1, Math.min(n, Math.floor((width + COL_GAP) / (MIN_PX + COL_GAP))))

/** spanIn is how many of n columns a note takes: one without a size of its own, its share with one. */
export const spanIn = (note: Sized, n: number): number => (note.sized ? columnSpan(note.size, n) : 1)

/** tracksFor is how many tracks the grid has: TRACKS on columns, else the 12 of auto (one below NARROW). */
export const tracksFor = (columns: number, width: number): number => (columns > 0 ? TRACKS : columnsFor(width))

/** placeColumns puts notes on n columns of TRACKS tracks, in order, starting a new row when the next does not fit. */
export function placeColumns(notes: ({ name: string } & Sized)[], n: number): Placement[] {
  n = Math.max(1, n)
  const per = TRACKS / n
  const out: Placement[] = []
  let start = 0
  let used = 0
  let row = 0
  const flush = (to: number) => {
    const part = notes.slice(start, to)
    const widths = part.map((x) => spanIn(x, n) * per)
    const sized = part.map((x) => x.sized === true)
    rows(TRACKS, widths, sized, false).forEach((c, i) =>
      out.push({ name: part[i].name, column: c.x + 1, row: row + 1, span: c.w, small: c.w <= TRACKS / 3 }),
    )
    row++
  }
  notes.forEach((note, i) => {
    const k = spanIn(note, n)
    if (used > 0 && used + k > n) {
      flush(i)
      start = i
      used = 0
    }
    used += k
  })
  if (start < notes.length) flush(notes.length)
  return out
}

/** sizeOnColumns is the fraction a note is at on n columns, as its header says it: k/n. */
export const sizeOnColumns = (note: Sized, n: number): Fraction => columnSize(spanIn(note, n), n)

/** stepColumns is the size one column wider (+1) or narrower (-1) on n columns, or null at either end. */
export function stepColumns(note: Sized, n: number, dir: -1 | 1): Fraction | null {
  const k = spanIn(note, n) + dir
  return k < 1 || k > n ? null : columnSize(k, n)
}

/**
 * ColumnDrag is a drag of a note's right edge on a board of columns: it
 * lands on the nearest whole number of columns, gaps counted. Like
 * WidthDrag it starts from the columns the note asks for, not the width
 * its row stretched it to.
 */
export class ColumnDrag {
  readonly start: number
  private readonly from: Fraction
  private landed: Fraction
  private done = false
  constructor(
    note: Sized,
    private readonly n: number,
    private readonly width: number,
    private readonly x0: number,
  ) {
    const k = spanIn(note, n)
    this.from = columnSize(k, n)
    this.landed = this.from
    this.start = spanPx((k * TRACKS) / n, width, COL_GAP, TRACKS)
  }
  /** move is the width to draw while the pointer is at x, and where it would land */
  move(x: number): { px: number; landing: Fraction } {
    const px = Math.max(0, this.start + x - this.x0)
    const col = spanPx(TRACKS / this.n, this.width, COL_GAP, TRACKS)
    this.landed = columnSize(Math.round((px + COL_GAP) / (col + COL_GAP)), this.n)
    return { px, landing: this.landed }
  }
  release(): Fraction | null {
    if (this.done) return null
    this.done = true
    return this.landed === this.from ? null : this.landed
  }
  cancel(): null {
    this.done = true
    return null
  }
}

/**
 * beside is the note delta rows from the one named that has its place in
 * its row (the k-th for the k-th, or the last of a shorter row), or null
 * when there is no such row: what K and J trade places with. As
 * internal/core/layout.Beside.
 */
export function beside(placed: Placement[], name: string, delta: -1 | 1): string | null {
  const i = placed.findIndex((p) => p.name === name)
  if (i < 0) return null
  const at = placed.filter((p, j) => j < i && p.row === placed[i].row).length
  const row = placed.filter((p) => p.row === placed[i].row + delta)
  if (!row.length) return null
  return row[Math.min(at, row.length - 1)].name
}

/** swap is an order with two notes trading places, or null when either is not in it. */
export function swap(order: string[], a: string, b: string): string[] | null {
  const i = order.indexOf(a)
  const j = order.indexOf(b)
  if (i < 0 || j < 0 || i === j) return null
  const out = [...order]
  ;[out[i], out[j]] = [out[j], out[i]]
  return out
}

/** moveTo is an order with one note taken out and put before (or after) another: where a dragged note is let go. */
export function moveTo(order: string[], name: string, target: string, after: boolean): string[] {
  if (name === target || !order.includes(name) || !order.includes(target)) return order
  const out = order.filter((n) => n !== name)
  out.splice(out.indexOf(target) + (after ? 1 : 0), 0, name)
  return out
}

/** A Slot is where a note is drawn, in the window's px. */
export interface Slot {
  name: string
  left: number
  top: number
  right: number
  bottom: number
}

/**
 * dropAt is where a note dragged to (x, y) goes: before or after the note
 * under the pointer, by the half it is in; past the end of a row, after
 * its last note; null over the dragged note itself or over nothing.
 */
export function dropAt(slots: Slot[], dragged: string, x: number, y: number): { target: string; after: boolean } | null {
  const under = slots.find((s) => x >= s.left && x < s.right && y >= s.top && y < s.bottom)
  if (under) return under.name === dragged ? null : { target: under.name, after: x >= (under.left + under.right) / 2 }
  const level = slots.filter((s) => y >= s.top && y < s.bottom)
  const last = level.reduce<Slot | null>((a, s) => (!a || s.right > a.right ? s : a), null)
  if (last && x >= last.right && last.name !== dragged) return { target: last.name, after: true }
  return null
}
