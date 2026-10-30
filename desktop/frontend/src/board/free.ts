// The rules of the free canvas (03 §3.6.1): notes anywhere, by x, y, w and
// h, in the canvas's own px. Pure: Canvas.svelte draws what these say and
// the shell keeps the places (services/canvas.ts). A note the canvas has
// not seen starts where the grid would put it, and that place is then kept.

import type { Place } from '../services/canvas'
import { SPACE } from '../lib/theme/scale'
import { heightOfRows, place as gridPlace, spanPx, COL_GAP } from './layout'

/** SNAP is the canvas's grid: with snap on, a note lands on a multiple of it. */
export const SNAP = 8
/** The smallest a note gets on the canvas (the shell's canvas.MinW, MinH). */
export const MIN_W = 160
export const MIN_H = 96
/** PAD is the margin around the notes. */
export const PAD = SPACE[5]
/** ROW_GAP is the space between the grid's rows a new note keeps (Grid.svelte's row gap). */
export const ROW_GAP = SPACE[6]
/** NEW_H is the height a note new to the canvas starts at when it has no rows of its own. */
export const NEW_H = 280
/** The canvas zooms from half to double (⌘ + wheel). */
export const ZOOM_MIN = 0.5
export const ZOOM_MAX = 2

/** snapTo rounds v to the canvas grid, or to a whole px with snap off. */
export const snapTo = (v: number, on: boolean): number => (on ? Math.round(v / SNAP) * SNAP : Math.round(v))

/** tiltOf is a small lean, -1.5° to 1.5°, from the note's name: the same every time. */
export function tiltOf(name: string): number {
  let h = 0
  for (const c of name) h = (h * 31 + c.codePointAt(0)!) | 0
  return ((Math.abs(h) % 7) - 3) * 0.5
}

/** topZ is the highest place in the stack. */
export const topZ = (places: Record<string, Place>): number => Object.values(places).reduce((z, p) => Math.max(z, p.z), 0)

/** extent is how far the notes reach, with the margin: the size the canvas scrolls over. */
export function extent(places: Place[]): { w: number; h: number } {
  let w = 0
  let h = 0
  for (const p of places) {
    w = Math.max(w, p.x + p.w)
    h = Math.max(h, p.y + p.h)
  }
  return { w: w + PAD, h: h + PAD }
}

interface Placeable {
  name: string
  size: string
  rows: number
}

/**
 * firstPlaces are places for the notes the canvas has not seen: where the
 * grid would draw them alone on a board width px wide, in the board's
 * order. On a canvas that already has notes among all, they start below
 * them at the left edge, so nothing new lands on top of what was placed by hand, nor
 * off to the side where a note it no longer shares a row with would be.
 */
export function firstPlaces(all: Placeable[], places: Record<string, Place>, width: number): Record<string, Place> {
  const notes = all.filter((n) => !places[n.name])
  if (!notes.length) return {}
  const inner = Math.max(width - 2 * PAD, 2 * MIN_W)
  const grid = gridPlace(notes, inner)
  const heightOf = (n: Placeable) => heightOfRows(n.rows) ?? NEW_H
  // each grid row is as tall as its tallest note
  const rowH: number[] = []
  grid.forEach((p, i) => (rowH[p.row - 1] = Math.max(rowH[p.row - 1] ?? 0, heightOf(notes[i]))))
  const rowTop: number[] = []
  rowH.reduce((top, h, r) => ((rowTop[r] = top), top + (h ?? 0) + ROW_GAP), 0)
  // below the notes of this board's own (its tab's) that have places: the
  // map holds every tab's, and another tab's must not push these down
  const known = all.filter((n) => places[n.name]).map((n) => places[n.name])
  const below = known.length ? extent(known).h : PAD
  let z = topZ(places)
  const out: Record<string, Place> = {}
  grid.forEach((p, i) => {
    const n = notes[i]
    if (places[n.name]) return
    const x = PAD + (p.column > 1 ? spanPx(p.column - 1, inner) + COL_GAP : 0)
    out[n.name] = {
      x: Math.round(x),
      y: Math.round(below + rowTop[p.row - 1]),
      w: Math.max(MIN_W, Math.round(spanPx(p.span, inner))),
      h: Math.max(MIN_H, heightOf(n)),
      tilt: tiltOf(n.name),
      z: ++z,
    }
  })
  return out
}

/** raise puts a note on top of the stack; null when it is already there. */
export function raise(places: Record<string, Place>, name: string): Place | null {
  const p = places[name]
  if (!p) return null
  const others = Object.entries(places).filter(([n]) => n !== name)
  if (others.every(([, o]) => o.z < p.z)) return null
  return { ...p, z: Math.max(...others.map(([, o]) => o.z)) + 1 }
}

/** moved is a note dragged by (dx, dy) screen px on a canvas zoomed by scale. */
export function moved(start: Place, dx: number, dy: number, scale: number, snap: boolean): Place {
  return {
    ...start,
    x: Math.max(0, snapTo(start.x + dx / scale, snap)),
    y: Math.max(0, snapTo(start.y + dy / scale, snap)),
  }
}

/** The edges a note is resized by: its right, its bottom, or the corner between them. */
export type Edge = 'e' | 's' | 'se'

/** resized is a note whose edge was dragged by (dx, dy) screen px. */
export function resized(start: Place, edge: Edge, dx: number, dy: number, scale: number, snap: boolean): Place {
  const out = { ...start }
  if (edge !== 's') out.w = Math.max(MIN_W, snapTo(start.w + dx / scale, snap))
  if (edge !== 'e') out.h = Math.max(MIN_H, snapTo(start.h + dy / scale, snap))
  return out
}

/** zoomed is the canvas's zoom after a wheel step: in on a scroll up, out on a scroll down, in tenths. */
export function zoomed(scale: number, deltaY: number): number {
  const next = deltaY < 0 ? scale + 0.1 : deltaY > 0 ? scale - 0.1 : scale
  return Math.round(Math.min(ZOOM_MAX, Math.max(ZOOM_MIN, next)) * 10) / 10
}

/** samePlace says whether two places draw the same. */
export const samePlace = (a: Place | undefined, b: Place | undefined): boolean =>
  !!a && !!b && a.x === b.x && a.y === b.y && a.w === b.w && a.h === b.h && a.tilt === b.tilt && a.z === b.z
