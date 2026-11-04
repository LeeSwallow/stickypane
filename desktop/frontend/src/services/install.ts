// The install service: the terminal command (the app's own stickypane
// linked onto the PATH) and the agents' connection (the app's stickypane
// setup), through the shell's SetupService (desktop/internal/shell/
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

export type Agent = 'claude' | 'codex'
export type Scope = 'user' | 'project' | 'local'

/** An agent's state: its command missing, the plugin off, ok, older than the app's, or failing to load. */
export type AgentState = 'missing' | 'off' | 'ok' | 'outdated' | 'broken'

/** The board plugin in one scope. */
export interface PluginStatus {
  version?: string
  state: 'ok' | 'outdated' | 'broken'
  error?: string
}

/** One agent, as `stickypane setup --status --json` says it; plugin and mcp by scope, absent where nothing is. */
export interface AgentStatus {
  agent: Agent
  found: boolean
  path?: string
  state: AgentState
  scopes: Scope[]
  plugin: Partial<Record<Scope, PluginStatus>>
  mcp: Partial<Record<Scope, { command: string }>>
  marketplace?: { source: string; where: string; gone?: boolean } | null
}

export interface AgentsStatus {
  /** the plugin version the app's stickypane carries */
  plugin: string
  agents: AgentStatus[]
  /** the board folder whose project and local scopes these are; "" for none */
  project: string
}

/** One thing for one agent: its skills (the plugin) or its MCP server, at a scope; undo removes. */
export interface AgentRequest {
  agent: Agent
  part: 'plugin' | 'mcp'
  scope: Scope
  undo: boolean
}

export interface SetupApi {
  command(): Promise<CommandState>
  install(target: Target, replace: boolean): Promise<InstallAnswer>
  uninstall(): Promise<InstallAnswer>
  /** where each agent stands: setup --status */
  agents(): Promise<AgentsStatus>
  /** what one agent's setup would run: setup --agents <agent> … --dry-run */
  agentPlan(r: AgentRequest): Promise<RunResult>
  agentRun(r: AgentRequest): Promise<RunResult>
  /** stickypane init in this window's project, again: the agent guide */
  initHere(): Promise<RunResult>
}

const call = <T>(method: string, ...args: unknown[]) => Call.ByName(setup(method), ...args) as Promise<T>

export const setupService: SetupApi = {
  command: () => call('Command'),
  install: (target, replace) => call('InstallCommand', target, replace),
  uninstall: () => call('UninstallCommand'),
  agents: () => call('Agents'),
  agentPlan: (r) => call('AgentPlan', r),
  agentRun: (r) => call('AgentRun', r),
  initHere: () => Call.ByName(project('InitHere')) as Promise<RunResult>,
}

/**
 * Where the agents stand, summed up for the home's checklist: connected
 * (an agent found has the plugin), none (no Claude Code or Codex on the
 * PATH), or not yet.
 */
export type AgentsState = 'connected' | 'none' | 'todo'

export function agentsState(st: AgentsStatus): AgentsState {
  const found = st.agents.filter((a) => a.found)
  if (found.length === 0) return 'none'
  if (found.some((a) => a.state === 'ok' || a.state === 'outdated')) return 'connected'
  return 'todo'
}

/** checkAgents asks setup --status where the agents stand. */
export async function checkAgents(api: Pick<SetupApi, 'agents'>): Promise<AgentsState> {
  return agentsState(await api.agents())
}

/** linked is the app's link that a terminal uses, if any. */
export const linked = (s: CommandState): Link | undefined => s.links.find((l) => l.state === 'apps')
