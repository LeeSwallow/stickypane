// The player: it measures the window's parts before a change is drawn,
// and after it moves them as the plan says (plan.ts), with the keyframes
// of keyframes.ts. Parts are found by their data attributes: a note's
// frame by data-frame, a card by data-card, a checklist item by
// data-item, a line drawn as with a pen by data-pen, log lines by .line.
// A part not on screen is skipped. A new change finishes what is still
// moving first (04 §4.4: nothing piles up).

import { movement, pen, slide, type Delta, type Look, type Movement } from './keyframes'
import type { Plan, Step, Target } from './plan'

/** Where every part was, and copies of the parts about to go. */
export interface Snapshot {
  rects: Map<string, DOMRect>
  ghosts: Map<string, HTMLElement>
}

const frameKey = (frame: string) => `f\u0000${frame}`
const cardKey = (frame: string, card: string) => `c\u0000${frame}\u0000${card}`
const itemKey = (frame: string, item: string) => `i\u0000${frame}\u0000${item}`

/** keyOf names a part the way a snapshot does; lines have none. */
export function keyOf(t: Target): string | null {
  switch (t.part) {
    case 'note':
      return frameKey(t.frame)
    case 'card':
      return cardKey(t.frame, t.card)
    case 'item':
      return itemKey(t.frame, t.item)
    default:
      return null
  }
}

function frames(root: ParentNode): HTMLElement[] {
  return [...root.querySelectorAll<HTMLElement>('[data-frame]')]
}

function parts(root: ParentNode): Map<string, HTMLElement> {
  const out = new Map<string, HTMLElement>()
  for (const f of frames(root)) {
    const name = f.dataset.frame ?? ''
    out.set(frameKey(name), f)
    for (const c of f.querySelectorAll<HTMLElement>('[data-card]')) out.set(cardKey(name, c.dataset.card ?? ''), c)
    for (const i of f.querySelectorAll<HTMLElement>('[data-item]')) out.set(itemKey(name, i.dataset.item ?? ''), i)
  }
  return out
}

/**
 * snapshot measures every part under root. The parts named in leaving are
 * copied too, so they can be seen going after the change took them away.
 */
export function snapshot(root: ParentNode, leaving: Set<string> = new Set()): Snapshot {
  const rects = new Map<string, DOMRect>()
  const ghosts = new Map<string, HTMLElement>()
  for (const [k, el] of parts(root)) {
    rects.set(k, el.getBoundingClientRect())
    if (leaving.has(k)) ghosts.set(k, el.cloneNode(true) as HTMLElement)
  }
  return { rects, ghosts }
}

function find(root: ParentNode, t: Target): HTMLElement[] {
  const frame = frames(root).find((f) => f.dataset.frame === t.frame)
  if (!frame) return []
  switch (t.part) {
    case 'note':
      return [frame]
    case 'card':
      return [...frame.querySelectorAll<HTMLElement>('[data-card]')].filter((c) => c.dataset.card === t.card).slice(0, 1)
    case 'item':
      return [...frame.querySelectorAll<HTMLElement>('[data-item]')].filter((c) => c.dataset.item === t.item).slice(0, 1)
    case 'lines':
      return [...frame.querySelectorAll<HTMLElement>('.log .line')].slice(-t.count)
  }
}

const deltaOf = (was: DOMRect | undefined, now: DOMRect): Delta | null =>
  was ? { x: was.left - now.left, y: was.top - now.top } : null

export interface PlayOptions {
  look: Look
  /** an agent's colour, for the highlight of the card it moved */
  colorOf?: (who: string) => string | undefined
  /** where copies of parts that went are shown; document.body by default */
  layer?: HTMLElement
}

export class Player {
  private running: Animation[] = []
  private extras: Element[] = []
  private turn = 0

  /** finish ends every movement at its end state and takes away what it added. */
  finish(): void {
    this.turn++
    for (const a of this.running.splice(0)) {
      try {
        a.finish()
      } catch {
        a.cancel()
      }
    }
    for (const e of this.extras.splice(0)) e.remove()
  }

  /** busy reports whether something is still moving. */
  get busy(): boolean {
    return this.running.length > 0
  }

  private run(el: Element, m: Movement | null, delay = 0): Animation | null {
    if (!m || typeof (el as HTMLElement).animate !== 'function') return null
    const a = el.animate(m.keyframes, { fill: 'backwards', ...m.timing, delay: delay + Number(m.timing.delay ?? 0) })
    this.running.push(a)
    a.addEventListener('finish', () => (this.running = this.running.filter((x) => x !== a)))
    a.addEventListener('cancel', () => (this.running = this.running.filter((x) => x !== a)))
    return a
  }

