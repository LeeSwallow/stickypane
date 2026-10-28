// Dev only: the changes the motion page plays, each a board and the
// board.changed a bridge would send for it, over a few notes from the
// shapes' fixture. Nothing here ships.
import type { Board, ChecklistData, KanbanData, Note } from '../board/model'
import type { Change, ChangeEvent } from '../lib/motion/plan'
import { fixture } from '../shapes/fixtures/fixtures'

const pick = (name: string): Note => structuredClone(fixture.tabs.flatMap((t) => t.notes).find((n) => n.name === name)!)
const at = '2026-10-29T03:20:00+09:00'
const card = (text: string, who: string[] = [], stalled = false) => ({ text, who, moved: at, detail: [], stalled })

/** start is the board the motion page opens on. */
export function start(): Board {
  const work = pick('work.md')
  work.data = {
    desc: [],
    columns: [
      { title: 'Todo', cards: [card('docs', ['codex']), card('release notes')] },
      { title: 'Doing', cards: [card('login API', ['claude']), card('rate limit', ['codex'])] },
      { title: 'Done', cards: [card('schema', ['claude'])] },
    ],
  } satisfies KanbanData
  const notes = [work, pick('plan.md'), pick('build.log'), pick('readme.md')].map((n) => ({ ...n, open: true, tab: '' }))
  return { ...fixture, title: 'motion', tabs: [{ name: '', title: 'motion', notes }], settings: { ...fixture.settings, tab: '' } }
}

export type Scene = (b: Board) => { board: Board; change: Change }

const note = (b: Board, name: string) => b.tabs[0].notes.find((n) => n.name === name)!
const ev = (type: string, n: string, extra: Partial<ChangeEvent> = {}): ChangeEvent => ({ note: n, type, kind: '', time: at, ...extra })

/** withNote is the board with one note replaced (or added at the end, or taken away with null). */
function withNote(b: Board, name: string, n: Note | null): Board {
  const notes = b.tabs[0].notes
  const has = notes.some((x) => x.name === name)
  const next = n === null ? notes.filter((x) => x.name !== name) : has ? notes.map((x) => (x.name === name ? n : x)) : [...notes, n]
  return { ...b, tabs: [{ ...b.tabs[0], notes: next }] }
}

function kanban(b: Board, f: (cols: KanbanData['columns']) => void): Note {
  const n = structuredClone(note(b, 'work.md'))
  f((n.data as KanbanData).columns)
  return { ...n, version: n.version + "'" }
}

function moveCard(b: Board, text: string, to: string): { n: Note; e: ChangeEvent } {
  let from = ''
  const n = kanban(b, (cols) => {
    for (const c of cols) {
      const i = c.cards.findIndex((x) => x.text === text)
      if (i < 0) continue
      from = c.title
      const [cd] = c.cards.splice(i, 1)
      cols.find((x) => x.title === to)!.cards.push({ ...cd, moved: at })
      return
    }
  })
  return { n, e: ev('card.moved', 'work.md', { item: text, from, to }) }
}

export const SCENES: Record<string, Scene> = {
  // an agent moves a card to another column: lifted, carried, slapped down
  move(b) {
    const { n, e } = moveCard(b, 'login API', 'Done')
    return { board: withNote(b, 'work.md', n), change: { notes: [n], removed: [], events: [e] } }
  },
  // a new card drops into a column
  add(b) {
    const n = kanban(b, (cols) => cols[0].cards.unshift(card('changelog', ['claude'])))
    return { board: withNote(b, 'work.md', n), change: { notes: [n], removed: [], events: [ev('card.added', 'work.md', { item: 'changelog', to: 'Todo' })] } }
  },
  // an item is ticked: drawn with a pen, it settles among the done ones
  tick(b) {
    const n = structuredClone(note(b, 'plan.md'))
    const d = n.data as ChecklistData
    const it = d.items.find((i) => !i.checked)
    if (!it) return { board: b, change: {} }
    it.checked = true
    it.done = at
    d.done++
    n.summary = `${d.done}/${d.total}`
    return { board: withNote(b, 'plan.md', n), change: { notes: [n], removed: [], events: [ev('item.ticked', 'plan.md', { item: it.text })] } }
  },
  // new lines at the end of a log push in from below
  log(b) {
    const n = structuredClone(note(b, 'build.log'))
    const lines = (n.data as { lines: string[] }).lines
    const more = ['$ go test ./...', 'ok  internal/api 0.41s', 'ok  internal/store 0.22s']
    lines.push(...more)
    return { board: withNote(b, 'build.log', n), change: { notes: [n], removed: [], events: more.map((l) => ev('log.appended', 'build.log', { item: l })) } }
  },
  // a card has waited too long: one wobble, and the warning ring drawn
  stall(b) {
    const n = kanban(b, (cols) => (cols[0].cards[0] = { ...cols[0].cards[0], stalled: true, moved: '2026-10-27T10:00:00+09:00' }))
    return { board: withNote(b, 'work.md', n), change: { notes: [n], removed: [], events: [] } }
  },
  // a note's text changed: it glows, then calms
  glow(b) {
    const n = structuredClone(note(b, 'readme.md'))
    n.data = { markdown: (n.data as { markdown: string }).markdown + '\nAnd one more line from the agent.\n' }
    return { board: withNote(b, 'readme.md', n), change: { notes: [n], removed: [], events: [ev('note.changed', 'readme.md')] } }
  },
  // a new note drops onto the board
  note(b) {
    const n: Note = { ...pick('welcome.md'), name: 'handoff.md', title: 'Handoff', open: true, tab: '', size: '1/2', data: { markdown: 'Picked up by **codex** at 03:20.' } }
    return { board: withNote(b, 'handoff.md', n), change: { notes: [n], removed: [], events: [ev('note.created', 'handoff.md')] } }
  },
  // a note goes: it tilts and fades, the others close up
  remove(b) {
    return { board: withNote(b, 'readme.md', null), change: { notes: [], removed: ['readme.md'], events: [ev('note.removed', 'readme.md')] } }
  },
  // many at once (04 §4.4): only the card moves, quicker and capped
  many(b) {
    let cur = b
    const events: ChangeEvent[] = []
    for (const [text, to] of [['docs', 'Doing'], ['release notes', 'Doing'], ['login API', 'Done'], ['rate limit', 'Done'], ['schema', 'Todo']] as const) {
      const { n, e } = moveCard(cur, text, to)
      cur = withNote(cur, 'work.md', n)
      events.push(e)
    }
    const t = SCENES.tick(cur)
    const l = SCENES.log(t.board)
    return {
      board: l.board,
      change: { notes: [note(l.board, 'work.md'), note(l.board, 'plan.md'), note(l.board, 'build.log')], removed: [], events: [...events, ...(t.change.events ?? []), ...(l.change.events ?? [])] },
    }
  },
}
