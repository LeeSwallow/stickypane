// The size side of the design system (03 §3.11): type, space, radii,
// strokes, elevation, layers, focus, motion and density. Colours are in
// tokens.ts; this file has no colour of its own (shadows take theirs from
// --shadow-rgb). Every number here becomes a CSS variable via scaleVars().

import type { Style } from '../doodle/rough'

/** A type style: font size and line height in px, and weight. */
export interface TypeStyle {
  size: number
  line: number
  weight: number
}

/**
 * TYPE is the type scale for each family. hand (Gaegu) is for names and
 * headings only and is set 3–5px larger than body to read at the same
 * height (03 §3.5); body is Pretendard; mono is JetBrains Mono.
 */
export const TYPE = {
  hand: {
    display: { size: 30, line: 36, weight: 700 },
    title: { size: 22, line: 28, weight: 700 },
    label: { size: 18, line: 24, weight: 700 },
  },
  body: {
    lg: { size: 15, line: 22, weight: 400 },
    md: { size: 13, line: 20, weight: 400 },
    sm: { size: 12, line: 16, weight: 400 },
    xs: { size: 11, line: 14, weight: 500 },
  },
  mono: {
    md: { size: 12, line: 18, weight: 400 },
    sm: { size: 11, line: 16, weight: 400 },
  },
} as const satisfies Record<string, Record<string, TypeStyle>>

export type Family = keyof typeof TYPE

/** SPACE is the 4px grid: space-1 … space-10. */
export const SPACE = [4, 8, 12, 16, 20, 24, 32, 40, 48, 64] as const

/** RADIUS: controls are softly rounded; notes are paper (square); the window is 16 (03 §3.6). */
export const RADIUS = { none: 0, sm: 4, md: 6, lg: 10, xl: 12, window: 16, pill: 999 } as const

/**
 * STROKE widths. hairline: dividers and control borders; ui: icons and
 * focus-free outlines; note: a note's border; emphasis: a stalled ring,
 * a card outline, a pressed pin.
 */
export const STROKE = { hairline: 1, ui: 1.75, note: 2, emphasis: 2.4 } as const

/**
 * ROUGH is how rough each hand-drawn thing may be in the doodle style.
 * Neat is always 0. Notes are calmer than the small marks, so a page of
 * notes reads as a page, not as noise.
 */
export const ROUGH = { note: 0.8, card: 1.1, check: 1, mark: 1.6 } as const

/** MAX_TILT is the most a note or card may lean, in degrees; nothing else leans. */
export const MAX_TILT = 1.5

/**
 * tiltOf is the lean of a note or card with a seed: one of nine steps in
 * [-MAX_TILT, MAX_TILT], and 0 in the neat style. Only things that hold no
 * controls may lean (a card, a sticker); a note's header never does.
 */
export function tiltOf(seed: number, style: Style): number {
  if (style === 'neat') return 0
  return +((((seed % 9) + 9) % 9 - 4) * (MAX_TILT / 4)).toFixed(3)
}

/** ELEVATION, lowest first. Shadows are drawn with the theme's --shadow-rgb. */
export const ELEVATION = {
  flat: 'none',
  taped: '0 1px 0 rgb(var(--shadow-rgb) / calc(.06 * var(--shadow-k))), 0 2px 6px -2px rgb(var(--shadow-rgb) / calc(.14 * var(--shadow-k)))',
  lifted: '0 2px 4px rgb(var(--shadow-rgb) / calc(.10 * var(--shadow-k))), 0 12px 28px -6px rgb(var(--shadow-rgb) / calc(.28 * var(--shadow-k)))',
  floating: '0 0 0 1px rgb(var(--shadow-rgb) / calc(.10 * var(--shadow-k))), 0 24px 64px -12px rgb(var(--shadow-rgb) / calc(.42 * var(--shadow-k)))',
} as const

