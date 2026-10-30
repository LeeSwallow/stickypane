import { describe, expect, it } from 'vitest'
import type { Place } from '../services/canvas'
import { changed, EDGE, firstSize, fitHeight, fits, FIT_MAX, overlaps, tidy, toBack, toFront } from './arrange'
import { firstPlaces, SNAP } from './free'

const at = (x: number, y: number, w: number, h: number, z = 1): Place => ({ x, y, w, h, tilt: 0.5, z })

// a messy canvas: notes on top of each other, one far off to the right
const messy: Record<string, Place> = {
  'a.md': at(300, 300, 301, 203, 3),
  'b.md': at(310, 310, 250, 150, 1),
  'c.md': at(1900, 40, 420, 260, 2),
  'd.md': at(0, 0, 900, 120, 4),
  'e.md': at(12, 600, 160, 96, 5),
}
const order = ['d.md', 'a.md', 'b.md', 'c.md', 'e.md']

describe('tidy', () => {
  for (const width of [480, 720, 1000, 1600]) {
    it(`leaves no note on another and none past the width (${width}px)`, () => {
      const out = tidy(order, messy, width)
      const list = Object.values(out)
      expect(list).toHaveLength(5)
      for (const p of list) {
        expect(p.x).toBeGreaterThanOrEqual(EDGE)
        expect(p.x + p.w).toBeLessThanOrEqual(width - EDGE)
        for (const v of [p.x, p.y, p.w, p.h]) expect(v % SNAP).toBe(0)
      }
      for (let i = 0; i < list.length; i++) for (let j = i + 1; j < list.length; j++) expect(overlaps(list[i], list[j])).toBe(false)
    })
  }
  it('keeps the board order: reading left to right, top to bottom', () => {
    const out = tidy(order, messy, 1000)
    const read = [...order].sort((a, b) => out[a].y - out[b].y || out[a].x - out[b].x)
    expect(read).toEqual(order)
  })
  it('keeps each size, the tilt and the stack', () => {
    const out = tidy(order, messy, 1600)
    expect(out['c.md']).toMatchObject({ w: 424, h: 264, tilt: 0.5, z: 2 })
    expect(out['e.md']).toMatchObject({ w: 160, h: 96 })
  })
  it('narrows a note wider than the canvas', () => {
    const out = tidy(order, messy, 480)
    expect(out['d.md'].w).toBeLessThanOrEqual(480 - 2 * EDGE)
  })
  it('a second tidy changes nothing', () => {
    for (const width of [480, 1000]) {
      const once = tidy(order, messy, width)
      expect(tidy(order, once, width)).toEqual(once)
      expect(changed(once, tidy(order, once, width))).toEqual({})
    }
  })
  it('leaves out names with no place', () => {
    expect(Object.keys(tidy(['x.md', 'a.md'], messy, 800))).toEqual(['a.md'])
  })
})

describe('first places', () => {
  it('a note alone in its row keeps its own width, inside the canvas (#63)', () => {
    for (const width of [640, 720, 1000]) {
      const p = firstPlaces([{ name: 'welcome.md', size: '2/3', rows: 0 }], {}, width)['welcome.md']
      expect(p.w).toBeLessThan(width * 0.75)
      expect(p.w).toBeGreaterThan(width * 0.5)
      expect(fits(p, width)).toBe(true)
    }
  })
  it('every note of a board lands inside the width', () => {
    const notes = ['1/4', '1/3', '1/2', '2/3', '3/4', '1', 'card'].map((size, i) => ({ name: `n${i}.md`, size, rows: 0 }))
    for (const width of [480, 720, 1000]) for (const p of Object.values(firstPlaces(notes, {}, width))) expect(fits(p, width)).toBe(true)
  })
})

describe('a new note fits what it holds', () => {
  it('on the grid, no lower than the smallest, no higher than the most, and unmeasured keeps its height', () => {
    expect(fitHeight(201, 280)).toBe(208)
    expect(fitHeight(40, 280)).toBe(96)
    expect(fitHeight(5000, 280)).toBe(FIT_MAX)
    expect(fitHeight(0, 280)).toBe(280)
  })
})

describe('one note', () => {
  it('to the front and to the back', () => {
    expect(toFront(messy, 'e.md')).toBeNull()
    expect(toFront(messy, 'b.md')).toEqual({ 'b.md': { ...messy['b.md'], z: 6 } })
    expect(toBack(messy, 'b.md')).toBeNull()
    const back = toBack(messy, 'e.md')!
    const stack = { ...messy, ...back }
    expect(Math.min(...Object.values(stack).map((p) => p.z))).toBe(stack['e.md'].z)
    expect(new Set(Object.values(stack).map((p) => p.z)).size).toBe(5)
  })
  it('back to its first size, where it is', () => {
    const p = firstSize({ name: 'a.md', size: '1/2', rows: 0 }, messy['a.md'], 1000)
    expect(p.x).toBe(300)
    expect(p.y).toBe(300)
    expect(p.w).toBeGreaterThan(400)
    expect(p.w).toBeLessThan(500)
  })
})
