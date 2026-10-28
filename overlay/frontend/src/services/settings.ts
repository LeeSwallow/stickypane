// The settings service. A board's settings (theme, style, language, what
// shows it) are the board's sticky.json, through the bridge's
// settings.get/set, shared with the terminal. The app's own (light or
// dark, the hotkey, the home window's language) are overlay.json, through
// the shell (RA13).

import { Call } from '@wailsio/runtime'
import { home, project } from '../lib/shell'
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
 * motion: "" (as the system says), "reduce", "off".
 */
export interface AppPrefs {
  hotkey: string
  language: string
  mode: string
  motion?: string
}

export interface SettingsApi {
  /** themes.list: every theme's palette, in the terminal's order */
  themes(): Promise<Palette[]>
  settings(): Promise<Partial<Settings>>
  setSettings(p: BoardSettingsPatch): Promise<unknown>
  appPrefs(): Promise<AppPrefs>
  setAppPrefs(p: AppPrefs): Promise<void>
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
}

/**
 * setAppPref changes some of the app's settings and keeps the others: the
 * shell takes them whole. It answers what was saved.
 */
export async function setAppPref(api: SettingsApi, patch: Partial<AppPrefs>): Promise<AppPrefs> {
  const now = await api.appPrefs()
  const next: AppPrefs = { hotkey: now?.hotkey ?? '', language: now?.language ?? '', mode: now?.mode ?? '', ...patch }
  const motion = patch.motion ?? now?.motion
  if (motion) next.motion = motion
  else delete next.motion
  await api.setAppPrefs(next)
  return next
}