/** Z layers, lowest first. */
export const Z = {
  paper: 0,
  note: 1,
  tape: 2,
  ring: 3,
  chrome: 10,
  menu: 100,
  tooltip: 200,
  toast: 300,
  scrim: 390,
  dialog: 400,
} as const

/** FOCUS is the keyboard focus ring: never shown for a mouse click (:focus-visible). */
export const FOCUS = { width: 2, offset: 2 } as const

/** MOTION is the motion tokens of 04 §4.2, in ms and cubic-béziers. */
export const MOTION = {
  short: 120,
  base: 200,
  long: 320,
  carry: 620,
  reduced: 180,
  ease: 'cubic-bezier(.2,.8,.2,1)',
  slap: 'cubic-bezier(.34,1.56,.64,1)',
  exit: 'cubic-bezier(.4,0,1,1)',
} as const

export type Density = 'comfortable' | 'compact'

/** DENSITY: control heights (sm, md, lg), the title bar and a row, per density. */
export const DENSITY = {
  comfortable: { controlSm: 24, controlMd: 28, controlLg: 32, titlebar: 44, row: 32, pad: 12 },
  compact: { controlSm: 20, controlMd: 24, controlLg: 28, titlebar: 36, row: 28, pad: 8 },
} as const satisfies Record<Density, Record<string, number>>

const px = (n: number) => `${n}px`

/** scaleVars is every size CSS variable for a density. */
export function scaleVars(density: Density = 'comfortable'): Record<string, string> {
  const v: Record<string, string> = {
    '--font-hand': "'Gaegu', 'Pretendard Variable', cursive",
    '--font-body': "'Pretendard Variable', system-ui, sans-serif",
    '--font-mono': "'JetBrains Mono', ui-monospace, monospace",
  }
  for (const [fam, styles] of Object.entries(TYPE)) {
    for (const [name, s] of Object.entries(styles) as [string, TypeStyle][]) {
      v[`--text-${fam}-${name}`] = px(s.size)
      v[`--leading-${fam}-${name}`] = px(s.line)
      v[`--weight-${fam}-${name}`] = String(s.weight)
    }
  }
  SPACE.forEach((n, i) => (v[`--space-${i + 1}`] = px(n)))
  for (const [k, n] of Object.entries(RADIUS)) v[`--radius-${k}`] = px(n)
  for (const [k, n] of Object.entries(STROKE)) v[`--stroke-${k}`] = px(n)
  for (const [k, s] of Object.entries(ELEVATION)) v[`--elev-${k}`] = s
  for (const [k, n] of Object.entries(Z)) v[`--z-${k}`] = String(n)
  v['--focus-width'] = px(FOCUS.width)
  v['--focus-offset'] = px(FOCUS.offset)
  v['--t-short'] = `${MOTION.short}ms`
  v['--t-base'] = `${MOTION.base}ms`
  v['--t-long'] = `${MOTION.long}ms`
  v['--t-carry'] = `${MOTION.carry}ms`
  v['--t-reduced'] = `${MOTION.reduced}ms`
  v['--ease'] = MOTION.ease
  v['--slap'] = MOTION.slap
  v['--exit'] = MOTION.exit
  const d = DENSITY[density]
  v['--control-sm'] = px(d.controlSm)
  v['--control-md'] = px(d.controlMd)
  v['--control-lg'] = px(d.controlLg)
  v['--titlebar-h'] = px(d.titlebar)
  v['--row-h'] = px(d.row)
  v['--pad'] = px(d.pad)
  v['--max-tilt'] = `${MAX_TILT}deg`
  return v
}

/** applyScale writes the size variables onto an element (the root, or a subtree for a density). */
export function applyScale(el: HTMLElement, density: Density = 'comfortable'): void {
  for (const [k, v] of Object.entries(scaleVars(density))) el.style.setProperty(k, v)
  el.dataset.density = density
}

/** density is a Svelte action: a subtree drawn compact or comfortable. */
export function density(el: HTMLElement, d: Density = 'comfortable') {
  applyScale(el, d)
  return { update: (n: Density) => applyScale(el, n) }
}
