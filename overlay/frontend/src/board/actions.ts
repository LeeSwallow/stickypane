// How the board window's calls change what it draws: each takes the
// service (services/board.ts) as an argument, so a test passes a fake.

import { withArrange, type ArrangePatch, type Board, type Note } from './model'
import type { BoardApi } from '../services/board'
import { errorText } from '../services/errors'
import { predict, withData } from './interact'

export type { BoardApi, Status } from '../services/board'
export type { Action } from './model'
import type { Action } from './model'

/** Update is how a call redraws the board at once: it is given a change to the board shown. */
export type Update = (patch: (b: Board) => Board) => void

/** arrange redraws the note as note.arrange will leave it, then asks the bridge. */
export async function arrange(api: BoardApi, note: string, p: ArrangePatch, update: Update): Promise<void> {
  update((b) => withArrange(b, note, p))
  await api.arrange({ note, ...p })
}

/** setSize is the size stepper, the dragged right edge and the - = keys: a fraction (layout.ts). */
export const setSize = (api: BoardApi, note: string, size: string, update: Update) => arrange(api, note, { size }, update)

/** setRows is the bottom edge, dragged (0: as tall as the content). */
export const setRows = (api: BoardApi, note: string, rows: number, update: Update) => arrange(api, note, { rows }, update)

export { toggleAction } from './model'

/**
 * act is note.act with the note's version. With update, a change the
 * window can tell (a tick, a card moved: interact.ts) is drawn at once. A
 * failure (a stale version, a note gone) is told through fail, never
 * thrown; either way the board is read again, so the window draws what the
 * file says now, and a guess the bridge refused is undone.
 */
export async function act(
  api: BoardApi,
  note: Note,
  a: Action,
  fail: (why: string) => void,
  refresh: () => Promise<void>,
  update?: Update,
): Promise<void> {
  const guess = update ? predict(note.kind, note.data, a) : null
  if (guess !== null && update) update((b) => withData(b, note.name, guess))
  try {
    await api.act(note.name, note.version, a)
  } catch (e) {
    fail(errorText(e))
  } finally {
    await refresh()
  }
}
