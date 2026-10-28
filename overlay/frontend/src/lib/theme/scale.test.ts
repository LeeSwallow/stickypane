import { describe, expect, test } from 'vitest'
import { THEMES } from './palettes'
import { resolveThemeName } from './mode'
import { DENSITY, ELEVATION, MAX_TILT, MOTION, RADIUS, SPACE, STROKE, TYPE, Z, scaleVars, tiltOf } from './scale'
import { chromeGrounds, contrast, deriveTokens, MIN_CONTRAST, noteTitleColor, TEXT_PAIRS } from './tokens'

describe('the scale is consistent', () => {
  test('spacing is on the 4px grid and grows', () => {
    expect(SPACE).toHaveLength(10)
    SPACE.forEach((n, i) => {
      expect(n % 4, `space-${i + 1}`).toBe(0)
      if (i) expect(n).toBeGreaterThan(SPACE[i - 1])
    })
  })
  test.each(Object.entries(TYPE))('the %s type scale is monotonic', (_, styles) => {
    const list = Object.values(styles)
    for (let i = 1; i < list.length; i++) {
      expect(list[i].size).toBeLessThan(list[i - 1].size)
      expect(list[i].line).toBeLessThanOrEqual(list[i - 1].line)
    }
    for (const s of list) {
      expect(s.line).toBeGreaterThanOrEqual(s.size)
      expect(s.line % 2, 'line heights are even').toBe(0)
    }
  })
  test('hand type is larger than the body it sits beside (03 §3.5)', () => {
    expect(TYPE.hand.label.size - TYPE.body.lg.size).toBeGreaterThanOrEqual(3)
  })
  test('strokes, radii, elevation and layers are ordered', () => {
    expect(STROKE.hairline).toBeLessThan(STROKE.ui)
    expect(STROKE.ui).toBeLessThan(STROKE.note)
    expect(STROKE.note).toBeLessThan(STROKE.emphasis)
    const r = [RADIUS.sm, RADIUS.md, RADIUS.lg, RADIUS.xl, RADIUS.window]
    expect([...r].sort((a, b) => a - b)).toEqual(r)
    expect(Object.keys(ELEVATION)).toEqual(['flat', 'taped', 'lifted', 'floating'])
    const z = Object.values(Z)
    expect([...z].sort((a, b) => a - b)).toEqual(z)
  })
  test('motion matches 04 §4.2', () => {
    expect([MOTION.short, MOTION.base, MOTION.long, MOTION.carry]).toEqual([120, 200, 320, 620])
    const v = scaleVars()
    expect(v['--t-base']).toBe('200ms')
    expect(v['--ease']).toBe('cubic-bezier(.2,.8,.2,1)')
  })
  test('compact is smaller than comfortable, on the grid', () => {
    for (const k of Object.keys(DENSITY.comfortable) as (keyof typeof DENSITY.comfortable)[]) {
      expect(DENSITY.compact[k]).toBeLessThan(DENSITY.comfortable[k])
      expect(DENSITY.compact[k] % 4).toBe(0)
      expect(DENSITY.comfortable[k] % 4).toBe(0)
    }
    expect(scaleVars('compact')['--control-md']).toBe('24px')
  })
  test('every token becomes a variable', () => {
    const v = scaleVars()
    expect(v['--space-1']).toBe('4px')
    expect(v['--space-10']).toBe('64px')
    expect(v['--text-hand-title']).toBe('22px')
    expect(v['--leading-body-md']).toBe('20px')
    expect(v['--stroke-note']).toBe('2px')
    expect(v['--z-dialog']).toBe('400')
    expect(v['--elev-floating']).toContain('--shadow-rgb')
  })
})

describe('tilt', () => {
  test('never more than 1.5 degrees, and none in neat', () => {
    const seen = new Set<number>()
    for (let s = 0; s < 500; s++) {
      const t = tiltOf(s * 7919, 'doodle')
      expect(Math.abs(t)).toBeLessThanOrEqual(MAX_TILT)
      seen.add(t)
      expect(tiltOf(s, 'neat')).toBe(0)
    }
    expect(seen.size).toBe(9)
  })
})

test('a note title in its colour reads as large text (3:1) on the paper', () => {
  for (const p of THEMES) {
    for (const c of p.notes) {
      expect(contrast(noteTitleColor(c, p, p.dark), chromeGrounds(p, p.dark).paper)).toBeGreaterThanOrEqual(3)
    }
  }
})

describe('text tokens reach 4.5:1 in every theme x mode', () => {
  for (const p of THEMES) {
    for (const mode of ['dark', 'light'] as const) {
      const q = THEMES.find((t) => t.name === resolveThemeName(p.name, mode, false))!
      test(`${p.name} in ${mode} (${q.name})`, () => {
        const t = deriveTokens(q, q.dark)
        for (const [text, grounds] of TEXT_PAIRS) {
          for (const g of grounds) {
            expect(contrast(t[text], t[g]), `${text} on ${g}: ${t[text]} / ${t[g]}`).toBeGreaterThanOrEqual(MIN_CONTRAST)
          }
        }
        // the focus ring is a non-text indicator: 3:1 on the paper and the chrome
        expect(contrast(t['--focus'], t['--paper'])).toBeGreaterThanOrEqual(3)
        expect(contrast(t['--focus'], t['--chrome'])).toBeGreaterThanOrEqual(3)
      })
    }
  }
})
