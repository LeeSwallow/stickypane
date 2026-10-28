// The icon set (03 §3.11.5): one 20px family, stroke 1.75, round caps and
// joins. Each icon is drawn once as straight lines and arcs; the doodle style
// bends every straight line a little (seeded, so an icon never changes) and
// the neat style draws it as written. Arcs and dots are the same in both.
import { seedOf } from '../../theme/tokens'
import type { Style } from '../../doodle/rough'

export const ICON_SIZE = 20
export const ICON_STROKE = 1.75

/** An icon: stroked paths, and optional filled ones (dots, a filled half). */
export interface IconDef {
  d: string[]
  fill?: string[]
}

const dot = (x: number, y: number, r = 1.25) =>
  `M${x - r} ${y}A${r} ${r} 0 1 0 ${x + r} ${y}A${r} ${r} 0 1 0 ${x - r} ${y}Z`
const circle = (x: number, y: number, r: number) => dot(x, y, r)
const frame = 'M3 4.5H17V15.5H3Z'

export const ICONS = {
  pin: { d: ['M7 3.5H13', 'M8.25 3.5V8.25L5.75 11.25H14.25L11.75 8.25V3.5', 'M10 11.25V16.75'] },
  'pin-off': { d: ['M7 3.5H13', 'M8.25 3.5V8.25L5.75 11.25H14.25L11.75 8.25V3.5', 'M10 11.25V16.75', 'M3.5 3.5L16.5 16.5'] },
  'tear-off': { d: ['M8.5 4.5H4V16H15.5V11.5', 'M10 10L16 4', 'M11.5 4H16V8.5'] },
  back: { d: ['M9 4.5L3.75 10L9 15.5', 'M3.75 10H16.25'] },
  close: { d: ['M5 5L15 15', 'M15 5L5 15'] },
  minimise: { d: ['M4.5 10H15.5'] },
  more: { d: [], fill: [dot(4.5, 10), dot(10, 10), dot(15.5, 10)] },
  plus: { d: ['M10 4V16', 'M4 10H16'] },
  'size-card': { d: [frame], fill: ['M5.25 6.75H9.25V10.25H5.25Z'] },
  'size-half': { d: [frame], fill: ['M5.25 6.75H9.25V13.25H5.25Z'] },
  'size-page': { d: [frame], fill: ['M5.25 6.75H14.75V13.25H5.25Z'] },
  // a note's width one step narrower or wider: two arrows to or from a middle line
  narrower: { d: ['M2.75 10H8', 'M5.75 7.5L8.25 10L5.75 12.5', 'M17.25 10H12', 'M14.25 7.5L11.75 10L14.25 12.5', 'M10 5V15'] },
  wider: { d: ['M8.25 10H3', 'M5.5 7.5L3 10L5.5 12.5', 'M11.75 10H17', 'M14.5 7.5L17 10L14.5 12.5', 'M10 5V15'] },
  zoom: { d: ['M11.5 3.75H16.25V8.5', 'M16.25 3.75L11.25 8.75', 'M8.5 16.25H3.75V11.5', 'M3.75 16.25L8.75 11.25'] },
  grid: { d: ['M3.5 3.5H8.75V8.75H3.5Z', 'M11.25 3.5H16.5V8.75H11.25Z', 'M3.5 11.25H8.75V16.5H3.5Z', 'M11.25 11.25H16.5V16.5H11.25Z'] },
  canvas: { d: ['M3.25 3.5H9.25V9.5H3.25Z', 'M11 6.25H16.75V12H11Z', 'M5.75 11.5H11.25V17H5.75Z'] },
  run: { d: ['M6.5 4.25L15.5 10L6.5 15.75Z'] },
  send: { d: ['M3.5 9.25L16.5 3.5L11 16.5L9 11Z', 'M9 11L16.5 3.5'] },
  'edit-external': { d: ['M4 16L4.6 12.75L12.75 4.6L15.4 7.25L7.25 15.4Z', 'M11 6.4L13.6 9'] },
  search: { d: [circle(8.75, 8.75, 5), 'M12.4 12.4L16.25 16.25'] },
  settings: { d: ['M3.5 6.5H5.25', 'M8.75 6.5H16.5', circle(7, 6.5, 1.75), 'M3.5 13.5H11.25', 'M14.75 13.5H16.5', circle(13, 13.5, 1.75)] },
  sun: {
    d: [
      circle(10, 10, 3.25), 'M10 2.75V4.25', 'M10 15.75V17.25', 'M2.75 10H4.25', 'M15.75 10H17.25',
      'M4.85 4.85L5.9 5.9', 'M14.1 14.1L15.15 15.15', 'M4.85 15.15L5.9 14.1', 'M14.1 5.9L15.15 4.85',
    ],
  },
  moon: { d: ['M15.75 12.25A6.5 6.5 0 1 1 7.75 4.25A5 5 0 0 0 15.75 12.25Z'] },
  auto: { d: [circle(10, 10, 6.5)], fill: ['M10 3.5A6.5 6.5 0 0 1 10 16.5Z'] },
  tab: { d: ['M3 15.5V4.5H8L9.5 6.75H17V15.5Z'] },
  'book-prev': { d: ['M12.5 4.75L7.25 10L12.5 15.25'] },
  'book-next': { d: ['M7.5 4.75L12.75 10L7.5 15.25'] },
  archive: { d: ['M3 4H17V7.75H3Z', 'M4.25 7.75V16H15.75V7.75', 'M8 10.75H12'] },
  trash: { d: ['M3.5 5.5H16.5', 'M7.75 5.5V3.5H12.25V5.5', 'M5 5.5L5.85 16.5H14.15L15 5.5', 'M8.5 8.5V13.5', 'M11.5 8.5V13.5'] },
  restore: { d: ['M7 4.25L3.5 7.75L7 11.25', 'M3.5 7.75H12A4.25 4.25 0 0 1 12 16.25H8'] },
  link: { d: ['M8.25 11.75L11.75 8.25', 'M9 6.25L10.5 4.75A3.25 3.25 0 0 1 15.25 9.5L13.75 11', 'M11 13.75L9.5 15.25A3.25 3.25 0 0 1 4.75 10.5L6.25 9'] },
  check: { d: ['M4.25 10.5L8.25 14.25L15.75 5.75'] },
  'chevron-down': { d: ['M5.5 8L10 12.5L14.5 8'] },
  warn: { d: ['M10 3.5L17 16H3Z', 'M10 8.25V11.5'], fill: [dot(10, 13.75, 1)] },
} as const satisfies Record<string, IconDef>

