// What the home window asks of the shell (HomeService, overlay/internal/shell).
// The Go structs carry no json tags, so the index's fields arrive as written
// in Go; nil slices arrive as null.
import { Call, Events } from '@wailsio/runtime'
import { home, noticesEvent } from '../lib/shell'

export interface LiveSession {
  PID: number
  Kind: string
  Started: string
}

export interface Row {
  ID: number
  Path: string
  Title: string
  LastAt: string
  /** the @assignees of the board's cards; the index leaves it empty */
  Agents?: string[] | null
  Live?: LiveSession[] | null
}

export interface Day {
  Date: string
  Rows: Row[] | null
}

export interface Removed {
  ID: number
  Path: string
  Title: string
}

export interface Groups {
  Active: Row[] | null
  Days: Day[] | null
  Removed: Removed[] | null
}

export interface StickypaneState {
  found: boolean
  bridge: boolean
  path?: string
  version?: string
  error?: string
}

export interface Notice {
  code: string
  detail: string
}

/** The home window's settings in overlay.json: language "" | ko | en, mode "" | light | dark. */
export interface Prefs {
  hotkey: string
  language: string
  mode: string
}

export interface HomeApi {
  ensure(): Promise<string>
  sessions(): Promise<Groups>
  restore(id: number): Promise<void>
  pickFolder(): Promise<string>
  initProject(dir: string): Promise<void>
  openProject(path: string): Promise<unknown>
  status(): Promise<StickypaneState>
  notices(): Promise<Notice[]>
  prefs(): Promise<Prefs>
  setPrefs(p: Prefs): Promise<void>
  /** onNotices calls fn when the shell's notices change; it returns the way to stop. */
  onNotices(fn: (n: Notice[]) => void): () => void
  copy(text: string): Promise<void>
}

const call = <T>(method: string, ...args: unknown[]) => Call.ByName(home(method), ...args) as Promise<T>

export const realApi: HomeApi = {
  ensure: () => call('Ensure'),
  sessions: () => call('Sessions'),
  restore: (id) => call('Restore', id),
  pickFolder: () => call('PickFolder'),
  initProject: (dir) => call('InitProject', dir),
  openProject: (path) => call('OpenProject', path),
  status: () => call('StickypaneStatus'),
  notices: () => call('Notices'),
  prefs: () => call('Prefs'),
  setPrefs: (p) => call('SetPrefs', p),
  onNotices: (fn) => Events.On(noticesEvent, (e) => fn((e.data as Notice[]) ?? [])),
  copy: (text) => navigator.clipboard.writeText(text),
}
