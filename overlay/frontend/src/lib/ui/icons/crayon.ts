// The crayon tier of the icon set (03 §3.11.5): icons that mean something
// on their own, drawn like the app icon — a thick dark outline round a
// light wash with crayon hatching at -41°, in a note colour. The neat style
// draws them flat: one solid fill and a straight outline, no hatching.
// Everything is seeded from the icon's name, so a redraw never moves a line.
import { seedOf, mix } from '../../theme/tokens'
import type { Style } from '../../doodle/rough'
import { bend } from './icons'

/** The note colours a crayon icon can be filled with (indices into palette.notes). */
export const NOTE = { yellow: 0, pink: 1, blue: 2, green: 3, purple: 4, orange: 5 } as const
type NoteName = keyof typeof NOTE

/** A filled shape (closed path) in a note colour, or a stroke drawn with the outline only. */
type Part = { fill: NoteName; d: string } | { line: string }

const ring = (x: number, y: number, r: number) => `M${x - r} ${y}A${r} ${r} 0 1 0 ${x + r} ${y}A${r} ${r} 0 1 0 ${x - r} ${y}Z`

export const CRAYONS = {
  pin: [
    { fill: 'pink', d: 'M6.75 2.75H13.25L12.1 4.5V8.1L14.9 11.25H5.1L7.9 8.1V4.5Z' },
    { line: 'M10 11.25V17.5' },
  ],
  'tear-off': [
    { fill: 'blue', d: 'M3.25 3.75H16.75V12.25L12.25 16.75H3.25Z' },
    { fill: 'yellow', d: 'M16.75 12.25H12.25V16.75Z' },
  ],
  plus: [
    { fill: 'yellow', d: 'M3.25 3.25H16.75V16.75H3.25Z' },
    { line: 'M10 6.75V13.25' },
    { line: 'M6.75 10H13.25' },
  ],
  run: [{ fill: 'green', d: 'M5.75 3.5L16.25 10L5.75 16.5Z' }],
  trash: [
    { fill: 'pink', d: 'M5 5.5L5.9 16.75H14.1L15 5.5Z' },
    { line: 'M3.25 5.5H16.75' },
    { line: 'M7.75 5.5V3.25H12.25V5.5' },
    { line: 'M8.5 8.5V13.75' },
    { line: 'M11.5 8.5V13.75' },
  ],
  archive: [
    { fill: 'orange', d: 'M4 7.75H16V16.5H4Z' },
    { fill: 'orange', d: 'M2.75 3.5H17.25V7.75H2.75Z' },
    { line: 'M8 10.75H12' },
  ],
  settings: [
    { line: 'M3 6.5H16.5' },
    { line: 'M3.5 13.5H17' },
    { fill: 'purple', d: ring(7, 6.5, 2.4) },
    { fill: 'blue', d: ring(13, 13.5, 2.4) },
  ],
  search: [
    { line: 'M12.75 12.75L16.75 16.75' },
    { fill: 'blue', d: ring(8.5, 8.5, 5.25) },
  ],
  sun: [
    { fill: 'yellow', d: ring(10, 10, 3.75) },
    { line: 'M10 2.25V3.75' },
    { line: 'M10 16.25V17.75' },
    { line: 'M2.25 10H3.75' },
    { line: 'M16.25 10H17.75' },
    { line: 'M4.5 4.5L5.5 5.5' },
    { line: 'M14.5 14.5L15.5 15.5' },
    { line: 'M4.5 15.5L5.5 14.5' },
    { line: 'M14.5 5.5L15.5 4.5' },
  ],
  moon: [{ fill: 'purple', d: 'M16 12.5A6.75 6.75 0 1 1 7.5 4A5.25 5.25 0 0 0 16 12.5Z' }],
} as const satisfies Record<string, readonly Part[]>

export type CrayonName = keyof typeof CRAYONS
export const CRAYON_NAMES = Object.keys(CRAYONS) as CrayonName[]
export const isCrayon = (n: string): n is CrayonName => n in CRAYONS

/** The outline width of a crayon icon in its 20-unit box (the app icon's heavy line). */
export const CRAYON_OUTLINE = 1.9

export interface CrayonColors {
  notes: string[]
  outline: string
  dark: boolean
}

export interface CrayonDrawing {
  /** each filled shape: its path, its wash, and (doodle) its hatching, clipped to the path */
  shapes: { d: string; wash: string; hatch: string[]; hatchColor: string }[]
  /** the outline over everything */
  outline: string[]
}

const f = (n: number) => +n.toFixed(2)

/** hatching is the crayon lines at -41°, a little uneven, across the 20-unit box. */
function hatching(seed: number): string[] {
  const a = (-41 * Math.PI) / 180
  const [dx, dy] = [Math.cos(a), Math.sin(a)]
  const [nx, ny] = [-dy, dx]
  let s = seed >>> 0
  const r = () => {
    s = (Math.imul(s, 1664525) + 1013904223) >>> 0
    return s / 4294967296 - 0.5
  }
  const out: string[] = []
  for (let k = -8; k <= 8; k++) {
    const off = k * 1.9 + r() * 0.5
    const cx = 10 + nx * off
    const cy = 10 + ny * off
    const l = 15 + r() * 2
    out.push(bend(`M${f(cx - dx * l)} ${f(cy - dy * l)}L${f(cx + dx * l)} ${f(cy + dy * l)}`, seed + k * 31))
  }
  return out
}

/** crayonDrawing is a crayon icon in a style and a palette's colours. */
export function crayonDrawing(name: CrayonName, style: Style, c: CrayonColors): CrayonDrawing {
  const seed = seedOf('crayon:' + name)
  const parts: readonly Part[] = CRAYONS[name]
  const shapes: CrayonDrawing['shapes'] = []
  const outline: string[] = []
  parts.forEach((p, i) => {
    const d = 'fill' in p ? p.d : p.line
    outline.push(style === 'neat' ? d : bend(d, seed + i * 97))
    if (!('fill' in p)) return
    const color = c.notes[NOTE[p.fill]] ?? '#cccccc'
    shapes.push({
      d,
      wash: style === 'neat' ? mix(color, '#ffffff', c.dark ? 0.1 : 0.25) : mix(color, '#ffffff', 0.72),
      hatch: style === 'neat' ? [] : hatching(seed + i * 13),
      hatchColor: c.dark ? color : mix(color, '#ffffff', 0.1),
    })
  })
  return { shapes, outline }
}
