// The settings service. A board's settings (theme, style, language, what
// shows it) are the board's sticky.json, through the bridge's
// settings.get/set, shared with the terminal. The app's own (light or
// dark, the hotkey, the home window's language, motion, new windows on
// top) are overlay.json, through the shell (RA13), and every window hears
// when they change. The settings window (03 §3.12) is opened through here.

import { Call, Events } from '@wailsio/runtime'
import { home, prefsEvent, project, settingsBoardEvent } from '../lib/shell'
import type { Settings } from '../board/model'
import { normalizePalette, type Palette } from '../lib/theme/palette'

/** What settings.set changes; a field left out stays. */
export interface BoardSettingsPatch {
  theme?: string
  style?: 'doodle' | 'neat'
  view?: 'terminal' | 'overlay' | 'both'
  language?: 'auto' | 'ko' | 'en'
}

/**
 * The app's settings in overlay.json. mode: "" (system), "light", "dark";
 * motion: "" (as the system says), "reduce", "off"; pinNew: a board's
 * window seen for the first time stays on top.
 */
export interface AppPrefs {
  hotkey: string
  language: string
  mode: string
  motion?: string
  pinNew?: boolean
}

export interface SettingsApi {
  /** themes.list: every theme's palette, in the terminal's order */
  themes(): Promise<Palette[]>
  settings(): Promise<Partial<Settings>>
  setSettings(p: BoardSettingsPatch): Promise<unknown>
  appPrefs(): Promise<AppPrefs>
  setAppPrefs(p: AppPrefs): Promise<void>
  /** opens the settings window on a section, for this window's board */
  openSettings(section?: Section): Promise<unknown>
}

/** readThemes reads themes.list's answer; a palette it cannot read is left out. */
export function readThemes(raw: unknown): Palette[] {
  const list = (raw as { themes?: unknown[] } | null)?.themes
  if (!Array.isArray(list)) return []
  const out: Palette[] = []
  for (const t of list) {
    try {
      out.push(normalizePalette(t))
    } catch {
      // a theme the app cannot draw is not offered
    }
  }
  return out
}

export const settingsService: SettingsApi = {
  themes: async () => readThemes(await Call.ByName(project('Themes'))),
  settings: () => Call.ByName(project('Settings')) as Promise<Partial<Settings>>,
  setSettings: (p) => Call.ByName(project('SetSetting'), p),
  appPrefs: () => Call.ByName(home('Prefs')) as Promise<AppPrefs>,
  setAppPrefs: (p) => Call.ByName(home('SetPrefs'), p) as Promise<void>,
  openSettings: (section = 'board') => Call.ByName(project('OpenSettings'), section),
}

/**
 * setAppPref changes some of the app's settings and keeps the others: the
 * shell takes them whole. It answers what was saved.
 */
export async function setAppPref(api: Pick<SettingsApi, 'appPrefs' | 'setAppPrefs'>, patch: Partial<AppPrefs>): Promise<AppPrefs> {
  const now = await api.appPrefs()
  const next: AppPrefs = {
    hotkey: now?.hotkey ?? '',
    language: now?.language ?? '',
    mode: now?.mode ?? '',
    pinNew: now?.pinNew ?? false,
    ...patch,
  }
  const motion = patch.motion ?? now?.motion
  if (motion) next.motion = motion
  else delete next.motion
  await api.setAppPrefs(next)
  return next
}

/** The sections of the settings window. */
export type Section = 'board' | 'app' | 'cli' | 'about'
export const SECTIONS: Section[] = ['board', 'app', 'cli', 'about']
export const sectionOf = (s: string | null | undefined): Section => (SECTIONS.includes(s as Section) ? (s as Section) : 'board')

/** Whether a hotkey can be had (the shell's HotkeyCheck). */
export interface HotkeyCheck {
  ok: boolean
  accel: string
  /** empty | key | modifier | reserved | taken */
  code?: string
  detail?: string
}

/** Opening the app at login: whether it can here, and whether it does. */
export interface Autostart {
  supported: boolean
  enabled: boolean
  error?: string
}

/** What 정보 shows. */
export interface About {
  app: string
  stickypane: { found: boolean; bridge: boolean; path?: string; version?: string }
}

/** The app's side of the settings: overlay.json, the hotkey, login, the window. */
export interface AppApi {
  prefs(): Promise<AppPrefs>
  setPrefs(p: AppPrefs): Promise<void>
  /** onPrefs calls fn when the app's settings change, from any window; it returns the way to stop */
  onPrefs(fn: (p: AppPrefs) => void): () => void
  checkHotkey(accel: string): Promise<HotkeyCheck>
  autostart(): Promise<Autostart>
  setAutostart(on: boolean): Promise<Autostart>
  about(): Promise<About>
  /** onBoard calls fn when the settings window's board changes (opened from another board, or closed) */
  onBoard(fn: (section?: Section) => void): () => void
  /** opens the links of 정보 in the browser */
  openURL(url: string): Promise<void>
}

const callHome = <T>(method: string, ...args: unknown[]) => Call.ByName(home(method), ...args) as Promise<T>

export const appService: AppApi = {
  prefs: () => callHome('Prefs'),
  setPrefs: (p) => callHome('SetPrefs', p),
  onPrefs: (fn) => Events.On(prefsEvent, (e) => e.data && fn(e.data as AppPrefs)),
  checkHotkey: (accel) => callHome('CheckHotkey', accel),
  autostart: () => callHome('Autostart'),
  setAutostart: (on) => callHome('SetAutostart', on),
  about: () => callHome('About'),
  onBoard: (fn) => Events.On(settingsBoardEvent, (e) => fn((e.data as { section?: Section } | undefined)?.section)),
  openURL: async (url) => {
    const { Browser } = await import('@wailsio/runtime')
    await Browser.OpenURL(url)
  },
}

/**
 * openSettings opens the settings window on a section: from a board's
 * window, for that board; from the home, with none.
 */
export const openSettings = (from: 'home' | 'project', section: Section = 'board'): Promise<unknown> =>
  from === 'project' ? Call.ByName(project('OpenSettings'), section) : Call.ByName(home('OpenSettings'), section)
