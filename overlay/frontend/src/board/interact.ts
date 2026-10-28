// What a hand on the board does, as pure functions: the note.act a card
// move or a key asks for, and how the window redraws before the bridge
// answers (optimistic). When the bridge refuses (a stale version, a card
// gone) the window reads the board again, which puts the truth back.

import type { Action, Board, ChecklistData, KanbanData, Note } from './model'

/** moveAction moves card i of column col to column to, at place at there (the end when absent). */
export function moveAction(col: number, i: number, to: number, at?: number): Action {
  const args: Record<string, string> = { col: String(col), i: String(i), to: String(to) }
  if (at !== undefined) args.at = String(at)
  return { name: 'move', args }
}

/**
 * dropAt is the move for a card dropped before card `before` of column to
 * (the end when before is null): `at` counts with the card already taken
 * out, as the bridge counts. Null when the card would stay where it is.
 */
export function dropAt(col: number, i: number, to: number, before: number | null, count: number): Action | null {
  if (col === to) {
    const j = before ?? count
    const at = j > i ? j - 1 : j
    return at === i ? null : moveAction(col, i, to, at)
  }
  return moveAction(col, i, to, before ?? undefined)
}

/**
 * keyMove is the move a key asks for on a focused card, as on the
 * terminal: h and l to the column before or after (to its end), j and k a
 * place down or up in its column. Null when it cannot go there.
 */
export function keyMove(data: KanbanData, col: number, i: number, key: string): Action | null {
  const n = data.columns.length
  const len = data.columns[col]?.cards.length ?? 0
  switch (key) {
    case 'h':
    case 'H':
      return col > 0 ? moveAction(col, i, col - 1) : null
    case 'l':
    case 'L':
      return col < n - 1 ? moveAction(col, i, col + 1) : null
    case 'j':
    case 'J':
      return i < len - 1 ? moveAction(col, i, col, i + 1) : null
    case 'k':
    case 'K':
      return i > 0 ? moveAction(col, i, col, i - 1) : null
  }
  return null
}

const num = (a: Action, k: string) => Number(a.args?.[k])

/**
 * predict is a note's data as the action will leave it, for the shapes the
 * window can tell (a ticked item, a moved card); null for the rest, which
 * wait for the bridge.
 */
export function predict(kind: string, data: unknown, a: Action): unknown | null {
  if (kind === 'checklist' && a.name === 'toggle' && data) {
    const d = data as ChecklistData
    const i = num(a, 'i')
    if (!d.items[i]) return null
    const items = d.items.map((it, j) => (j === i ? { ...it, checked: !it.checked, done: it.checked ? undefined : new Date().toISOString() } : it))
    return { ...d, items, done: items.filter((it) => it.checked).length }
  }
  if (kind === 'board' && a.name === 'move' && data) {
    const d = data as KanbanData
    const [col, i, to] = [num(a, 'col'), num(a, 'i'), num(a, 'to')]
    const card = d.columns[col]?.cards[i]
    if (!card || !d.columns[to]) return null
    const columns = d.columns.map((c) => ({ ...c, cards: [...c.cards] }))
    columns[col].cards.splice(i, 1)
    const at = a.args?.at === undefined ? columns[to].cards.length : Math.min(num(a, 'at'), columns[to].cards.length)
    columns[to].cards.splice(at, 0, col === to ? card : { ...card, moved: new Date().toISOString(), stalled: false })
    return { ...d, columns }
  }
  return null
}

/** withData is the board with one note's (or page's) data replaced. */
export function withData(b: Board, name: string, data: unknown): Board {
  const put = (n: Note): Note =>
    n.name === name ? { ...n, data } : n.pages ? { ...n, pages: n.pages.map(put) } : n
  return { ...b, tabs: b.tabs.map((t) => ({ ...t, notes: t.notes.map(put) })) }
}