export type IconName = keyof typeof ICONS

export const ICON_NAMES = Object.keys(ICONS) as IconName[]

// --- the doodle bend -------------------------------------------------------

type Cmd = { op: string; args: number[] }

const ARITY: Record<string, number> = { M: 2, L: 2, H: 1, V: 1, A: 7, Q: 4, C: 6, Z: 0 }

/** parse reads the absolute commands the icons are written in. */
export function parse(d: string): Cmd[] {
  const out: Cmd[] = []
  const re = /([MLHVAQCZ])([^MLHVAQCZ]*)/g
  for (const m of d.matchAll(re)) {
    const op = m[1]
    const nums = (m[2].match(/-?\d*\.?\d+(?:e-?\d+)?/g) ?? []).map(Number)
    const n = ARITY[op]
    if (n === 0) out.push({ op, args: [] })
    else for (let i = 0; i < nums.length; i += n) out.push({ op: i && op === 'M' ? 'L' : op, args: nums.slice(i, i + n) })
  }
  return out
}

const f = (n: number) => +n.toFixed(2)

/** rand is a small seeded generator (mulberry32), in [-1, 1]. */
function rand(seed: number) {
  let a = seed >>> 0
  return () => {
    a = (a + 0x6d2b79f5) >>> 0
    let t = a
    t = Math.imul(t ^ (t >>> 15), t | 1)
    t ^= t + Math.imul(t ^ (t >>> 7), t | 61)
    return (((t ^ (t >>> 14)) >>> 0) / 4294967296) * 2 - 1
  }
}

/**
 * bend turns each straight line of a path into a gentle curve: its middle
 * pushed sideways by at most 3% of its length (and never more than 0.45px),
 * so the icon looks drawn with a pen and still reads at 20px.
 */
export function bend(d: string, seed: number): string {
  const r = rand(seed)
  let x = 0
  let y = 0
  let sx = 0
  let sy = 0
  const parts: string[] = []
  const line = (x2: number, y2: number) => {
    const len = Math.hypot(x2 - x, y2 - y)
    if (len < 0.5) {
      parts.push(`L${f(x2)} ${f(y2)}`)
      return
    }
    const off = Math.min(0.45, len * 0.03) * (r() > 0 ? 1 : -1) * (0.6 + 0.4 * Math.abs(r()))
    const nx = -(y2 - y) / len
    const ny = (x2 - x) / len
    const t = 0.5 + r() * 0.12
    parts.push(`Q${f(x + (x2 - x) * t + nx * off)} ${f(y + (y2 - y) * t + ny * off)} ${f(x2)} ${f(y2)}`)
  }
  for (const c of parse(d)) {
    const a = c.args
    switch (c.op) {
      case 'M':
        ;[x, y, sx, sy] = [a[0], a[1], a[0], a[1]]
        parts.push(`M${f(x)} ${f(y)}`)
        continue
      case 'L':
        line(a[0], a[1])
        ;[x, y] = [a[0], a[1]]
        continue
      case 'H':
        line(a[0], y)
        x = a[0]
        continue
      case 'V':
        line(x, a[0])
        y = a[0]
        continue
      case 'Z':
        if (Math.hypot(sx - x, sy - y) > 0.01) line(sx, sy)
        parts.push('Z')
        ;[x, y] = [sx, sy]
        continue
      default:
        parts.push(c.op + a.map(f).join(' '))
        ;[x, y] = [a[a.length - 2], a[a.length - 1]]
    }
  }
  return parts.join('')
}

const cache = new Map<string, { d: string[]; fill: string[] }>()

/** iconPaths is an icon as drawn in a style: neat as written, doodle bent. */
export function iconPaths(name: IconName, style: Style): { d: string[]; fill: string[] } {
  const key = `${style}:${name}`
  const hit = cache.get(key)
  if (hit) return hit
  const def: IconDef = ICONS[name]
  const seed = seedOf(name)
  const out = {
    d: style === 'neat' ? [...def.d] : def.d.map((d, i) => bend(d, seed + i * 101)),
    fill: [...(def.fill ?? [])],
  }
  cache.set(key, out)
  return out
}
