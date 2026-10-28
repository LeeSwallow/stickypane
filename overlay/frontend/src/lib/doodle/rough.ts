// Drawing the hand-made parts as SVG path lists. rough.js makes the doodle
// paths (seeded, so a drawing never changes between renders); the neat style
// builds its own straight segments, so "roughness 0" is exactly straight.
import rough from 'roughjs'
import type { Options } from 'roughjs/bin/core'

export type Style = 'doodle' | 'neat'

export type Role = 'wash' | 'hatch' | 'outline' | 'stroke' | 'fill'

export interface Seg {
  role: Role
  d: string
  stroke: string
  strokeWidth: number
  fill: string
  opacity?: number
}

const gen = rough.generator()

function fromRough(drawable: ReturnType<typeof gen.line>, role: Role, opacity?: number): Seg[] {
  return gen.toPaths(drawable).map((p) => ({
    role,
    d: p.d,
    stroke: p.stroke,
    strokeWidth: p.strokeWidth,
    fill: p.fill ?? 'none',
    opacity,
  }))
}

const f = (n: number) => +n.toFixed(2)
const rectD = (x: number, y: number, w: number, h: number) =>
  `M${f(x)} ${f(y)}L${f(x + w)} ${f(y)}L${f(x + w)} ${f(y + h)}L${f(x)} ${f(y + h)}Z`

interface Common {
  style: Style
  seed: number
}

export interface BoxOpts extends Common {
  stroke: string
  strokeWidth?: number
  fill?: string
  roughness?: number
  bowing?: number
}

/** boxPaths is a hand-drawn (or straight) rectangle outline, inset 2px, with an optional solid fill. */
export function boxPaths(w: number, h: number, o: BoxOpts): Seg[] {
  const sw = o.strokeWidth ?? (o.style === 'neat' ? 1.2 : 1.4)
  if (o.style === 'neat') {
    const segs: Seg[] = []
    if (o.fill) segs.push({ role: 'fill', d: rectD(2, 2, w - 4, h - 4), stroke: 'none', strokeWidth: 0, fill: o.fill })
    segs.push({ role: 'stroke', d: rectD(2, 2, w - 4, h - 4), stroke: o.stroke, strokeWidth: sw, fill: 'none' })
    return segs
  }
  const opt: Options = { stroke: o.stroke, strokeWidth: sw, roughness: o.roughness ?? 1.3, bowing: o.bowing ?? 1.2, seed: o.seed }
  if (o.fill) {
    opt.fill = o.fill
    opt.fillStyle = 'solid'
  }
  return fromRough(gen.rectangle(2, 2, w - 4, h - 4, opt), 'stroke')
}

export interface CrayonOpts extends Common {
  wash: string
  hatch: string
  outline: string
}

/** crayonPaths fills a card the way the icon is drawn: a light wash, crayon hatching at -41 degrees, a thick outline. */
export function crayonPaths(w: number, h: number, o: CrayonOpts): Seg[] {
  const [x, y, bw, bh] = [2, 2, w - 4, h - 4]
  if (o.style === 'neat') {
    return [
      { role: 'wash', d: rectD(x, y, bw, bh), stroke: 'none', strokeWidth: 0, fill: o.wash },
      { role: 'outline', d: rectD(x, y, bw, bh), stroke: o.outline, strokeWidth: 1.2, fill: 'none' },
    ]
  }
  return [
    ...fromRough(gen.rectangle(x, y, bw, bh, { stroke: 'none', fill: o.wash, fillStyle: 'solid', roughness: 1, seed: o.seed }), 'wash'),
    ...fromRough(
      gen.rectangle(x, y, bw, bh, {
        stroke: 'none', fill: o.hatch, fillStyle: 'hachure', hachureAngle: -41, hachureGap: 2.6, fillWeight: 1.5,
        roughness: 2.2, seed: o.seed + 1,
      }),
      'hatch',
    ),
    ...fromRough(gen.rectangle(x, y, bw, bh, { stroke: o.outline, strokeWidth: 2.4, roughness: 1.1, bowing: 1, seed: o.seed + 2 }), 'outline'),
  ]
}

