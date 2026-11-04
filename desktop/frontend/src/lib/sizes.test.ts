import { describe, expect, test } from 'vitest'
import { columnSpan, FRACTIONS, normalSize, ratioOf, shareOf } from './sizes'

describe('a size is read once', () => {
  test('every fraction is its numerator over its denominator', () => {
    expect(FRACTIONS.map(ratioOf)).toEqual([[1, 4], [1, 3], [1, 2], [2, 3], [3, 4], [1, 1]])
    for (const f of FRACTIONS) expect(shareOf(f)).toBeCloseTo(ratioOf(f)[0] / ratioOf(f)[1])
  })
  test('the old names and a size in any case read as the board writes them', () => {
    expect([normalSize('Page'), normalSize(' half '), normalSize('CARD'), normalSize('2/5'), normalSize('')]).toEqual(['1', '1/2', 'card', null, null])
  })
  test('columns read the old names too; a card or nonsense is one column', () => {
    expect([columnSpan('page', 4), columnSpan('HALF', 4), columnSpan('card', 4), columnSpan('2/5', 4)]).toEqual([4, 2, 1, 1])
  })
})
