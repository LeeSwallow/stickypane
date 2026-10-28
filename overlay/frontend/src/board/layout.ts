// Where notes go on the grid (03 §3.6) and what a size or a dragged edge
// comes to (03 §3.7). Pure: the Grid component draws what place() says, so
// another layout (a free canvas) can stand in for it.
//
// A note's size is a fraction of the board's width, as on the terminal
// (internal/widget/size.go): 1/4 1/3 1/2 2/3 3/4 1. The old names are
// still read: page is 1, half is 1/2, and card is the narrowest fraction a
// card fits in. The grid has 12 columns, so every fraction is whole.

import { FRACTIONS, isFraction, type Fraction } from '../lib/sizes'
import { SPACE } from '../lib/theme/scale'

export { FRACTIONS, type Fraction }

/** GRID is the number of columns of a wide window. */
export const GRID = 12

/** NARROW is the width below which the grid has one column. */
export const NARROW = 720

/** CARD_PX is the width of one card: what "card" asks for (the terminal's 36 cells). */
export const CARD_PX = 300

/** MIN_PX is the narrowest a note is drawn: a narrower fraction takes the next one (the terminal's 20 cells). */
export const MIN_PX = 240

/** ROW_PX is the height of one terminal row on the window (13px body text at 1.55, plus spacing). */
export const ROW_PX = 22

/** MIN_ROWS is the shortest a dragged note gets. */
export const MIN_ROWS = 3

export const columnsFor = (width: number): number => (width < NARROW ? 1 : GRID)

const ALIASES: Record<string, Fraction | 'card'> = { page: '1', half: '1/2', card: 'card' }

/** normalSize is a size as the board writes it (a fraction, or card), or null when it is not one. */
export function normalSize(s: string): Fraction | 'card' | null {
  const k = s.trim().toLowerCase()
  if (k in ALIASES) return ALIASES[k]
  return isFraction(k) ? k : null
}

/** share is a fraction of n, rounded down. */
function share(f: Fraction, n: number): number {
  const [num, den] = f.includes('/') ? f.split('/').map(Number) : [1, 1]
  return Math.floor((n * num) / den)
}

/** colsOf is how many of the 12 columns a fraction takes. */
export const colsOf = (f: Fraction): number => share(f, GRID)

/** step is where a size is among FRACTIONS on a board width px wide; a card is the narrowest it fits in, nonsense the whole width. */
function step(size: string, width: number): number {
  const s = normalSize(size)
  if (s === 'card') {
    const i = FRACTIONS.findIndex((f) => share(f, width) >= CARD_PX)
    return i < 0 ? FRACTIONS.length - 1 : i
  }
  return s ? FRACTIONS.indexOf(s) : FRACTIONS.length - 1
}

/** sizeShown is the fraction a note is at: what its header says. */
export const sizeShown = (size: string, width: number): Fraction => FRACTIONS[step(size, width)]

/** stepSize is the size one step narrower (-1, the `-` key) or wider (+1, `=`), stopping at the ends. */
export function stepSize(size: string, width: number, dir: -1 | 1): Fraction {
  return FRACTIONS[Math.max(0, Math.min(FRACTIONS.length - 1, step(size, width) + dir))]
}

/** drawnStep is the step a size is drawn at: its own, or the next one that is not narrower than MIN_PX. */
function drawnStep(size: string, width: number): number {
  for (let i = step(size, width); i < FRACTIONS.length; i++) if (share(FRACTIONS[i], width) >= MIN_PX) return i
  return FRACTIONS.length - 1
}

/** spanOf is how many of the 12 columns a note of that size asks for on a board width px wide. */
export const spanOf = (size: string, width: number): number => colsOf(FRACTIONS[drawnStep(size, width)])

/** COL_GAP is the gap between the grid's columns in px (space-4; Grid.svelte's column gap). */
export const COL_GAP = SPACE[3]

/** spanPx is how wide span columns are drawn on a grid width px wide: the columns and the gaps between them. */
export function spanPx(span: number, width: number, gap = COL_GAP): number {
  const col = (width - (GRID - 1) * gap) / GRID
  return span * col + (span - 1) * gap
}

