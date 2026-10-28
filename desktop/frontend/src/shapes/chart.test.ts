import { describe, expect, it } from 'vitest'
import { bars, calendar, grouped, spark } from './chart'
import type { ChartPoint } from './data'

const p = (label: string, value: number | null, text = String(value)): ChartPoint => ({ label, text, value })

describe('bars', () => {
  it('scales to the largest value; nothing for a zero, a negative or a value that is not a number', () => {
    expect(bars([p('a', 50), p('b', 100), p('c', 0), p('d', -3), p('e', null, 'x')]).map((b) => b.ratio)).toEqual([0.5, 1, 0, 0, 0])
    expect(bars([p('a', 0)])[0].ratio).toBe(0)
  })
})

describe('spark', () => {
  it('runs from the first to the last value across the box, highest at the top', () => {
    const s = spark([p('mon', 3), p('tue', 9), p('wed', 6, '6.0')], 100, 10)!
    expect(s.line).toEqual([[0, 10], [50, 0], [100, 5]])
    expect([s.low, s.high, s.first, s.last, s.latest]).toEqual([3, 9, 'mon', 'wed', '6.0'])
  })
  it('a flat line sits in the middle; fewer than two values draw no line', () => {
    expect(spark([p('a', 2), p('b', 2)], 10, 10)!.line).toEqual([[0, 5], [10, 5]])
    expect(spark([p('a', 2), p('b', null, 'x')], 10, 10)).toBeNull()
  })
})

describe('calendar', () => {
  it('lays days out in weeks from Monday, shaded by rank, as the terminal does', () => {
    // 2026-10-05 is a Monday
    const c = calendar([p('2026-10-05', 1), p('2026-10-06', 2), p('2026-10-07', 3), p('2026-10-08', 4), p('2026-10-12', 0), p('note', 9)])!
    expect(c.weeks).toHaveLength(2)
    expect(c.weeks[0].map((d) => d.date)).toEqual(['2026-10-05', '2026-10-06', '2026-10-07', '2026-10-08', '2026-10-09', '2026-10-10', '2026-10-11'])
    expect(c.weeks[0].slice(0, 5).map((d) => d.level)).toEqual([0, 1, 2, 3, null])
    expect(c.weeks[1][0].level).toBe('zero')
    expect(c.months).toEqual([10, 0])
    expect(c.total).toBe(10)
    expect(c.peak).toEqual({ text: '4', date: '2026-10-08' })
  })
  it('one day far above the others does not leave them all pale', () => {
    const c = calendar([p('2026-10-05', 1), p('2026-10-06', 2), p('2026-10-07', 1000)])!
    expect(c.weeks[0].slice(0, 3).map((d) => d.level)).toEqual([1, 2, 3])
  })
  it('shows the latest weeks that fit', () => {
    const c = calendar([p('2026-01-05', 1), p('2026-10-05', 2)], 4)!
    expect(c.weeks).toHaveLength(4)
    expect(c.weeks[3][0].date).toBe('2026-10-05')
  })
  it('is null when no label is a date', () => {
    expect(calendar([p('mon', 1), p('2026-02-30', 1)])).toBeNull()
  })
})

it('grouped writes sums as the terminal does', () => {
  expect(grouped(1234567)).toBe('1,234,567')
  expect(grouped(-1200.5)).toBe('-1,200.5')
  expect(grouped(2.0)).toBe('2')
  expect(grouped(0.123)).toBe('0.12')
})
