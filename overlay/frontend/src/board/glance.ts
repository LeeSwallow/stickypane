// The Glance line (03 §3.6): one tag per @name on the board's cards, with
// the card that name is on now and how long it has been there.

import { allNotes, type Card, type KanbanData, type Tab } from './model'

export interface GlanceTag {
  who: string
  /** the card it is on now; null: resting (only cards in the last column) */
  card: Card | null
  /** the board note that card is on */
  note: string | null
  /** ms since the card moved; null when the file does not say */
  since: number | null
  /** one of its cards stalled (the bridge's call, by the `stale` setting) */
  stalled: boolean
}

const time = (s?: string) => {
  const t = s ? Date.parse(s) : NaN
  return Number.isFinite(t) ? t : -Infinity
}

/**
 * glance lists every name on the cards of every board note, in the order
 * met. Its card now is the most recently moved of its cards that are not
 * in a board's last column; on a tie the later column (Doing over Todo).
 */
export function glance(tabs: Tab[], now: Date): GlanceTag[] {
  const tags = new Map<string, GlanceTag & { at: number; col: number }>()
  for (const n of allNotes(tabs)) {
    if (n.kind !== 'board' || !n.data) continue
    const cols = (n.data as KanbanData).columns ?? []
    cols.forEach((c, ci) => {
      const last = ci === cols.length - 1
      for (const card of c.cards) {
        for (const who of card.who ?? []) {
          let tag = tags.get(who)
          if (!tag) {
            tag = { who, card: null, note: null, since: null, stalled: false, at: -Infinity, col: -1 }
            tags.set(who, tag)
          }
          if (card.stalled) tag.stalled = true
          if (last) continue
          const at = time(card.moved)
          if (tag.card === null || at > tag.at || (at === tag.at && ci > tag.col)) {
            Object.assign(tag, { card, note: n.name, at, col: ci, since: at === -Infinity ? null : now.getTime() - at })
          }
        }
      }
    })
  }
  return [...tags.values()].map(({ at: _a, col: _c, ...t }) => t)
}

/** stalledCount is the number of stalled cards on the board, for "멈춤 1". */
export function stalledCount(tabs: Tab[]): number {
  let n = 0
  for (const note of allNotes(tabs))
    if (note.kind === 'board' && note.data)
      for (const c of (note.data as KanbanData).columns ?? []) n += c.cards.filter((cd) => cd.stalled).length
  return n
}
