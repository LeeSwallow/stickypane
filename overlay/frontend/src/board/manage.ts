// What the ⋯ menu of a note does to the note as a whole (10 §10.3):
// archive, delete with a way back, move, run a script, send a request. Each
// is one call of the notes service; the board is read again after it, and
// what happened is said in a short line (with 되돌리기 after a delete).
// The confirmations are the window's (Board.svelte); these run after them.

import { t } from '../i18n'
import { errorText } from '../services/errors'
import type { NotesApi } from '../services/notes'
import type { Note } from './model'

export interface Said {
  message: string
  action?: { label: string; run: () => void }
}

export interface ManageDeps {
  api: NotesApi
  refresh: () => Promise<void>
  say: (s: Said) => void
  fail: (why: string) => void
}

/** What each item of the menu does to a note of a kind; run and send ask first. */
export type Manage = 'archive' | 'trash' | 'move' | 'run' | 'send'

/** manageItems are the note's own items of the ⋯ menu, by what the note is. */
export function manageItems(note: Note): Manage[] {
  if (note.error) return ['trash']
  const out: Manage[] = []
  if (note.kind === 'script') out.push('run')
  if (note.kind === 'api') out.push('send')
  out.push('move', 'archive', 'trash')
  return out
}

async function attempt(d: ManageDeps, title: string, f: () => Promise<Said | null>): Promise<boolean> {
  try {
    const s = await f()
    if (s) d.say(s)
    return true
  } catch (e) {
    d.fail(t('note.failed', { title, error: errorText(e) }))
    return false
  } finally {
    await d.refresh()
  }
}

export const archive = (d: ManageDeps, n: Note) =>
  attempt(d, n.title, async () => (await d.api.archive(n.name), { message: t('note.archived', { title: n.title }) }))

/** trash deletes the note; the line it leaves brings it back (note.restore). */
export const trash = (d: ManageDeps, n: Note) =>
  attempt(d, n.title, async () => {
    await d.api.trash(n.name)
    return { message: t('note.trashed'), action: { label: t('note.restore'), run: () => void restore(d, n) } }
  })

export const restore = (d: ManageDeps, n: Note) =>
  attempt(d, n.title, async () => (await d.api.restore(n.name), { message: t('note.restored', { title: n.title }) }))

/** move: to is a tab ("work/"), "." for the top level, or a new name. */
export const move = (d: ManageDeps, n: Note, to: string) =>
  attempt(d, n.title, async () => {
    const r = await d.api.move(n.name, to.trim())
    return { message: t('note.moved', { to: r?.note ?? to }) }
  })

export const run = (d: ManageDeps, n: Note) =>
  attempt(d, n.title, async () => {
    const r = await d.api.run(n.name)
    return { message: t(r?.code ? 'run.failed' : 'run.done', { code: r?.code ?? 0, took: r?.took ?? 0 }) }
  })

/** send sends the note's first request, as enter on the terminal does with none chosen. */
export const send = (d: ManageDeps, n: Note) =>
  attempt(d, n.title, async () => (await d.api.send(n.name, 1), { message: t('send.done') }))

/** moveTargets are where a note can go: the top level and every other tab. */
export function moveTargets(tabs: { name: string }[], from: string): string[] {
  const out = from === '' ? [] : ['.']
  for (const x of tabs) if (x.name && x.name !== from) out.push(x.name + '/')
  return out
}