/** crayonLikeRect is one hatched square with a thick outline at x,y: a sheet of the logo. */
export function crayonLikeRect(
  x: number,
  y: number,
  w: number,
  h: number,
  o: Common & { color: string; outline: string },
): Seg[] {
  if (o.style === 'neat') {
    return [
      { role: 'wash', d: rectD(x, y, w, h), stroke: o.outline, strokeWidth: 2, fill: o.color },
    ]
  }
  return fromRough(
    gen.rectangle(x, y, w, h, {
      fill: o.color, fillStyle: 'hachure', hachureAngle: -41, hachureGap: 1.8, fillWeight: 1.2,
      stroke: o.outline, strokeWidth: 2, roughness: 1.2, seed: o.seed,
    }),
    'outline',
  )
}

export interface LineOpts extends Common {
  stroke: string
  strokeWidth?: number
  roughness?: number
}

/** linePaths is a hand-drawn (or straight) line. */
export function linePaths(x1: number, y1: number, x2: number, y2: number, o: LineOpts): Seg[] {
  const sw = o.strokeWidth ?? 1.6
  if (o.style === 'neat') {
    return [{ role: 'stroke', d: `M${f(x1)} ${f(y1)}L${f(x2)} ${f(y2)}`, stroke: o.stroke, strokeWidth: sw, fill: 'none' }]
  }
  return fromRough(gen.line(x1, y1, x2, y2, { stroke: o.stroke, strokeWidth: sw, roughness: o.roughness ?? 1.2, seed: o.seed }), 'stroke')
}

/** polyline is straight segments through points, for the tick and the arrow head. */
export function polylinePaths(pts: [number, number][], o: LineOpts): Seg[] {
  const sw = o.strokeWidth ?? 1.8
  if (o.style === 'neat') {
    const d = pts.map(([x, y], i) => `${i ? 'L' : 'M'}${f(x)} ${f(y)}`).join('')
    return [{ role: 'stroke', d, stroke: o.stroke, strokeWidth: sw, fill: 'none' }]
  }
  return fromRough(gen.linearPath(pts, { stroke: o.stroke, strokeWidth: sw, roughness: o.roughness ?? 1.1, seed: o.seed }), 'stroke')
}

export interface EllipseOpts extends Common {
  stroke: string
  strokeWidth?: number
  fill?: string
  roughness?: number
}

/** ellipsePaths is a hand-drawn ellipse (a circle when w == h); neat draws it as a true ellipse. */
export function ellipsePaths(cx: number, cy: number, w: number, h: number, o: EllipseOpts): Seg[] {
  const sw = o.strokeWidth ?? 1.4
  if (o.style === 'neat') {
    const rx = w / 2
    const ry = h / 2
    const d = `M${f(cx - rx)} ${f(cy)}A${f(rx)} ${f(ry)} 0 1 0 ${f(cx + rx)} ${f(cy)}A${f(rx)} ${f(ry)} 0 1 0 ${f(cx - rx)} ${f(cy)}Z`
    return [{ role: 'stroke', d, stroke: o.fill ? 'none' : o.stroke, strokeWidth: sw, fill: o.fill ?? 'none' }]
  }
  const opt: Options = { stroke: o.stroke, strokeWidth: sw, roughness: o.roughness ?? 1.3, seed: o.seed }
  if (o.fill) {
    opt.fill = o.fill
    opt.fillStyle = 'solid'
  }
  return fromRough(gen.ellipse(cx, cy, w, h, opt), 'stroke')
}

export interface MarkOpts extends Common {
  color: string
}

/** markPaths is a highlighter stroke over the lower half of a w x h text box. */
export function markPaths(w: number, h: number, o: MarkOpts): Seg[] {
  if (o.style === 'neat') {
    return [{ role: 'fill', d: rectD(0, h * 0.45, w, h * 0.5), stroke: 'none', strokeWidth: 0, fill: o.color, opacity: 0.4 }]
  }
  return fromRough(
    gen.rectangle(0, h * 0.45, w, h * 0.5, {
      stroke: 'none', fill: o.color, fillStyle: 'zigzag', hachureGap: 2.5, fillWeight: 3, roughness: 2, seed: o.seed,
    }),
    'fill',
    0.4,
  )
}
