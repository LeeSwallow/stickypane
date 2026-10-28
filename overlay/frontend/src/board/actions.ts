// What the board window asks of its project's bridge, through the shell's
// ProjectService (overlay/internal/shell/services.go). A torn-off note or
// a canvas uses the same calls.

import { Call } from '@wailsio/runtime'
import { home, project } from '../lib/shell'
import { withArrange, type ArrangePatch, type Board, type Note } from './model'

export interface Status {
  window: number
  path: string
  state: string
  error?: string
  pinned: boolean
  version?: string
}

export type { Action } from './model'
import type { Action } from './model'

export interface BoardApi {
  status(): Promise<Status>
  load(): Promise<Board>
  act(note: string, version: string, action: Action): Promise<unknown>
  arrange(p: { note: string } & ArrangePatch): Promise<unknown>
  order(tab: string, order: string[]): Promise<unknown>
  open(note: string): Promise<unknown>
  pin(on: boolean): Promise<unknown>
  prefs(): Promise<{ hotkey?: string; mode?: string }>
}

/** shellApi calls the shell by name (the generated bindings are not checked in). */
export const shellApi: BoardApi = {
  status: () => Call.ByName(project('Status')) as Promise<Status>,
  load: () => Call.ByName(project('Load')) as Promise<Board>,
  act: (note, version, action) => Call.ByName(project('Act'), { note, version, action }),
  arrange: (p) => Call.ByName(project('Arrange'), p),
  order: (tab, order) => Call.ByName(project('Order'), { tab, order }),
  open: (note) => Call.ByName(project('Open'), note),
  pin: (on) => Call.ByName(project('Pin'), on),
  prefs: () => Call.ByName(home('Prefs')) as Promise<{ hotkey?: string; mode?: string }>,
}

/** Update is how a call redraws the board at once: it is given a change to the board shown. */
export type Update = (patch: (b: Board) => Board) => void

/** arrange redraws the note as note.arrange will leave it, then asks the bridge. */
export async function arrange(api: BoardApi, note: string, p: ArrangePatch, update: Update): Promise<void> {
  update((b) => withArrange(b, note, p))
  await api.arrange({ note, ...p })
}

/** setSize is the ▢ ▭ ▬ buttons and the - = keys. */
export const setSize = (api: BoardApi, note: string, size: string, update: Update) => arrange(api, note, { size }, update)

/** setRows is the bottom edge, dragged (0: as tall as the content). */
export const setRows = (api: BoardApi, note: string, rows: number, update: Update) => arrange(api, note, { rows }, update)

export { toggleAction } from './model'

/**
 * act is note.act with the note's version. A failure (a stale version, a
 * note gone) is told through fail, never thrown; either way the board is
 * read again, so the window draws what the file says now.
 */
export async function act(api: BoardApi, note: Note, a: Action, fail: (why: string) => void, refresh: () => Promise<void>): Promise<void> {
  try {
    await api.act(note.name, note.version, a)
  } catch (e) {
    fail(String(e))
  } finally {
    await refresh()
  }
}
