// The names the Go shell (overlay/internal/shell) answers to. The bindings
// that `wails3 build` generates are not checked in, so the first screens
// call the services by name; these are their names.

const pkg = 'github.com/LeeSwallow/stickypane/overlay/internal/shell'

export const home = (method: string) => `${pkg}.HomeService.${method}`
export const project = (method: string) => `${pkg}.ProjectService.${method}`
export const setup = (method: string) => `${pkg}.SetupService.${method}`
export const canvas = (method: string) => `${pkg}.CanvasService.${method}`

export type WindowKind = 'home' | 'project' | 'note'

// windowKind reads which window this page is from its query string: the
// shell opens a project window at /?window=project and a note torn off its
// board at /?window=note&note=<name>.
export function windowKind(search: string): WindowKind {
  const w = new URLSearchParams(search).get('window')
  return w === 'project' || w === 'note' ? w : 'home'
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
