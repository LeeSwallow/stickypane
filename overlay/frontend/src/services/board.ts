// The board service: what a project window asks of its project's bridge,
// through the shell's ProjectService (overlay/internal/shell/services.go).
// Components call this and the other services, never the runtime (RA13);
// a test or the dev pages pass a fake of the same shape.

import { Call, Events } from '@wailsio/runtime'
import { home, project } from '../lib/shell'
import type { Action, ArrangePatch, Board } from '../board/model'

export interface Status {
  window: number
  path: string
  state: string
  error?: string
  pinned: boolean
  version?: string
}

export interface BoardApi {
  status(): Promise<Status>
  load(): Promise<Board>
  act(note: string, version: string, action: Action): Promise<unknown>
  arrange(p: { note: string } & ArrangePatch): Promise<unknown>
  order(tab: string, order: string[]): Promise<unknown>
  open(note: string): Promise<unknown>
  pin(on: boolean): Promise<unknown>
  prefs(): Promise<{ hotkey?: string; mode?: string; language?: string; motion?: string }>
}

export const boardService: BoardApi = {
  status: () => Call.ByName(project('Status')) as Promise<Status>,
  load: () => Call.ByName(project('Load')) as Promise<Board>,
  act: (note, version, action) => Call.ByName(project('Act'), { note, version, action }),
  arrange: (p) => Call.ByName(project('Arrange'), p),
  order: (tab, order) => Call.ByName(project('Order'), { tab, order }),
  open: (note) => Call.ByName(project('Open'), note),
  pin: (on) => Call.ByName(project('Pin'), on),
  prefs: () => Call.ByName(home('Prefs')) as Promise<{ hotkey?: string; mode?: string; language?: string; motion?: string }>,
}

/** The shell's events, by name; on returns the way to stop listening. */
export interface Bus {
  on(name: string, fn: (e: { data?: unknown }) => void): () => void
}

export const events: Bus = { on: (name, fn) => Events.On(name, fn) }
