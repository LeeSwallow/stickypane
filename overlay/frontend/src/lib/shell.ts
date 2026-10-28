// The names the Go shell (overlay/internal/shell) answers to. The bindings
// that `wails3 build` generates are not checked in, so the first screens
// call the services by name; these are their names.

const pkg = 'github.com/LeeSwallow/stickypane/overlay/internal/shell'

export const home = (method: string) => `${pkg}.HomeService.${method}`
export const project = (method: string) => `${pkg}.ProjectService.${method}`

export type WindowKind = 'home' | 'project'

// windowKind reads which window this page is from its query string: the
// shell opens a project window at /?window=project.
export function windowKind(search: string): WindowKind {
  return new URLSearchParams(search).get('window') === 'project' ? 'project' : 'home'
}

// The Wails events a project window listens to, by its window id.
export const changedEvent = (id: number) => `board.changed:${id}`
export const restartedEvent = (id: number) => `bridge.restarted:${id}`
export const statusEvent = (id: number) => `project.status:${id}`
export const noticesEvent = 'home.notices'
