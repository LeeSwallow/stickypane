// The window's own state (03 §3.7): the tab shown, the note zoomed, the page
// of each book, the way back, and which notes changed since it saw them.
// None of it is saved; the board's files are not touched by any of it.

export interface Stop {
  tab: string
  zoom: string | null
}

export interface View extends Stop {
  /** the page shown of each book, by the book's name */
  pages: Record<string, number>
  /** where Esc and ← go back to, last first */
  back: Stop[]
}

export const initView = (tab: string): View => ({ tab, zoom: null, pages: {}, back: [] })

export const canGoBack = (v: View): boolean => v.zoom !== null || v.back.length > 0

/** zoomIn makes one note fill the window; back returns to where it was. */
export const zoomIn = (v: View, name: string): View =>
  v.zoom === name ? v : { ...v, zoom: name, back: [...v.back, { tab: v.tab, zoom: v.zoom }] }

/** navigate follows a link: the note's tab, the note zoomed, and a way back. */
export const navigate = (v: View, tab: string, name: string): View =>
  v.tab === tab && v.zoom === name ? v : { ...v, tab, zoom: name, back: [...v.back, { tab: v.tab, zoom: v.zoom }] }

/** goBack is Esc, ← and a click outside a zoomed note. */
export function goBack(v: View): View {
  if (v.back.length === 0) return v.zoom === null ? v : { ...v, zoom: null }
  const prev = v.back[v.back.length - 1]
  return { ...v, tab: prev.tab, zoom: prev.zoom, back: v.back.slice(0, -1) }
}

/** selectTab is a click on a tab: the grid of that tab, nothing to go back to. */
export const selectTab = (v: View, tab: string): View => ({ ...v, tab, zoom: null, back: [] })

/**
 * followTab moves the window to the tab the terminal's `tab` setting names
 * when that setting changed (an agent's `show`), and only then: the window
 * keeps a tab of its own and never writes the setting (10 §10.1).
 */
export const followTab = (v: View, before: string, now: string): View => (before === now ? v : selectTab(v, now))

const clamp = (i: number, count: number) => Math.max(0, Math.min(Math.max(0, count - 1), i))

/** pageOf is the page shown of a book of count pages. */
export const pageOf = (v: View, book: string, count: number): number => clamp(v.pages[book] ?? 0, count)

/** flip turns a book's page by delta, stopping at the first and the last. */
export const flip = (v: View, book: string, delta: number, count: number): View => ({
  ...v,
  pages: { ...v.pages, [book]: clamp(pageOf(v, book, count) + delta, count) },
})

/** goToPage shows a book's page i (the page list). */
export const goToPage = (v: View, book: string, i: number, count: number): View => ({
  ...v,
  pages: { ...v.pages, [book]: clamp(i, count) },
})

/** Seen is the modTime of each note when the window last looked at it. */
export type Seen = Record<string, string>

/** seeAll records the notes the window has not met yet as seen; a later change stays marked. */
export function seeAll(seen: Seen, notes: { name: string; modTime: string }[]): Seen {
  let out = seen
  for (const n of notes) if (!(n.name in out)) out = { ...out, [n.name]: n.modTime }
  return out
}

/** isChanged is the `•` beside a title: the file moved on since the window last looked. */
export const isChanged = (seen: Seen, n: { name: string; modTime: string }): boolean =>
  n.name in seen && Date.parse(n.modTime) > Date.parse(seen[n.name])