/**
 * snapSize is the size a dragged right edge lands on: the step from was
 * at, until the width (px) passes the middle between it and the next step
 * either way, so an edge moved a little does not change the size. A step's
 * width counts the grid's gaps, as it is drawn.
 */
export function snapSize(from: string, px: number, width: number): Fraction {
  const w = (i: number) => spanPx(spanOf(FRACTIONS[i], width), width)
  let i = step(from, width)
  while (i < FRACTIONS.length - 1 && 2 * px > w(i) + w(i + 1)) i++
  while (i > 0 && 2 * px < w(i - 1) + w(i)) i--
  return FRACTIONS[i]
}

/** A cell is a place across the grid: its row, where it starts and how wide it is (0-based). */
export interface Cell {
  row: number
  x: number
  w: number
}

/**
 * rows puts items of the given widths side by side, in order, starting a
 * new row when the next one does not fit, and stretches each row to the
 * full width, every item growing in proportion to its width. The columns
 * rounding leaves over go one each to the items that lost the most to it,
 * the first of equals first. A port of internal/layout.Rows.
 */
export function rows(total: number, widths: number[]): Cell[] {
  const cells: Cell[] = []
  const stretch = (from: number) => {
    const row = cells.slice(from)
    const sum = row.reduce((n, c) => n + c.w, 0)
    if (!row.length || sum <= 0) return
    const lost = row.map((c) => (c.w * total) % sum)
    let left = total
    for (const c of row) {
      c.w = Math.floor((c.w * total) / sum)
      left -= c.w
    }
    for (; left > 0; left--) {
      let most = 0
      for (let i = 0; i < row.length; i++) if (lost[i] > lost[most]) most = i
      row[most].w++
      lost[most] = -1
    }
    let x = 0
    for (const c of row) {
      c.x = x
      x += c.w
    }
  }
  let row = 0
  let start = 0
  let used = 0
  for (const raw of widths) {
    const w = Math.max(Math.min(raw, total), 1)
    if (used > 0 && used + w > total) {
      stretch(start)
      row++
      start = cells.length
      used = 0
    }
    cells.push({ row, x: 0, w })
    used += w
  }
  stretch(start)
  return cells
}

export interface Placement {
  name: string
  /** 1-based grid column and row */
  column: number
  row: number
  /** how many columns it spans */
  span: number
  /** drawn narrow (four columns or fewer, after its row is stretched): its shape draws small */
  small: boolean
}

/**
 * place puts the notes, in the order given (the terminal's: pinned first,
 * then the user's order, then by name), on a board width px wide: side by
 * side on 12 columns, each row filled. Nothing is moved up to fill a gap,
 * so the order on the screen is the order of the board. Below NARROW every
 * note has a row of its own.
 */
export function place(notes: { name: string; size: string }[], width: number): Placement[] {
  if (columnsFor(width) === 1) return notes.map((n, i) => ({ name: n.name, column: 1, row: i + 1, span: 1, small: false }))
  const asked = notes.map((n) => spanOf(n.size, width))
  return rows(GRID, asked).map((c, i) => ({ name: notes[i].name, column: c.x + 1, row: c.row + 1, span: c.w, small: c.w <= 4 }))
}

/**
 * WidthDrag is a drag of a note's right edge. It starts from the width the
 * note asks for (its fraction, gaps counted), not the width its row
 * stretched it to, so a nudge does not jump a lone note to another size.
 * landing is the fraction a release would save; release gives it, or null
 * when it is the note's own or the drag was cancelled.
 */
export class WidthDrag {
  readonly start: number
  private from: Fraction
  private landed: Fraction
  private done = false
  constructor(
    size: string,
    private readonly width: number,
    private readonly x0: number,
  ) {
    this.from = sizeShown(size, width)
    this.landed = this.from
    this.start = spanPx(spanOf(size, width), width)
  }
  /** move is the width to draw while the pointer is at x, and where it would land */
  move(x: number): { px: number; landing: Fraction } {
    const px = Math.max(0, this.start + x - this.x0)
    this.landed = snapSize(this.from, px, this.width)
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
