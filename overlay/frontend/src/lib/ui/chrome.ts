// Window chrome (03 §3.11.6): how a window gets its close / minimise / zoom.
// The shell opens every window with ?chrome=: "mac" keeps the native
// traffic lights over our title bar (hidden-inset), so the bar leaves room
// for them; "own" (Linux) has no native buttons, so the bar draws close and
// minimise itself. In a browser (the dev pages) it is "none".
import { home, project, type WindowKind } from '../shell'
import { SPACE } from '../theme/scale'
import type { MenuItem } from './menu'

export type Chrome = 'mac' | 'own' | 'none'

/** chromeOf reads the chrome from a page's query string. */
export function chromeOf(search: string): Chrome {
  const c = new URLSearchParams(search).get('chrome')
  return c === 'mac' || c === 'own' ? c : 'none'
}

/**
 * TRAFFIC_LIGHTS is the room the macOS buttons take at the bar's start, in
 * px. With the hidden-inset title bar and its (empty) unified toolbar the
 * lights sit further right, and larger on recent macOS, than in a plain
 * title bar: 92px holds all three.
 */
export const TRAFFIC_LIGHTS = 92

/** barLead is the title bar's left padding: the lights and space-3 on macOS, space-4 elsewhere. */
export const barLead = (c: Chrome): number => (c === 'mac' ? TRAFFIC_LIGHTS + SPACE[2] : SPACE[3])

/** The window actions the title bar wires its buttons and its menu to. */
export interface WindowActions {
  close: () => void
  minimise: () => void
  /** ends the app (not just this window): the shell's Quit */
  quit: () => void
}

const runtime = () => import('@wailsio/runtime')

/**
 * windowActions are the real ones, for a window of a kind. Closing a
 * project window closes it (the app stays in the tray); closing the home
 * hides it (the shell does that). Quit ends the app.
 */
export function windowActions(kind: WindowKind): WindowActions {
  const svc = kind === 'project' ? project : home
  return {
    close: () => void runtime().then((r) => r.Window.Close()),
    minimise: () => void runtime().then((r) => r.Window.Minimise()),
    quit: () => void runtime().then((r) => r.Call.ByName(svc('Quit'))),
  }
}

/** quitShortcut is what quits from the keyboard: Cmd+Q from the macOS app menu; none elsewhere. */
export const quitShortcut = (c: Chrome): string | undefined => (c === 'mac' ? '⌘Q' : undefined)

/**
 * windowMenu is the end of every title bar's ⋯ menu: close this window,
 * and quit the app (it would otherwise live on in the tray).
 */
export function windowMenu(chrome: Chrome, t: (k: string) => string): MenuItem[] {
  return [
    { kind: 'separator', id: 'sep-window' },
    { id: 'window.close', label: t('ui.closeWindow'), icon: 'close' },
    { id: 'app.quit', label: t('app.quit'), shortcut: quitShortcut(chrome) },
  ]
}

/** runWindowMenu does a window item of the ⋯ menu; false when the id is not one. */
export function runWindowMenu(id: string, a: WindowActions): boolean {
  if (id === 'window.close') a.close()
  else if (id === 'app.quit') a.quit()
  else return false
  return true
}
