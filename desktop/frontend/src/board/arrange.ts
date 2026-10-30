// Tidying a board's layout (03 §3.6.2): lay the canvas's notes out again,
// put one note back to its first size, bring one to the front or send it
// to the back. Pure: Board.svelte hands what these say to the shell
// (services/canvas.ts), which keeps it in overlay.json.

import type { Place } from '../services/canvas'
import { firstPlaces, MIN_H, PAD, snapTo, SNAP } from './free'

/** GAP is the space tidying leaves between notes, on the 8px grid. */
export const GAP = 3 * SNAP
/** EDGE is the margin round the notes, on the 8px grid. */
export const EDGE = snapTo(PAD, true)

/**
 * tidy lays the notes named in order out again from the top left, in
 * rows, left to right, each keeping its size (narrowed to the width when
 * it is wider), on the 8px grid, none on another and none past width.
 * A note with no place is left out. Tidying what tidy gave changes
 * nothing.
 */
export function tidy(order: string[], places: Record<string, Place>, width: number): Record<string, Place> {
  const right = Math.max(width - EDGE, EDGE + SNAP)
  const widest = Math.max(SNAP, Math.floor((right - EDGE) / SNAP) * SNAP)
  const out: Record<string, Place> = {}
  let x = EDGE
  let y = EDGE
  let rowH = 0
  for (const name of order) {
    const p = places[name]
    if (!p) continue
    const w = Math.min(snapTo(p.w, true), widest)
    const h = snapTo(p.h, true)
    if (x > EDGE && x + w > right) {
      x = EDGE
      y += rowH + GAP
      rowH = 0
    }
    out[name] = { ...p, x, y, w, h }
    x += w + GAP
    rowH = Math.max(rowH, h)
  }
  return out
}

/** changed are the places in next that differ from before. */
export function changed(before: Record<string, Place>, next: Record<string, Place>): Record<string, Place> {
  const out: Record<string, Place> = {}
  for (const [name, p] of Object.entries(next)) {
    const b = before[name]
    if (!b || b.x !== p.x || b.y !== p.y || b.w !== p.w || b.h !== p.h || b.z !== p.z || b.tilt !== p.tilt) out[name] = p
  }
  return out
}

interface Placeable {
  name: string
  size: string
  rows: number
}

/** firstSize is a note's place with the size it would first have on a canvas width wide, where it is now. */
export function firstSize(note: Placeable, at: Place, width: number): Place {
  const first = firstPlaces([note], {}, width)[note.name]
  return first ? { ...at, w: first.w, h: first.h } : at
}

/** FIT_MAX is the tallest a new note is made to fit its content; a longer one scrolls inside. */
export const FIT_MAX = 560

/**
 * fitHeight is the height a new note on the canvas takes for content
 * natural px tall: on the 8px grid, no lower than the canvas's smallest,
 * no higher than FIT_MAX. A note with nothing measured (0) keeps fallback.
 */
export function fitHeight(natural: number, fallback: number): number {
  if (!(natural > 0)) return fallback
  return Math.min(FIT_MAX, Math.max(MIN_H, Math.ceil(natural / SNAP) * SNAP))
}

/** fits says whether a place is inside a canvas width wide. */
export const fits = (p: Place, width: number): boolean => p.x >= 0 && p.x + p.w <= width

/** toFront is the note on top of every other; null when it is already. */
export function toFront(places: Record<string, Place>, name: string): Record<string, Place> | null {
  const p = places[name]
  if (!p) return null
  const others = Object.entries(places).filter(([n]) => n !== name)
  if (others.every(([, o]) => o.z < p.z)) return null
  return { [name]: { ...p, z: Math.max(...others.map(([, o]) => o.z)) + 1 } }
}

/** toBack is the note under every other, the stack numbered again from 1; null when it is already. */
export function toBack(places: Record<string, Place>, name: string): Record<string, Place> | null {
  const p = places[name]
  if (!p) return null
  const others = Object.entries(places).filter(([n]) => n !== name)
  if (others.every(([, o]) => o.z > p.z)) return null
  const stack = [name, ...others.sort(([, a], [, b]) => a.z - b.z).map(([n]) => n)]
  const out: Record<string, Place> = {}
  stack.forEach((n, i) => {
    if (places[n].z !== i + 1) out[n] = { ...places[n], z: i + 1 }
  })
  return out
}

/** overlaps says whether two places share any area. */
export const overlaps = (a: Place, b: Place): boolean => a.x < b.x + b.w && b.x < a.x + a.w && a.y < b.y + b.h && b.y < a.y + a.h