  /** extra adds a short-lived element (a glow, a copy) and takes it away when its animation ends. */
  private extra(el: HTMLElement, a: Animation | null) {
    this.extras.push(el)
    const gone = () => {
      el.remove()
      this.extras = this.extras.filter((x) => x !== el)
    }
    if (a) {
      a.addEventListener('finish', gone)
      a.addEventListener('cancel', gone)
    } else gone()
  }

  /** play moves the parts under root as the plan says, from where the snapshot saw them. */
  play(root: ParentNode, before: Snapshot, plan: Plan, o: PlayOptions): void {
    this.finish()
    if (o.look.level === 'off') return
    const now = parts(root)
    const targeted = new Set(plan.steps.map((s) => keyOf(s.target)).filter((k): k is string => !!k))

    // FLIP: frames pushed by notes that came, went or changed size, and
    // the cards and items pushed aside in the notes that changed
    const flip = new Set(plan.flip)
    for (const [k, el] of now) {
      if (targeted.has(k)) continue
      const [part, frame] = k.split('\u0000')
      if (part !== 'f' && !flip.has(frame)) continue
      const d = deltaOf(before.rects.get(k), el.getBoundingClientRect())
      if (d) this.run(el, slide(d, o.look))
    }

    for (const s of plan.steps) this.step(root, before, now, s, o)
  }

  private step(root: ParentNode, before: Snapshot, now: Map<string, HTMLElement>, s: Step, o: PlayOptions) {
    const key = keyOf(s.target)
    if (s.effect === 'leave') {
      const ghost = key ? before.ghosts.get(key) : undefined
      const was = key ? before.rects.get(key) : undefined
      if (!ghost || !was) return
      Object.assign(ghost.style, {
        position: 'fixed',
        left: `${was.left}px`,
        top: `${was.top}px`,
        width: `${was.width}px`,
        height: `${was.height}px`,
        margin: '0',
        pointerEvents: 'none',
        zIndex: 'var(--z-ring, 50)',
      })
      ghost.setAttribute('aria-hidden', 'true')
      ghost.removeAttribute('data-frame')
      ghost.removeAttribute('data-card')
      ;(o.layer ?? document.body).appendChild(ghost)
      this.extra(ghost, this.run(ghost, movement('leave', o.look), s.delay))
      return
    }
    const els = key ? (now.get(key) ? [now.get(key)!] : []) : find(root, s.target)
    els.forEach((el, i) => {
      if (s.effect === 'glow') return this.glow(el, s.delay, o)
      const d = key ? deltaOf(before.rects.get(key), el.getBoundingClientRect()) : null
      const a = this.run(el, movement(s.effect, o.look, d), s.delay + (s.effect === 'push' ? i * 40 : 0))
      // a carried or dropped part passes over its neighbours
      if (a && (s.effect === 'carry' || s.effect === 'drop')) this.above(el, a)
      if (s.effect === 'tick') this.pens(el, o.look, s.delay + 120, 240)
      if (s.effect === 'nudge') this.pens(el, o.look, s.delay + 80, 440)
      if (s.effect === 'carry') this.glow(el, s.delay + 560, o, s.who ? o.colorOf?.(s.who) : undefined)
    })
  }

  /** above keeps el over the parts around it while a moves it. */
  private above(el: HTMLElement, a: Animation) {
    const was = el.style.zIndex
    el.style.zIndex = '5'
    if (getComputedStyle(el).position === 'static') el.style.position = 'relative'
    const back = () => (el.style.zIndex = was)
    a.addEventListener('finish', back)
    a.addEventListener('cancel', back)
  }

  /** pens draws the lines marked data-pen inside el as with a pen. */
  // A ring is drawn once its parent is measured, a frame after the change:
  // the lines are looked for then, unless a newer change came meanwhile.
  private pens(el: HTMLElement, look: Look, delay: number, duration: number) {
    const turn = this.turn
    const draw = () => {
      if (turn !== this.turn) return
      for (const p of el.querySelectorAll<SVGPathElement>('[data-pen] path')) {
        const len = typeof p.getTotalLength === 'function' ? p.getTotalLength() : 0
        this.run(p, pen(Math.ceil(len), look, 0, duration), delay)
      }
    }
    if (typeof requestAnimationFrame === 'function') requestAnimationFrame(() => requestAnimationFrame(draw))
    else draw()
  }

  /** glow lays a soft light over el that fades once; under reduced motion a thick outline. */
  private glow(el: HTMLElement, delay: number, o: PlayOptions, color?: string) {
    const m = movement('glow', o.look)
    if (!m) return
    const g = document.createElement('span')
    g.className = o.look.level === 'reduce' ? 'motion-glow outline' : 'motion-glow'
    g.setAttribute('aria-hidden', 'true')
    if (color) g.style.setProperty('--glow', color)
    if (getComputedStyle(el).position === 'static') el.style.position = 'relative'
    el.appendChild(g)
    this.extra(g, this.run(g, m, delay))
  }
}
