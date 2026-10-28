// The launch intro, "꾸물꾸물" (04 §4.3.1): when the app's first window of
// the day shows, the icon's three sheets drop in one after another and
// wriggle, fly to the logo's place, and the notes give one small wriggle
// each. Later windows that day, and a window called back, only wriggle the
// notes (250 ms). Reduced motion is a 120 ms fade; motion off is nothing.
// Pure: when, which, and the keyframes. Intro.svelte plays them.

import { MOTION } from '../theme/scale'
import type { Level } from './keyframes'

export type IntroMode = 'full' | 'short' | 'fade' | 'none'

/** Where the day the intro was last played is kept (localStorage, per computer). */
export const INTRO_KEY = 'stickypane.intro'

/** dayOf is the local calendar day, as the intro remembers it. */
export const dayOf = (d: Date): string =>
  `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')}`

/** introMode says what a window plays when it first shows. */
export function introMode(level: Level, today: string, last: string | null): IntroMode {
  if (level === 'off') return 'none'
  if (level === 'reduce') return 'fade'
  return last === today ? 'short' : 'full'
}

/** The times of the full intro, in ms (04 §4.3.1); all of it ends by `end`, under 1.5 s. */
export const INTRO = {
  /** each sheet drops this long after the one before */
  stagger: 80,
  /** the sheets wriggle until here, then fly */
  settle: 450,
  fly: 200,
  /** the outlines are drawn again at these times */
  boil: [0, 150, 300],
  /** then the notes, 30 ms apart, at most 8 wriggle, the rest fade */
  notesAt: 650,
  notesGap: 30,
  notesMax: 8,
  note: 250,
  end: 900,
} as const

export interface Timed {
  keyframes: Keyframe[]
  timing: KeyframeAnimationOptions
}

/**
 * sheet is how sheet i (0 blue, 1 pink, 2 yellow) drops and wriggles:
 * from above, tilted, to its place, then −3° ↔ +2° twice. Neat drops it
 * and fades it in, without the wriggle.
 */
export function sheet(i: number, neat: boolean): Timed {
  const delay = i * INTRO.stagger
  const duration = INTRO.settle - delay
  if (neat)
    return {
      keyframes: [
        { opacity: 0, transform: 'translateY(-24px)' },
        { opacity: 1, transform: 'translateY(0)', offset: 0.5 },
        { opacity: 1, transform: 'translateY(0)' },
      ],
      timing: { delay, duration, easing: MOTION.ease, fill: 'both' },
    }
  return {
    keyframes: [
      { opacity: 0, transform: 'translateY(-30px) rotate(6deg) scale(1.08)', offset: 0 },
      { opacity: 1, transform: 'translateY(0) rotate(0deg) scale(1)', offset: 0.35, easing: MOTION.ease },
      { transform: 'rotate(-3deg)', offset: 0.5 },
      { transform: 'rotate(2deg)', offset: 0.65 },
      { transform: 'rotate(-3deg)', offset: 0.8 },
      { transform: 'rotate(2deg)', offset: 0.9 },
      { opacity: 1, transform: 'rotate(0deg)', offset: 1 },
    ],
    timing: { delay, duration, easing: MOTION.slap, fill: 'both' },
  }
}

/** fly takes the three sheets from the window's middle to the logo's place, shrinking to it. */
export function fly(dx: number, dy: number, scale: number): Timed {
  return {
    keyframes: [
      { transform: 'translate(0px, 0px) scale(1)', opacity: 1 },
      { transform: `translate(${dx}px, ${dy}px) scale(${scale})`, opacity: 0.2 },
    ],
    timing: { delay: INTRO.settle, duration: INTRO.fly, easing: MOTION.ease, fill: 'both' },
  }
}

/**
 * wriggle is note i's one small wriggle (±1.5°, 0.98 → 1) as the board
 * appears, from `at`; after the eighth, a note only fades in.
 */
export function wriggle(i: number, at: number, neat: boolean): Timed {
  const delay = at + Math.min(i, INTRO.notesMax) * INTRO.notesGap
  if (i >= INTRO.notesMax || neat)
    return { keyframes: [{ opacity: 0 }, { opacity: 1 }], timing: { delay, duration: INTRO.note, easing: MOTION.ease, fill: 'backwards' } }
  return {
    keyframes: [
      { opacity: 0, transform: 'scale(0.98) rotate(0deg)' },
      { opacity: 1, transform: 'scale(0.99) rotate(-1.5deg)', offset: 0.35 },
      { transform: 'scale(1) rotate(1.5deg)', offset: 0.7 },
      { opacity: 1, transform: 'scale(1) rotate(0deg)' },
    ],
    timing: { delay, duration: INTRO.note, easing: MOTION.ease, fill: 'backwards' },
  }
}

/** fadeIn is the reduced intro: the window's content comes up by opacity alone. */
export const fadeIn = (): Timed => ({ keyframes: [{ opacity: 0 }, { opacity: 1 }], timing: { duration: MOTION.short, easing: 'linear' } })
