// What moves when the board changes (04 §4.3–4.5): given the board shown
// before, the board after and the bridge's board.changed, which parts of
// the window move, how, and when. Pure: no DOM, no clock. The player
// (play.ts) finds the parts by their data attributes and moves them.

import type { Board, ChecklistData, KanbanData, Note } from '../../board/model'

/** One thing that happened in a note, as board.changed (and `stickypane watch --json`) says it. */
export interface ChangeEvent {
  note: string
  kind?: string
  type: string
  item?: string
  from?: string
  to?: string
  time?: string
}

/** The board.changed notification's params (internal/surface/bridge changes). */
export interface Change {
  notes?: Note[]
  removed?: string[]
  events?: ChangeEvent[]
}

/**
 * A part of the window: a note's frame, a card of a board note, an item of
 * a checklist, or the last lines of a log. `frame` is the note whose frame
 * holds it (a book's, for a page).
 */
export type Target =
  | { part: 'note'; frame: string }
  | { part: 'card'; frame: string; card: string }
  | { part: 'item'; frame: string; item: string }
  | { part: 'lines'; frame: string; count: number }

/**
 * How a part moves (04 §4.3):
 * - carry: lifted, carried to its new place, slapped down (an agent moved a card)
 * - slap: lands with a small overshoot (the person's own move, already drawn)
 * - drop: falls from above and sticks (a new card or note)
 * - unfold: opens downwards and fades in (a new item)
 * - push: comes in from below (new log lines)
 * - tick: the tick is drawn with a pen and the row settles
 * - untick: the row fades back
 * - nudge: one wobble and the warning ring drawn (a card that stalled)
 * - glow: a soft light that fades in about 1.5 s (a note that changed)
 * - leave: tilts and fades away (a note or card that went)
 */
export type Effect = 'carry' | 'slap' | 'drop' | 'unfold' | 'push' | 'tick' | 'untick' | 'nudge' | 'glow' | 'leave'

export interface Step {
  target: Target
  effect: Effect
  /** ms after the change is drawn */
  delay: number
  /** the agent a carried card belongs to: its highlight takes their colour */
  who?: string
}

export interface Plan {
  steps: Step[]
  /** notes whose parts slide to their new places (FLIP) */
  flip: string[]
}

/** The spacing and caps of 04 §4.4. */
export const BATCH = {
  /** up to this many events each get their own movement */
  each: 3,
  eachGap: 60,
  /** above that, only card moves do, this far apart, at most this many */
  movesGap: 30,
  moves: 6,
  /** nothing starts later than this */
  cap: 1200,
} as const

/** How long each movement takes (04 §4.3), to keep within the cap. */
export const DURATION: Record<Effect, number> = {
  carry: 620,
  slap: 200,
  drop: 380,
  unfold: 200,
  push: 200,
  tick: 500,
  untick: 120,
  nudge: 520,
  glow: 1500,
  leave: 200,
}

export interface PlanOptions {
  /** the frames on screen; parts elsewhere are left out (and not counted) */
  visible?: Set<string>
}

/** frameOf is the frame that draws a note: its own, or its book's for a page. */
export function frameOf(b: Board | null, name: string): string {
  if (!b) return name
  for (const t of b.tabs)
    for (const n of t.notes) {
      if (n.name === name) return name
      if ((n.pages ?? []).some((p) => p.name === name)) return n.name
    }
  return name
}

function noteIn(b: Board | null, name: string): Note | null {
  if (!b) return null
  for (const t of b.tabs)
    for (const n of t.notes) {
      if (n.name === name) return n
      const p = (n.pages ?? []).find((x) => x.name === name)
      if (p) return p
    }
  return null
}

const columnsOf = (n: Note | null): KanbanData['columns'] => ((n?.data as KanbanData | null)?.columns ?? [])
const itemsOf = (n: Note | null): ChecklistData['items'] => ((n?.data as ChecklistData | null)?.items ?? [])

function cardIn(n: Note | null, text: string): { column: string; stalled: boolean; who: string[] } | null {
  for (const c of columnsOf(n)) {
    const card = c.cards.find((x) => x.text === text)
    if (card) return { column: c.title, stalled: card.stalled, who: card.who }
  }
  return null
}

const same = (a: unknown, b: unknown) => JSON.stringify(a ?? null) === JSON.stringify(b ?? null)

/**
 * planChange says what moves for one board.changed. prev is the board the
 * window showed when the change came (with the person's own guesses
 * already drawn: those do not move again), next the board after it.
 */
