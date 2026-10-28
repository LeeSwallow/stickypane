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

/** seedOf makes a stable positive seed for rough.js from an id. */
export const seedOf = (s: string): number => [...s].reduce((h, ch) => (h * 31 + ch.charCodeAt(0)) % 9973, 7) + 1

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
 * hashes to, the same function as the terminal's and the bridge's.
 */
export function agentColor(name: string, notes: string[]): string {
  return notes[noteIndexOf(name) % notes.length]
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

/** cardColorsFor derives a card's colours from a note colour (§3.4.1). */
export function cardColorsFor(color: string, p: Palette, dark: boolean): CardColors {
  const wash = mix(color, '#ffffff', dark ? 0.35 : 0.78)
  const hatch = dark ? color : mix(color, '#ffffff', 0.25)
  const ink = dark ? mix(p.base, '#000000', 0.4) : p.text
  return {
    wash,
    hatch,
    outline: dark ? mix(p.base, '#000000', 0.55) : p.text,
    ink: darkenTo(ink, [wash, hatched(wash, hatch)], MIN_CONTRAST),
  }
}

/** cardColors is cardColorsFor note i; hatchSeen is the hatched ground text sits on. */
export const cardColors = (p: Palette, i: number, dark: boolean): CardColors & { hatchSeen: string } => {
  const c = cardColorsFor(p.notes[i], p, dark)
  return { ...c, hatchSeen: hatched(c.wash, c.hatch) }
}

/** deriveTokens is every CSS variable the screen uses, for a palette. */
export function deriveTokens(p: Palette, dark: boolean): Record<string, string> {
  const t: Record<string, string> = {
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
    '--card-ink': dark ? mix(p.base, '#000000', 0.4) : p.text,
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
