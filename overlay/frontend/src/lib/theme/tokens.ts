// Colour tokens derived from a palette (docs/overlay/03-design-system.md §3.4.1).
import { NOTE_NAMES, type Palette } from './palette'

const bytes = (h: string) => [1, 3, 5].map((i) => parseInt(h.slice(i, i + 2), 16))

/** mix blends a toward b by t (0..1) in sRGB bytes. */
export function mix(a: string, b: string, t: number): string {
  const x = bytes(a)
  const y = bytes(b)
  return '#' + x.map((v, i) => Math.round(v + (y[i] - v) * t).toString(16).padStart(2, '0')).join('')
}

function luminance(h: string): number {
  const [r, g, b] = bytes(h).map((v) => {
    const c = v / 255
    return c <= 0.03928 ? c / 12.92 : ((c + 0.055) / 1.055) ** 2.4
  })
  return 0.2126 * r + 0.7152 * g + 0.0722 * b
}

/** contrast is the WCAG contrast ratio of two "#rrggbb" colours. */
export function contrast(a: string, b: string): number {
  const [hi, lo] = [luminance(a), luminance(b)].sort((x, y) => y - x)
  return (hi + 0.05) / (lo + 0.05)
}

/** seedOf makes a stable positive seed for rough.js from an id (FNV-1a, 32 bits). */
export function seedOf(s: string): number {
  let h = 0x811c9dc5
  for (const b of new TextEncoder().encode(s)) {
    h ^= b
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return (h % 0x7fffffff) + 1
}

/** noteIndexOf is FNV-1a 32 of the name's UTF-8 bytes, mod 6: the Go note colour hash. */
export function noteIndexOf(name: string): number {
  let h = 0x811c9dc5
  for (const b of new TextEncoder().encode(name)) {
    h ^= b
    h = Math.imul(h, 0x01000193) >>> 0
  }
  return h % NOTE_NAMES.length
}

/**
 * agentColor is the colour of an agent name: the note colour its name
 * hashes to, the same function as the bridge's note `color(name, chosen)`.
 * A `chosen` colour name (yellow, pink, ..., case-insensitive) wins before
 * hashing. @owners in the overlay use this note-colour hash; the terminal's
 * NameStyle is a different 4-style hash and is not what this ports.
 */
export function agentColor(name: string, notes: string[], chosen = ''): string {
  const i = NOTE_NAMES.indexOf(chosen.toLowerCase() as (typeof NOTE_NAMES)[number])
  return notes[i >= 0 ? i : noteIndexOf(name) % notes.length]
}

export const noteColors = (p: Palette): string[] => p.notes.slice(0, 6)

export interface CardColors {
  /** the light fill of the card */
  wash: string
  /** the crayon hatching */
  hatch: string
  outline: string
  /** text on the card */
  ink: string
}

/** hatched is the ground text meets on a crayon card: wash with hatch lines over half of it. */
const hatched = (wash: string, hatch: string) => mix(wash, hatch, 0.5)

export const MIN_CONTRAST = 4.5

/**
 * darkenTo mixes ink toward black, as little as it takes, until it reaches
 * the ratio on every ground. The theme's own Text is kept whenever it
 * already does; only light themes with a mid-tone Text (tokyonight-day,
 * catppuccin-latte on some notes) need it.
 */
function darkenTo(ink: string, grounds: string[], ratio: number): string {
  let out = ink
  for (let t = 0; t <= 1.0001; t += 0.02) {
    out = mix(ink, '#000000', t)
    if (grounds.every((g) => contrast(out, g) >= ratio)) return out
  }
  return out
}

const washOf = (color: string, dark: boolean) => mix(color, '#ffffff', dark ? 0.35 : 0.78)
const hatchOf = (color: string, dark: boolean) => (dark ? color : mix(color, '#ffffff', 0.25))

/**
 * cardInk is the text colour on every card of a palette (the --card-ink
 * token): dark themes use Base darkened 40%, light themes use Text; either is
 * darkened toward black just enough to reach 4.5:1 on the wash and on the
 * 50% wash/hatch mix of all six note colours.
 */
export function cardInk(p: Palette, dark: boolean): string {
  const grounds = p.notes.slice(0, 6).flatMap((c) => {
    const wash = washOf(c, dark)
    return [wash, hatched(wash, hatchOf(c, dark))]
  })
  return darkenTo(dark ? mix(p.base, '#000000', 0.4) : p.text, grounds, MIN_CONTRAST)
}

/** cardColorsFor derives a card's colours from a note colour (§3.4.1). */
export function cardColorsFor(color: string, p: Palette, dark: boolean): CardColors {
  return {
    wash: washOf(color, dark),
    hatch: hatchOf(color, dark),
    outline: dark ? mix(p.base, '#000000', 0.55) : p.text,
    ink: cardInk(p, dark),
  }
}

/** cardColors is cardColorsFor note i; hatchSeen is the hatched ground text sits on. */
export const cardColors = (p: Palette, i: number, dark: boolean): CardColors & { hatchSeen: string } => {
  const c = cardColorsFor(p.notes[i], p, dark)
  return { ...c, hatchSeen: hatched(c.wash, c.hatch) }
}

/**
 * reach moves a text colour toward `to`, as little as it takes, until it
 * reaches the ratio on every ground (used for secondary and status text).
 */
function reach(ink: string, to: string, grounds: string[], ratio: number): string {
  for (let t = 0; t <= 1.0001; t += 0.02) {
    const out = mix(ink, to, t)
    if (grounds.every((g) => contrast(out, g) >= ratio)) return out
  }
  return to
}

const rgbOf = (h: string) => bytes(h).join(' ')
const alpha = (h: string, a: number) => `rgb(${rgbOf(h)} / ${a})`

/** inkOn is the text on a filled ground: the theme's base or text, whichever is clearer, if it reaches 4.5:1. */
function inkOn(ground: string, p: Palette): string {
  const by = (cs: string[]) => [...cs].sort((a, b) => contrast(b, ground) - contrast(a, ground))[0]
  const own = by([p.base, p.text])
  return contrast(own, ground) >= MIN_CONTRAST ? own : by(['#ffffff', '#111111'])
}

/** fillFor darkens a colour just enough that text on it reaches 4.5:1 (a filled button). */
function fillFor(c: string, p: Palette): string {
  for (let t = 0; t <= 1.0001; t += 0.02) {
    const g = mix(c, '#000000', t)
    if (contrast(inkOn(g, p), g) >= MIN_CONTRAST) return g
  }
  return c
}

/** The grounds chrome text sits on, for a palette and mode. */
export function chromeGrounds(p: Palette, dark: boolean) {
  const paper = dark ? mix(p.base, '#000000', 0.06) : mix(p.base, '#ffffff', 0.5)
  return {
    paper,
    chrome: dark ? mix(p.base, '#000000', 0.2) : mix(paper, p.text, 0.035),
    surface: dark ? mix(p.base, p.text, 0.07) : mix(p.base, '#ffffff', 0.9),
    sunk: dark ? mix(p.base, '#000000', 0.32) : mix(paper, p.text, 0.06),
  }
}

/**
 * chromeTokens are the colours of the crisp chrome (03 §3.11.2): surfaces,
 * hairlines, hover, the text ramp, and text on filled controls. Every text
 * token reaches 4.5:1 on every ground it is used on.
 */
export function chromeTokens(p: Palette, dark: boolean): Record<string, string> {
  const g = chromeGrounds(p, dark)
  const grounds = [g.paper, g.chrome, g.surface, g.sunk]
  const toward = dark ? p.text : '#000000'
  const tip = dark ? mix(p.base, p.text, 0.16) : mix(p.text, '#000000', 0.15)
  const accentFill = fillFor(p.accent, p)
  const dangerFill = fillFor(p.bad, p)
  return {
    '--chrome': g.chrome,
    '--surface': g.surface,
    '--surface-sunk': g.sunk,
    '--line': mix(p.base, p.text, dark ? 0.13 : 0.12),
    '--line-strong': mix(p.base, p.text, dark ? 0.24 : 0.26),
    '--hover': alpha(p.text, dark ? 0.07 : 0.05),
    '--press': alpha(p.text, dark ? 0.12 : 0.09),
    '--ink-2': reach(p.muted, p.text, grounds, MIN_CONTRAST),
    '--ink-3': p.muted,
    '--accent-text': reach(p.accent, toward, grounds, MIN_CONTRAST),
    '--accent-wash': alpha(p.accent, dark ? 0.16 : 0.1),
    '--accent-fill': accentFill,
    '--accent-ink': inkOn(accentFill, p),
    '--danger-fill': dangerFill,
    '--danger-ink': inkOn(dangerFill, p),
    '--danger-text': reach(p.bad, toward, grounds, MIN_CONTRAST),
    '--danger-wash': alpha(p.bad, dark ? 0.16 : 0.1),
    '--good-text': reach(p.good, toward, grounds, MIN_CONTRAST),
    '--good-wash': alpha(p.good, dark ? 0.16 : 0.12),
    '--warn-text': reach(p.warn, toward, grounds, MIN_CONTRAST),
    '--warn-wash': alpha(p.warn, dark ? 0.16 : 0.12),
    '--tip': tip,
    '--tip-ink': dark ? p.text : reach(p.base, '#ffffff', [tip], MIN_CONTRAST),
    '--focus': p.accent,
    '--scrim': dark ? 'rgb(0 0 0 / .5)' : alpha(p.text, 0.22),
    '--shadow-rgb': dark ? '0 0 0' : rgbOf(mix(p.text, '#000000', 0.4)),
    '--shadow-k': dark ? '2.2' : '1',
  }
}

/**
 * noteTitleColor is a note's title in its own colour, moved toward the ink
 * just enough to read as large text (3:1) on the paper: the yellow of a
 * light theme would otherwise vanish.
 */
export function noteTitleColor(color: string, p: Palette, dark: boolean): string {
  const g = chromeGrounds(p, dark)
  return reach(color, dark ? p.text : '#000000', [g.paper], 3)
}

/** TEXT_PAIRS lists each text token with the grounds it must reach 4.5:1 on (tested). */
export const TEXT_PAIRS: [text: string, grounds: string[]][] = [
  ['--ink', ['--paper', '--chrome', '--surface', '--surface-sunk']],
  ['--ink-2', ['--paper', '--chrome', '--surface', '--surface-sunk']],
  ['--accent-text', ['--paper', '--chrome', '--surface', '--surface-sunk']],
  ['--danger-text', ['--paper', '--chrome', '--surface', '--surface-sunk']],
  ['--good-text', ['--paper', '--chrome', '--surface', '--surface-sunk']],
  ['--warn-text', ['--paper', '--chrome', '--surface', '--surface-sunk']],
  ['--accent-ink', ['--accent-fill']],
  ['--danger-ink', ['--danger-fill']],
  ['--tip-ink', ['--tip']],
]

/** deriveTokens is every CSS variable the screen uses, for a palette. */
export function deriveTokens(p: Palette, dark: boolean): Record<string, string> {
  const t: Record<string, string> = {
    ...chromeTokens(p, dark),
    '--paper': dark ? mix(p.base, '#000000', 0.06) : mix(p.base, '#ffffff', 0.5),
    '--base': p.base,
    '--ink': p.text,
    '--pencil': p.muted,
    '--accent': p.accent,
    '--select': p.select,
    '--good': p.good,
    '--warn': p.warn,
    '--bad': p.bad,
    '--info': p.info,
    '--grid': mix(p.base, p.text, dark ? 0.07 : 0.09),
    '--grid-major': mix(p.base, p.text, dark ? 0.13 : 0.17),
    '--dot': mix(p.base, p.text, dark ? 0.12 : 0.16),
    '--tape': dark ? 'rgba(255,255,255,.13)' : 'rgba(0,0,0,.08)',
    '--card-ink': cardInk(p, dark),
    '--outline': dark ? mix(p.base, '#000000', 0.55) : p.text,
  }
  NOTE_NAMES.forEach((n, i) => (t['--' + n] = p.notes[i]))
  return t
}

/** applyTheme writes the tokens onto the root element. */
export function applyTheme(p: Palette, dark: boolean, el: HTMLElement = document.documentElement): void {
  for (const [k, v] of Object.entries(deriveTokens(p, dark))) el.style.setProperty(k, v)
  el.dataset.mode = dark ? 'dark' : 'light'
  el.style.colorScheme = dark ? 'dark' : 'light'
}
