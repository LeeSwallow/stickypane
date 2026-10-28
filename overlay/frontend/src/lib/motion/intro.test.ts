import { describe, expect, it } from 'vitest'
import { dayOf, fadeIn, fly, INTRO, INTRO_KEY, introMode, sheet, wriggle } from './intro'
import { lastDay, remember } from './stage'

const end = (t: { timing: KeyframeAnimationOptions }) => Number(t.timing.delay ?? 0) + Number(t.timing.duration ?? 0)

describe('introMode (04 §4.3.1)', () => {
  it('the first window of the day plays it all; later ones only wriggle the notes', () => {
    expect(introMode('full', '2026-10-29', null)).toBe('full')
    expect(introMode('full', '2026-10-29', '2026-10-28')).toBe('full')
    expect(introMode('full', '2026-10-29', '2026-10-29')).toBe('short')
  })
  it('reduced motion is a fade, and motion off is nothing', () => {
    expect(introMode('reduce', '2026-10-29', null)).toBe('fade')
    expect(introMode('off', '2026-10-29', null)).toBe('none')
  })
  it('a day is the local calendar day', () => {
    expect(dayOf(new Date(2026, 0, 5, 23, 59))).toBe('2026-01-05')
  })
})

describe('the full intro', () => {
  it('ends in under 1.5 s', () => {
    expect(INTRO.end).toBeLessThan(1500)
    for (let i = 0; i < 3; i++) for (const neat of [false, true]) expect(end(sheet(i, neat))).toBeLessThanOrEqual(INTRO.settle)
    expect(end(fly(10, 10, 0.2))).toBe(INTRO.settle + INTRO.fly)
    expect(INTRO.notesAt + end(wriggle(20, 0, false))).toBeLessThanOrEqual(INTRO.end + INTRO.note)
  })

  it('the sheets drop blue, pink, yellow in turn and wriggle −3° ↔ +2° twice', () => {
    expect([0, 1, 2].map((i) => sheet(i, false).timing.delay)).toEqual([0, 80, 160])
    const turns = sheet(0, false).keyframes.map((k) => k.transform).filter((t) => /^rotate\((-3|2)deg\)$/.test(String(t)))
    expect(turns).toEqual(['rotate(-3deg)', 'rotate(2deg)', 'rotate(-3deg)', 'rotate(2deg)'])
  })

  it('neat drops and fades without the wriggle', () => {
    expect(JSON.stringify(sheet(1, true).keyframes)).not.toContain('rotate')
    expect(JSON.stringify(wriggle(0, 0, true).keyframes)).not.toContain('rotate')
  })

  it('at most eight notes wriggle, 30 ms apart; the rest only fade', () => {
    expect([0, 1, 7].map((i) => wriggle(i, 650, false).timing.delay)).toEqual([650, 680, 860])
    expect(JSON.stringify(wriggle(7, 0, false).keyframes)).toContain('1.5deg')
    expect(wriggle(8, 0, false).keyframes).toEqual([{ opacity: 0 }, { opacity: 1 }])
    expect(wriggle(30, 0, false).timing.delay).toBe(INTRO.notesMax * INTRO.notesGap)
  })

  it('nothing in it repeats', () => {
    for (const t of [sheet(0, false), fly(1, 1, 1), wriggle(0, 0, false), fadeIn()]) expect(t.timing.iterations ?? 1).toBe(1)
  })

  it('reduced is opacity alone, 120 ms', () => {
    expect(fadeIn()).toEqual({ keyframes: [{ opacity: 0 }, { opacity: 1 }], timing: { duration: 120, easing: 'linear' } })
  })
})

describe('the day it last played', () => {
  const store = () => {
    const m = new Map<string, string>()
    return { getItem: (k: string) => m.get(k) ?? null, setItem: (k: string, v: string) => void m.set(k, v), m } as unknown as Storage & { m: Map<string, string> }
  }
  it('is kept in the store', () => {
    const s = store()
    expect(lastDay(s)).toBeNull()
    remember(s, '2026-10-29')
    expect(lastDay(s)).toBe('2026-10-29')
    expect((s as unknown as { m: Map<string, string> }).m.get(INTRO_KEY)).toBe('2026-10-29')
  })
  it('a store that refuses, or none, is no harm', () => {
    const bad = { getItem: () => { throw new Error('denied') }, setItem: () => { throw new Error('denied') } } as unknown as Storage
    expect(lastDay(bad)).toBeNull()
    expect(() => remember(bad, 'x')).not.toThrow()
    expect(lastDay(null)).toBeNull()
  })
})
