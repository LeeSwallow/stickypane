// The movements of 04 §4.3 as Web Animations keyframes: transform and
// opacity only (§4.7). Pure, so a test reads them. Neat style keeps the
// moves and fades and drops the tilts and wobbles; reduced motion (§4.6)
// turns every one into a change of opacity.

import { MOTION } from '../theme/scale'
import type { Effect } from './plan'

export type Level = 'full' | 'reduce' | 'off'

export interface Look {
  style: 'doodle' | 'neat'
  level: Level
}

export interface Movement {
  keyframes: Keyframe[]
  timing: KeyframeAnimationOptions
}

/** How far a part has to come from, for the slides (FLIP): its old place minus its new one. */
export interface Delta {
  x: number
  y: number
}

const t = (x: number, y: number, extra = '') => `translate(${x}px, ${y}px)${extra ? ' ' + extra : ''}`

/** fade is §4.6's answer to every movement: the part comes in by opacity. */
const fade = (from = 0): Movement => ({
  keyframes: [{ opacity: from }, { opacity: 1 }],
  timing: { duration: MOTION.reduced, easing: 'linear' },
})

/**
 * movement is the keyframes of one effect. delta is where a sliding part
 * was (carry, slap, tick, untick); null when it did not move. It is null
 * when the level is off.
 */
export function movement(effect: Effect, look: Look, delta: Delta | null = null): Movement | null {
  if (look.level === 'off') return null
  const doodle = look.style === 'doodle'
  const d = delta ?? { x: 0, y: 0 }
  if (look.level === 'reduce') {
    switch (effect) {
      case 'leave':
        return { keyframes: [{ opacity: 1 }, { opacity: 0 }], timing: { duration: MOTION.reduced, easing: 'linear' } }
      case 'glow':
        // the light is a thick outline that stays two seconds (§4.6)
        return { keyframes: [{ opacity: 1 }, { opacity: 1, offset: 0.95 }, { opacity: 0 }], timing: { duration: 2000, easing: 'linear' } }
      case 'nudge':
      case 'carry':
      case 'tick':
      case 'untick':
      case 'slap':
        return fade(0.4)
      default:
        return fade()
    }
  }
  switch (effect) {
    case 'carry':
      // lift (1.07, +3°) where it was, carry, slap down (.97 → 1)
      return {
        keyframes: [
          { transform: t(d.x, d.y, 'scale(1)'), offset: 0 },
          { transform: t(d.x, d.y - 6, doodle ? 'scale(1.07) rotate(3deg)' : 'scale(1.04)'), offset: 0.2, easing: MOTION.ease },
          { transform: t(0, -6, doodle ? 'scale(1.07) rotate(3deg)' : 'scale(1.04)'), offset: 0.75, easing: MOTION.slap },
          { transform: t(0, 0, 'scale(0.97)'), offset: 0.88 },
          { transform: t(0, 0, 'scale(1)'), offset: 1 },
        ],
        timing: { duration: MOTION.carry, easing: 'linear' },
      }
    case 'slap':
      return {
        keyframes: [{ transform: t(d.x, d.y, 'scale(0.97)') }, { transform: t(0, 0, 'scale(1)') }],
        timing: { duration: MOTION.base, easing: MOTION.slap },
      }
    case 'drop':
      return {
        keyframes: [
          { opacity: 0, transform: t(0, -26, doodle ? 'scale(1.06) rotate(4deg)' : 'scale(1.03)') },
          { opacity: 1, transform: t(0, 0, 'scale(1) rotate(0deg)') },
        ],
        timing: { duration: 380, easing: MOTION.slap },
      }
    case 'unfold':
      return {
        keyframes: [
          { opacity: 0, transform: 'translateY(-6px) scaleY(0.6)', transformOrigin: 'top' },
          { opacity: 1, transform: 'translateY(0) scaleY(1)', transformOrigin: 'top' },
        ],
        timing: { duration: MOTION.base, easing: MOTION.ease },
      }
    case 'push':
      return {
        keyframes: [
          { opacity: 0, transform: 'translateY(0.9em)' },
          { opacity: 1, transform: 'translateY(0)' },
        ],
        timing: { duration: MOTION.base, easing: MOTION.ease },
      }
    case 'tick':
      // the row slides to where done items go, then settles
      return {
        keyframes: [
          { transform: t(d.x, d.y, 'scale(1)'), offset: 0 },
          { transform: t(0, 0, doodle ? 'scale(1.03) rotate(-0.6deg)' : 'scale(1.02)'), offset: 0.6, easing: MOTION.slap },
          { transform: t(0, 0, 'scale(1) rotate(0deg)'), offset: 1 },
        ],
        timing: { duration: 500, easing: MOTION.ease },
      }
    case 'untick':
      return {
        keyframes: [
          { opacity: 0.5, transform: t(d.x, d.y) },
          { opacity: 1, transform: t(0, 0) },
        ],
        timing: { duration: MOTION.short, easing: MOTION.ease },
      }
    case 'nudge':
      if (!doodle) return { keyframes: [{ opacity: 0.5 }, { opacity: 1 }], timing: { duration: MOTION.long, easing: MOTION.ease } }
      return {
        keyframes: [
          { transform: 'rotate(0deg)' },
          { transform: 'rotate(-2.5deg)' },
          { transform: 'rotate(2deg)' },
          { transform: 'rotate(-1deg)' },
          { transform: 'rotate(0deg)' },
        ],
        timing: { duration: 520, easing: MOTION.ease },
      }
    case 'glow':
      // comes up quickly, holds a moment, calms down: once
      return {
        keyframes: [
          { opacity: 0, offset: 0 },
          { opacity: 1, offset: 0.1 },
          { opacity: 1, offset: 0.3 },
          { opacity: 0, offset: 1 },
        ],
        timing: { duration: 1500, easing: 'ease-out' },
      }
    case 'leave':
      return {
        keyframes: [
          { opacity: 1, transform: 'translateY(0) rotate(0deg)' },
          { opacity: 0, transform: doodle ? 'translateY(-4px) rotate(-3deg)' : 'translateY(-4px)' },
        ],
        timing: { duration: MOTION.base, easing: MOTION.exit },
      }
  }
}

/** slide is a part pushed aside by the change: from its old place to its new one (FLIP). */
export function slide(delta: Delta, look: Look): Movement | null {
  if (look.level !== 'full' || (Math.abs(delta.x) < 1 && Math.abs(delta.y) < 1)) return null
  return {
    keyframes: [{ transform: t(delta.x, delta.y) }, { transform: t(0, 0) }],
    timing: { duration: MOTION.base, easing: MOTION.ease },
  }
}

/** pen is a line drawn as with a pen: its dash runs from hidden to whole. */
export function pen(length: number, look: Look, delay = 0, duration = 240): Movement | null {
  if (look.level !== 'full' || look.style === 'neat' || !(length > 0)) return null
  return {
    keyframes: [
      { strokeDasharray: `${length}`, strokeDashoffset: `${length}` },
      { strokeDasharray: `${length}`, strokeDashoffset: '0' },
    ],
    timing: { duration, delay, easing: MOTION.ease, fill: 'backwards' },
  }
}

/** levelOf is the motion the window plays: the app's setting first, then the system's. */
export function levelOf(setting: string | undefined, systemReduced: boolean): Level {
  if (setting === 'off') return 'off'
  if (setting === 'reduce' || systemReduced) return 'reduce'
  return 'full'
}