export function planChange(prev: Board | null, next: Board, change: Change, o: PlanOptions = {}): Plan {
  const shown = (frame: string) => !o.visible || o.visible.has(frame)
  const events = change.events ?? []
  const created = new Set(events.filter((e) => e.type === 'note.created').map((e) => e.note))
  const removed = new Set([...(change.removed ?? []), ...events.filter((e) => e.type === 'note.removed').map((e) => e.note)])

  // the movements events ask for, in the order they came
  const moves: Step[] = []
  const add = (target: Target, effect: Effect, who?: string) => {
    // a part that went was on screen if the snapshot saw it: the player knows
    if (effect === 'leave' || shown(target.frame)) moves.push({ target, effect, delay: 0, ...(who ? { who } : {}) })
  }
  // log lines arrive together: one push per log
  const lines = new Map<string, number>()

  for (const e of events) {
    const frame = frameOf(e.type === 'note.removed' ? prev : next, e.note)
    const before = noteIn(prev, e.note)
    const after = noteIn(next, e.note)
    switch (e.type) {
      case 'note.created':
        add({ part: 'note', frame }, 'drop')
        break
      case 'note.removed':
        add({ part: 'note', frame }, 'leave')
        break
      case 'card.added':
        if (!created.has(e.note) && e.item && !cardIn(before, e.item)) add({ part: 'card', frame, card: e.item }, 'drop')
        break
      case 'card.moved': {
        if (!e.item) break
        const was = cardIn(before, e.item)
        // already drawn where it went: the person's own move
        if (was && was.column === e.to) add({ part: 'card', frame, card: e.item }, 'slap')
        else add({ part: 'card', frame, card: e.item }, 'carry', cardIn(after, e.item)?.who[0])
        break
      }
      case 'card.removed':
        if (e.item && cardIn(before, e.item)) add({ part: 'card', frame, card: e.item }, 'leave')
        break
      case 'item.ticked': {
        if (!e.item) break
        const was = itemsOf(before).find((i) => i.text === e.item)
        if (!was?.checked) add({ part: 'item', frame, item: e.item }, 'tick')
        break
      }
      case 'item.unticked': {
        if (!e.item) break
        const was = itemsOf(before).find((i) => i.text === e.item)
        if (!was || was.checked) add({ part: 'item', frame, item: e.item }, 'untick')
        break
      }
      case 'item.added':
        if (!created.has(e.note) && e.item && !itemsOf(before).some((i) => i.text === e.item)) add({ part: 'item', frame, item: e.item }, 'unfold')
        break
      case 'log.appended':
        if (!created.has(e.note)) lines.set(frame, (lines.get(frame) ?? 0) + 1)
        break
    }
  }
  for (const [frame, count] of lines) add({ part: 'lines', frame, count }, 'push')

  // a card that has just stalled wobbles once and gets its ring
  for (const n of change.notes ?? []) {
    const before = noteIn(prev, n.name)
    if (!before || n.kind !== 'board') continue
    const frame = frameOf(next, n.name)
    for (const c of columnsOf(n))
      for (const card of c.cards) {
        if (!card.stalled) continue
        const was = cardIn(before, card.text)
        if (was && !was.stalled) add({ part: 'card', frame, card: card.text }, 'nudge')
      }
  }

  // §4.4: up to three each in turn; more, and only card moves, quicker and fewer
  let steps: Step[]
  if (moves.length <= BATCH.each) steps = moves.map((s, i) => ({ ...s, delay: i * BATCH.eachGap }))
  else
    steps = moves
      .filter((s) => s.effect === 'carry')
      .slice(0, BATCH.moves)
      .map((s, i) => ({ ...s, delay: i * BATCH.movesGap }))
  steps = steps.map((s) => ({ ...s, delay: Math.max(0, Math.min(s.delay, BATCH.cap - DURATION[s.effect])) }))

  // a note whose content changed otherwise (not by the person's own guess,
  // not new, not gone, not already moving) glows once
  const glowing = new Set<string>(steps.map((s) => s.target.frame))
  for (const n of change.notes ?? []) {
    if (created.has(n.name) || removed.has(n.name)) continue
    const before = noteIn(prev, n.name)
    if (!before) continue
    if (same(before.data, n.data) && before.title === n.title && same(before.pages?.map((p) => p.data), n.pages?.map((p) => p.data))) continue
    const frame = frameOf(next, n.name)
    if (!shown(frame) || glowing.has(frame)) continue
    glowing.add(frame)
    steps.push({ target: { part: 'note', frame }, effect: 'glow', delay: 0 })
  }

  // parts slide in the notes that changed and, when notes came or went, the grid
  const flip = new Set<string>()
  for (const n of change.notes ?? []) if (!created.has(n.name)) flip.add(frameOf(next, n.name))
  return { steps, flip: [...flip].filter(shown) }
}

/**
 * leaving is the parts a change takes away that are on screen before it is
 * drawn: notes removed (not a book's page: the book stays) and cards
 * removed. They are copied before the change, to be seen going.
 */
export function leaving(prev: Board | null, change: Change): Target[] {
  const out: Target[] = []
  const gone = new Set([...(change.removed ?? []), ...(change.events ?? []).filter((e) => e.type === 'note.removed').map((e) => e.note)])
  for (const name of gone) if (frameOf(prev, name) === name && noteIn(prev, name)) out.push({ part: 'note', frame: name })
  for (const e of change.events ?? [])
    if (e.type === 'card.removed' && e.item && !gone.has(e.note)) out.push({ part: 'card', frame: frameOf(prev, e.note), card: e.item })
  return out
}

/** asChange reads a board.changed's params; anything else is no change. */
export function asChange(data: unknown): Change {
  const d = (data && typeof data === 'object' ? data : {}) as Record<string, unknown>
  return {
    notes: Array.isArray(d.notes) ? (d.notes as Note[]) : [],
    removed: Array.isArray(d.removed) ? (d.removed as string[]).filter((x) => typeof x === 'string') : [],
    events: Array.isArray(d.events) ? (d.events as ChangeEvent[]).filter((e) => e && typeof e.type === 'string' && typeof e.note === 'string') : [],
  }
}
