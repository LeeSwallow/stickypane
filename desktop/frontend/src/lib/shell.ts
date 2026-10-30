// The names the Go shell (desktop/internal/shell) answers to. The bindings
// that `wails3 build` generates are not checked in, so the first screens
// call the services by name; these are their names.

const pkg = 'github.com/LeeSwallow/stickypane/desktop/internal/shell'

export const home = (method: string) => `${pkg}.HomeService.${method}`
export const project = (method: string) => `${pkg}.ProjectService.${method}`
export const setup = (method: string) => `${pkg}.SetupService.${method}`
export const canvas = (method: string) => `${pkg}.CanvasService.${method}`

export type WindowKind = 'home' | 'project' | 'note' | 'settings'

// windowKind reads which window this page is from its query string: the
// shell opens a project window at /?window=project, a note torn off its
// board at /?window=note&note=<name>, and the settings at
// /?window=settings&section=<section>.
export function windowKind(search: string): WindowKind {
  const w = new URLSearchParams(search).get('window')
  return w === 'project' || w === 'note' || w === 'settings' ? w : 'home'
}

/** noteOf is the note a torn-off note's window was opened on. */
export const noteOf = (search: string): string => new URLSearchParams(search).get('note') ?? ''

// The Wails events a project window listens to, by its window id.
export const changedEvent = (id: number) => `board.changed:${id}`
export const restartedEvent = (id: number) => `bridge.restarted:${id}`
export const statusEvent = (id: number) => `project.status:${id}`
export const noticesEvent = 'home.notices'
// A project window's layout changed outside it: a note torn off, put back or gone.
export const canvasEvent = (id: number) => `canvas.changed:${id}`
// The app's settings changed (every window hears it); the settings
// window's board changed.
export const prefsEvent = 'prefs.changed'
export const settingsBoardEvent = 'settings.board'
