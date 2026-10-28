import { describe, expect, test } from 'vitest'
import { logoPaths } from './logo'
import { boxPaths, crayonPaths, linePaths } from './rough'

test('the logo is three sheets, straight in neat', () => {
  const notes = ['#111111', '#222222', '#333333', '#444444', '#555555', '#666666']
  expect(logoPaths(notes, 'doodle', false, '#000000').length).toBeGreaterThanOrEqual(3)
  const neat = logoPaths(notes, 'neat', true, '#000000')
  expect(neat).toHaveLength(3)
  expect(neat.map((s) => s.d).join('')).not.toMatch(/C/)
  expect(logoPaths([], 'doodle', false, '#000')).toEqual([])
})

describe('neat style draws straight', () => {
  test('a neat box has no curves', () => {
    const ps = boxPaths(100, 50, { stroke: '#000', style: 'neat', seed: 3 })
    expect(ps.map((p) => p.d).join(' ')).not.toMatch(/C/i)
  })
  test('a neat line is a straight segment', () => {
    const [p] = linePaths(0, 5, 80, 4, { stroke: '#000', style: 'neat', seed: 3 })
    expect(p.d).not.toMatch(/C/i)
  })
  test('a neat crayon card is a solid wash and a straight outline, no hatching', () => {
    const ps = crayonPaths(100, 50, { hatch: '#f00', wash: '#fee', outline: '#000', style: 'neat', seed: 1 })
    expect(ps.map((p) => p.d).join(' ')).not.toMatch(/C/i)
    expect(ps.some((p) => p.role === 'hatch')).toBe(false)
  })
  test('a doodle crayon card has wash, hatching and outline', () => {
    const ps = crayonPaths(100, 50, { hatch: '#f00', wash: '#fee', outline: '#000', style: 'doodle', seed: 1 })
    expect(ps.map((p) => p.role)).toEqual(expect.arrayContaining(['wash', 'hatch', 'outline']))
  })
  test('a doodle box is hand drawn and seeded', () => {
    const a = boxPaths(100, 50, { stroke: '#000', style: 'doodle', seed: 3 })
    const b = boxPaths(100, 50, { stroke: '#000', style: 'doodle', seed: 3 })
    const c = boxPaths(100, 50, { stroke: '#000', style: 'doodle', seed: 4 })
    expect(a.map((p) => p.d).join()).toMatch(/C/)
    expect(a).toEqual(b)
    expect(a).not.toEqual(c)
  })
})
