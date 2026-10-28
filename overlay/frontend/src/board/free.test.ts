import { describe, expect, it } from 'vitest'
import type { Place } from '../services/canvas'
import { layoutOf } from '../services/canvas'
import {
  extent, firstPlaces, MIN_H, MIN_W, moved, NEW_H, PAD, raise, resized, samePlace, snapTo, tiltOf, topZ, zoomed,
} from './free'
import { ROW_PX } from './layout'

const at = (x: number, y: number, w = 200, h = 100, z = 1): Place => ({ x, y, w, h, tilt: 0, z })

describe('the layout the shell answers', () => {
  it('is a canvas with snap unless it says otherwise', () => {
    expect(layoutOf(null)).toEqual({ mode: 'canvas', snap: true, places: {}, torn: {} })
    expect(layoutOf({ mode: 'grid', snap: false })).toMatchObject({ mode: 'grid', snap: false })
    expect(layoutOf({ mode: 'tiles' as never }).mode).toBe('canvas')
  })
})

describe('a note new to the canvas', () => {
  const notes = [
    { name: 'a.md', size: '1/2', rows: 0 },
    { name: 'b.md', size: '1/2', rows: 10 },
    { name: 'c.md', size: '1', rows: 0 },
  ]
  it('starts where the grid would draw it', () => {
    const p = firstPlaces(notes, {}, 1048)
    expect(Object.keys(p)).toEqual(['a.md', 'b.md', 'c.md'])
    expect(p['a.md']).toMatchObject({ x: PAD, y: PAD })
    // side by side, half each
    expect(p['b.md'].y).toBe(PAD)
    expect(p['b.md'].x).toBeGreaterThan(p['a.md'].x + p['a.md'].w)
    expect(p['b.md'].h).toBe(10 * ROW_PX)
    // the next row is below the taller of the two
    expect(p['c.md'].y).toBeGreaterThan(PAD + NEW_H)
    expect(p['c.md'].w).toBeGreaterThan(p['a.md'].w * 2)
    // stacked in the board's order
    expect([p['a.md'].z, p['b.md'].z, p['c.md'].z]).toEqual([1, 2, 3])
  })
  it('is placed below what was placed by hand, and the rest stay', () => {
    const known = { 'a.md': at(500, 40, 200, 300, 7) }
    const p = firstPlaces(notes, known, 1048)
    expect(p['a.md']).toBeUndefined()
    expect(p['b.md'].y).toBeGreaterThanOrEqual(340)
    expect(p['b.md'].z).toBe(8)
    expect(firstPlaces(notes, { ...known, ...p }, 1048)).toEqual({})
  })
  it('leans a little, the same every time', () => {
    expect(tiltOf('a.md')).toBe(tiltOf('a.md'))
    for (const n of ['a', 'b', 'release.md', '한글.md']) expect(Math.abs(tiltOf(n))).toBeLessThanOrEqual(1.5)
  })
})

describe('moving, resizing and stacking', () => {
  it('lands on the 8px grid with snap, anywhere without', () => {
    expect(snapTo(13, true)).toBe(16)
    expect(snapTo(13.4, false)).toBe(13)
    expect(moved(at(100, 100), 13, -5, 1, true)).toMatchObject({ x: 112, y: 96 })
    expect(moved(at(100, 100), 13, -5, 1, false)).toMatchObject({ x: 113, y: 95 })
  })
  it('counts the drag in the canvas\'s px when it is zoomed, and never goes off its top or left', () => {
    expect(moved(at(100, 100), 100, 100, 2, false)).toMatchObject({ x: 150, y: 150 })
    expect(moved(at(10, 10), -100, -100, 1, true)).toMatchObject({ x: 0, y: 0 })
  })
  it('resizes by an edge or the corner, never below the smallest', () => {
    expect(resized(at(0, 0), 'e', 40, 40, 1, true)).toMatchObject({ w: 240, h: 100 })
    expect(resized(at(0, 0), 's', 40, 40, 1, true)).toMatchObject({ w: 200, h: 144 })
    expect(resized(at(0, 0), 'se', -500, -500, 1, true)).toMatchObject({ w: MIN_W, h: MIN_H })
  })
  it('puts a touched note on top, once', () => {
    const p = { a: at(0, 0, 200, 100, 1), b: at(0, 0, 200, 100, 5) }
    expect(raise(p, 'a')).toMatchObject({ z: 6 })
    expect(raise(p, 'b')).toBeNull()
    expect(raise(p, 'nope')).toBeNull()
    expect(raise({ a: at(0, 0) }, 'a')).toBeNull()
    expect(topZ(p)).toBe(5)
  })
  it('scrolls over the notes and a margin', () => {
    expect(extent([at(10, 20, 100, 50), at(300, 0, 40, 400)])).toEqual({ w: 340 + PAD, h: 400 + PAD })
    expect(extent([])).toEqual({ w: PAD, h: PAD })
  })
  it('zooms in tenths from half to double', () => {
    expect(zoomed(1, -1)).toBe(1.1)
    expect(zoomed(1, 1)).toBe(0.9)
    expect(zoomed(2, -1)).toBe(2)
    expect(zoomed(0.5, 3)).toBe(0.5)
    expect(zoomed(1.3, 0)).toBe(1.3)
  })
  it('tells a place that draws the same', () => {
    expect(samePlace(at(1, 2), at(1, 2))).toBe(true)
    expect(samePlace(at(1, 2), at(1, 3))).toBe(false)
    expect(samePlace(undefined, at(1, 2))).toBe(false)
  })
})
