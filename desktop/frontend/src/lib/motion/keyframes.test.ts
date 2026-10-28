import { describe, expect, it } from 'vitest'
import { levelOf, movement, pen, slide, type Look } from './keyframes'
import type { Effect } from './plan'

const full: Look = { style: 'doodle', level: 'full' }
const neat: Look = { style: 'neat', level: 'full' }
const reduce: Look = { style: 'doodle', level: 'reduce' }
const all: Effect[] = ['carry', 'slap', 'drop', 'unfold', 'push', 'tick', 'untick', 'nudge', 'glow', 'leave']

const props = (k: Keyframe[]) => new Set(k.flatMap((f) => Object.keys(f).filter((x) => !['offset', 'easing', 'composite'].includes(x))))

describe('movement', () => {
  it('moves only transform and opacity (04 §4.7)', () => {
    for (const e of all)
      for (const look of [full, neat, reduce]) {
        const m = movement(e, look, { x: 10, y: -20 })!
        for (const p of props(m.keyframes)) expect(['transform', 'opacity', 'transformOrigin']).toContain(p)
      }
  })

  it('a carried card starts where it was, lifted and tilted, and lands flat at its place', () => {
    const m = movement('carry', full, { x: -120, y: 40 })!
    expect(m.timing.duration).toBe(620)
    expect(m.keyframes[0].transform).toBe('translate(-120px, 40px) scale(1)')
    expect(String(m.keyframes[1].transform)).toContain('scale(1.07) rotate(3deg)')
    expect(m.keyframes.at(-1)!.transform).toBe('translate(0px, 0px) scale(1)')
  })

  it('the stall wobble goes −2.5° → +2° → −1° → 0 once', () => {
    const m = movement('nudge', full)!
    expect(m.keyframes.map((k) => k.transform)).toEqual(['rotate(0deg)', 'rotate(-2.5deg)', 'rotate(2deg)', 'rotate(-1deg)', 'rotate(0deg)'])
    expect(m.timing.iterations ?? 1).toBe(1)
  })

  it('nothing loops (anti-goal X4)', () => {
    for (const e of all) for (const look of [full, neat, reduce]) expect(movement(e, look)!.timing.iterations ?? 1).toBe(1)
  })

  it('the glow fades within about 1.5 s', () => {
    const m = movement('glow', full)!
    expect(m.timing.duration).toBe(1500)
    expect(m.keyframes.at(-1)!.opacity).toBe(0)
  })

  it('neat keeps the moves and drops the tilts and wobbles', () => {
    for (const e of all) expect(JSON.stringify(movement(e, neat)!.keyframes)).not.toMatch(/rotate\((?!0deg)/)
  })

  it('reduced motion is opacity alone, 180 ms; the glow is an outline that stays two seconds (04 §4.6)', () => {
    for (const e of all) {
      const m = movement(e, reduce)!
      expect(props(m.keyframes)).toEqual(new Set(['opacity']))
      expect(m.timing.duration).toBe(e === 'glow' ? 2000 : 180)
    }
  })

  it('off is nothing at all', () => {
    for (const e of all) expect(movement(e, { style: 'doodle', level: 'off' })).toBeNull()
    expect(slide({ x: 40, y: 0 }, { style: 'doodle', level: 'off' })).toBeNull()
  })
})

describe('slide and pen', () => {
  it('a pushed part slides from its old place, and not at all when it did not move or motion is reduced', () => {
    expect(slide({ x: 0, y: 30 }, full)!.keyframes).toEqual([{ transform: 'translate(0px, 30px)' }, { transform: 'translate(0px, 0px)' }])
    expect(slide({ x: 0.4, y: 0 }, full)).toBeNull()
    expect(slide({ x: 0, y: 30 }, reduce)).toBeNull()
  })

  it('a line is drawn by its dash, in doodle style only', () => {
    expect(pen(50, full)!.keyframes).toEqual([
      { strokeDasharray: '50', strokeDashoffset: '50' },
      { strokeDasharray: '50', strokeDashoffset: '0' },
    ])
    expect(pen(50, neat)).toBeNull()
    expect(pen(0, full)).toBeNull()
  })
})

describe('levelOf', () => {
  it('the app setting wins, then the system', () => {
    expect(levelOf('', false)).toBe('full')
    expect(levelOf('', true)).toBe('reduce')
    expect(levelOf('reduce', false)).toBe('reduce')
    expect(levelOf('off', true)).toBe('off')
    expect(levelOf(undefined, false)).toBe('full')
  })
})
