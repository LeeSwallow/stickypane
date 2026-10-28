// The window service: close, minimise and quit, for the title bar. The
// runtime is reached from services only (RA13).
import { home, project, type WindowKind } from '../lib/shell'
import type { WindowActions } from '../lib/ui/chrome'

const runtime = () => import('@wailsio/runtime')

/**
 * windowActions are the real ones, for a window of a kind. Closing a
 * project window closes it (the app stays in the tray); closing the home
 * hides it (the shell does that). Quit ends the app.
 */
export function windowActions(kind: WindowKind): WindowActions {
  const svc = kind === 'home' ? home : project
  return {
    close: () => void runtime().then((r) => r.Window.Close()),
    minimise: () => void runtime().then((r) => r.Window.Minimise()),
    quit: () => void runtime().then((r) => r.Call.ByName(svc('Quit'))),
  }
}
