import { describe, expect, it } from 'vitest'
import { columnsFor, spans, UNITS } from './grid'

describe('the home grid', () => {
  it('two or three cards to a row as the window allows', () => {
    expect(columnsFor(900)).toBe(3)
    expect(columnsFor(600)).toBe(2)
    expect(columnsFor(360)).toBe(1)
  })
  it('every row fills the width, and the cards of a row are as wide as each other', () => {
    for (const cols of [1, 2, 3])
      for (let n = 1; n <= 8; n++) {
        const s = spans(n, cols)
        let at = 0
        const rows: number[][] = []
        for (const w of s) {
          if (at % UNITS === 0) rows.push([])
          rows[rows.length - 1].push(w)
          at += w
        }
        for (const r of rows) {
          expect(r.reduce((a, b) => a + b, 0)).toBe(UNITS)
          expect(new Set(r).size).toBe(1)
        }
      }
  })
  it('a lone card in a row of three takes the row; full rows are thirds', () => {
    expect(spans(4, 3)).toEqual([2, 2, 2, 6])
    expect(spans(5, 3)).toEqual([2, 2, 2, 3, 3])
    expect(spans(2, 3)).toEqual([3, 3])
  })
})
