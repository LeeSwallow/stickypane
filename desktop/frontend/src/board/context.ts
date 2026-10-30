// What a note's frame and its shape ask of the window around them. The board
// window provides it; a canvas or a torn-off note window provides the same,
// so a note draws and acts the same anywhere.

import type { MenuItem } from '../lib/ui/menu'
import { getContext } from 'svelte'
import type { Action } from './model'
import type { ArrangePatch, Note } from './model'
import type { Manage, Request } from './manage'
import { NOTE_NAMES, type Palette } from '../lib/theme/palette'

export const boardContext = Symbol('stickypane.board')

export interface BoardCtx {
  /** the note a link or an embed names, or null */
  find(link: string): Note | null
  /** follow a [[link]]: that note, zoomed, with a way back */
  navigate(link: string): void
  /** note.act on a note, with its version */
  act(note: Note, action: Action): Promise<void>
  /** note.arrange: open, size, rows, pin, colour, title */
  arrange(note: string, patch: ArrangePatch): Promise<void>
  zoom(name: string): void
  back(): void
  canBack(): boolean
  /** turn a book's page by delta, or go to page `to` */
  flip(book: string, delta: number, to?: number): void
  pageOf(book: string, count: number): number
  /** the `•` marker: the note changed since the window saw it */
  changed(note: Note): boolean
  seen(name: string): void
  /** the @name the Glance line filters by, or null */
  filter(): string | null
  now(): Date
  /** note.open: the note's file in the person's editor (E, "편집기로 열기") */
  edit?(name: string): void
  /**
   * the ⋯ menu's archive, delete, move, run and send (board/manage.ts);
   * a .http note's own Send names the request (its place from 1 and title)
   */
  manage?(note: Note, what: Manage, request?: Request): void
  /** tear the note off into a window of its own (the board window) */
  tear?(name: string): void
  /** put a torn-off note back on its board (the note's own window); its close does this */
  putBack?(name: string): void
  /** the ⋯ items that change where the note sits: 원래 크기로, and on the canvas 맨 앞으로 · 맨 뒤로 (03 §3.6.2) */
  layoutItems?(note: Note): MenuItem[]
  layoutAct?(note: Note, id: string): void
}

export const useBoard = (): BoardCtx => getContext<BoardCtx>(boardContext)

/** noteHex is a note colour name's colour in the palette ("yellow" is notes[0]). */
export function noteHex(name: string, p: Palette | null): string {
  const i = NOTE_NAMES.indexOf(name.toLowerCase() as (typeof NOTE_NAMES)[number])
  if (!p) return `var(--${NOTE_NAMES[Math.max(0, i)]})`
  return p.notes[i >= 0 ? i : 0]
}
