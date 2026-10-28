// The install service: the terminal command (the app's own stickypane
// linked onto the PATH) and the agents' connection (the app's stickypane
// setup), through the shell's SetupService (overlay/internal/shell/
// install.go). Making a board in a window's folder is the project's
// (ProjectService.MakeBoard, InitHere). Components call this, never the
// runtime (RA13).

import { Call } from '@wailsio/runtime'
import { project, setup } from '../lib/shell'

export type Target = 'user' | 'system'

/** One place the command can be linked: none, the app's, or a program that is not the app's. */
export interface Link {
  target: Target
  path: string
  state: 'none' | 'apps' | 'other'
  real?: string
}

/** A stickypane a terminal finds on its PATH. */
export interface Copy {
  path: string
  real: string
  version?: string
  apps: boolean
}

/** How to get past an older copy that comes first: brew, cask or remove. */
export interface Fix {
  kind: 'brew' | 'cask' | 'remove'
  command: string
  before?: string
  after?: string
}

export interface CommandState {
  /** the app's own stickypane; "" in a build without one */
  bundled: string
  version: string
  links: Link[]
  first?: Copy | null
  shadow?: Fix | null
  userDir: string
  onPath: boolean
  rcFile: string
  rcLine: string
}

export interface InstallAnswer {
  state: CommandState
  /** a program that is not the app's is where the link would go: ask first */
  occupied?: string
  /** the password dialog was closed */
  cancelled?: boolean
}

/** What a run printed, and the command it was. */
export interface RunResult {
  command: string
  output: string
}

export interface SetupApi {
  command(): Promise<CommandState>
  install(target: Target, replace: boolean): Promise<InstallAnswer>
  uninstall(): Promise<InstallAnswer>
  /** what connecting would run: setup --dry-run */
  connectPlan(mcp: boolean): Promise<RunResult>
  connect(mcp: boolean): Promise<RunResult>
  /** stickypane init in this window's project, again: the agent guide */
  initHere(): Promise<RunResult>
}

const call = <T>(method: string, ...args: unknown[]) => Call.ByName(setup(method), ...args) as Promise<T>

export const setupService: SetupApi = {
  command: () => call('Command'),
  install: (target, replace) => call('InstallCommand', target, replace),
  uninstall: () => call('UninstallCommand'),
  connectPlan: (mcp) => call('ConnectPlan', mcp),
  connect: (mcp) => call('Connect', mcp),
  initHere: () => Call.ByName(project('InitHere')) as Promise<RunResult>,
}

/** linked is the app's link that a terminal uses, if any. */
export const linked = (s: CommandState): Link | undefined => s.links.find((l) => l.state === 'apps')
