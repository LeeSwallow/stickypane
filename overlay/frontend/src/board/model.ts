// What board.load answers (internal/bridge/load.go), as the window reads it,
// and the few pure helpers on it. The window never reads a note's file.

export interface Card {
  text: string
  who: string[]
  /** RFC 3339, when the line says */
  moved?: string
  detail: string[]
  stalled: boolean
}
export interface Column {
  title: string
  cards: Card[]
}
export interface KanbanData {
  desc: string[]
  columns: Column[]
}
export interface Item {
  text: string
  checked: boolean
  /** RFC 3339, when the line says */
  done?: string
}
export interface ChecklistData {
  items: Item[]
  prose: string[]
  done: number
  total: number
}
export interface NoteData {
  markdown: string
}

export interface Note {
  name: string
  tab: string
  title: string
  kind: string
  icon: string
  open: boolean
  /** a fraction of the board (1/4 … 1), or card; old names are read too (layout.ts) */
  size: string
  rows: number
  color: string
  pinned: boolean
  version: string
  summary: string
  data: unknown
  modTime: string
  linked: boolean
  target?: string
  error?: string
  pages?: Note[]
}

export interface Tab {
  name: string
  title: string
  notes: Note[]
}

export interface Settings {
  theme: string
  style: string
  view: string
  language: { choice: string; resolved: string }
  tab: string
  time: string
  editor: string
  stale: string
}

export interface Board {
  title: string
  tabs: Tab[]
  settings: Settings
  /** normalised with normalizePalette before use */
  palette: unknown
}

/** Action is a note.act request: the shape's verb and its arguments, all strings. */
export interface Action {
  name: string
  args?: Record<string, string>
}

/** toggleAction ticks (or unticks) the item at place i of a checklist's file. */
export const toggleAction = (i: number): Action => ({ name: 'toggle', args: { i: String(i) } })

/** ArrangePatch is what note.arrange changes; "" and 0 put a choice back. */
export interface ArrangePatch {
  open?: boolean
  size?: string
  rows?: number
  pin?: boolean
  color?: string
  title?: string
}

/**
 * itemOrder is the order a checklist shows its items in, as places in the
 * file (what `toggle` takes): the open ones first, as written, then the
 * done ones, the most recent first, as the terminal shows them.
 */
export function itemOrder(items: Item[]): number[] {
  const at = (i: number) => {
    const t = Date.parse(items[i].done ?? '')
    return Number.isFinite(t) ? t : -Infinity
  }
  const idx = items.map((_, i) => i)
  return [...idx.filter((i) => !items[i].checked), ...idx.filter((i) => items[i].checked).sort((a, b) => at(b) - at(a))]
}

/** allNotes is every note of the board, book pages included, tab by tab. */
export function allNotes(tabs: Tab[]): Note[] {
  const out: Note[] = []
  for (const t of tabs)
    for (const n of t.notes) {
      out.push(n)
      for (const p of n.pages ?? []) out.push(p)
    }
  return out
}

/** findNote is the note (or book page) of that name, or null. */
export function findNote(tabs: Tab[], name: string): Note | null {
  return allNotes(tabs).find((n) => n.name === name) ?? null
}

/**
 * withArrange is the board with one note re-arranged as note.arrange will
 * do it, so the window redraws at once; the bridge's board.changed then
 * brings the truth. The board given is not changed.
 */
export function withArrange(b: Board, name: string, p: ArrangePatch): Board {
  const apply = (n: Note): Note => {
    if (n.name !== name) return n.pages ? { ...n, pages: n.pages.map((pg) => (pg.name === name ? { ...pg, ...titleOf(p) } : pg)) } : n
    const out = { ...n }
    if (p.open !== undefined) out.open = p.open
    if (p.size !== undefined && p.size !== '') out.size = p.size
    if (p.rows !== undefined) out.rows = p.rows
    if (p.pin !== undefined) out.pinned = p.pin
    if (p.color !== undefined && p.color !== '') out.color = p.color
    if (p.title !== undefined && p.title !== '') out.title = p.title
    return out
  }
  return { ...b, tabs: b.tabs.map((t) => ({ ...t, notes: t.notes.map(apply) })) }
}

const titleOf = (p: ArrangePatch): Partial<Note> => (p.title ? { title: p.title } : {})

const base = (f: string) => f.slice(f.lastIndexOf('/') + 1)
const ext = (f: string) => {
  const b = base(f)
  const i = b.lastIndexOf('.')
  return i > 0 ? b.slice(i) : ''
}

/**
 * resolveLink finds the note a [[link]] names among the board's files, as
 * internal/api.Resolve does: the name itself, with ".md", or the one file
 * whose name without folder and extension it is; "" when none or several.
 */
export function resolveLink(target: string, files: string[]): string {
  target = target.trim()
  for (const f of files) if (f === target || f === target + '.md') return f
  let found = ''
  for (const f of files) {
    const b = base(f)
    if (b.slice(0, b.length - ext(f).length) === target || b === target) {
      if (found) return ''
      found = f
    }
  }
  return found
}

/** linkName is the note part of a link: "todo#item|alias" is "todo". */
export const linkName = (raw: string): string => raw.split('|')[0].split('#')[0].trim()

export type Segment = { md: string } | { embed: string }

const embedLine = /^\s*!\[\[([^\]]+)\]\]\s*$/
const fence = /^\s{0,3}(```|~~~)/

/**
 * splitEmbeds cuts Markdown at the lines that are only ![[name]] (outside
 * code fences): those are drawn as the note they name, one level deep, as
 * the terminal's kinds.WithEmbeds does.
 */
export function splitEmbeds(md: string): Segment[] {
  const out: Segment[] = []
  let buf: string[] = []
  let inFence: string | null = null
  const flush = () => {
    const s = buf.join('\n').replace(/^\n+|\n+$/g, '')
    if (s.trim()) out.push({ md: s })
    buf = []
  }
  for (const line of md.split('\n')) {
    const f = line.match(fence)
    if (f) inFence = inFence === null ? f[1] : inFence === f[1] ? null : inFence
    const m = inFence === null && !f ? line.match(embedLine) : null
    if (m) {
      flush()
      out.push({ embed: linkName(m[1]) })
    } else buf.push(line)
  }
  flush()
  return out
}

/** htmlFor is the HTML rendered for these segments, or none when it was made for others. */
export const htmlFor = (rendered: { segs: Segment[]; html: string[] }, segs: Segment[]): string[] =>
  rendered.segs === segs ? rendered.html : []
