// Tests only: a SetupApi over a made-up machine. Nothing is run or linked.
import { vi } from 'vitest'
import type { AgentRequest, AgentStatus, AgentsStatus, CommandState, SetupApi } from '../services/install'

/** An agent found on the PATH with nothing of stickypane's yet. */
export function agentOff(agent: 'claude' | 'codex', over: Partial<AgentStatus> = {}): AgentStatus {
  return {
    agent,
    found: true,
    path: '/Users/min/.local/bin/' + agent,
    state: 'off',
    scopes: agent === 'claude' ? ['user', 'project', 'local'] : ['user'],
    plugin: {},
    mcp: {},
    ...over,
  }
}

/** What setup --dry-run would print for one request. */
export function planOf(r: AgentRequest): string {
  if (r.part === 'mcp') {
    const scope = r.agent === 'claude' ? ` --scope ${r.scope}` : ''
    return `stickypane setup runs:\n  ${r.agent} mcp ${r.undo ? 'remove' : 'add'}${scope} stickypane`
  }
  const verb = r.undo ? (r.agent === 'claude' ? 'uninstall' : 'remove') : r.agent === 'claude' ? 'install' : 'add'
  return `stickypane setup runs:\n  ${r.agent} plugin ${verb} board@stickypane` + (r.agent === 'claude' ? ` --scope ${r.scope}` : '')
}

export function fakeSetup(over: Partial<CommandState> = {}, apiOver: Partial<SetupApi> = {}, agents?: Partial<AgentsStatus>) {
  let state: CommandState = {
    bundled: '/Applications/Stickypane.app/Contents/Helpers/stickypane',
    version: '0.2.0',
    links: [
      { target: 'user', path: '/Users/min/.local/bin/stickypane', state: 'none' },
      { target: 'system', path: '/usr/local/bin/stickypane', state: 'none' },
    ],
    first: null,
    shadow: null,
    userDir: '/Users/min/.local/bin',
    onPath: true,
    rcFile: '/Users/min/.zshrc',
    rcLine: 'export PATH="$HOME/.local/bin:$PATH"',
    ...over,
  }
  let status: AgentsStatus = { plugin: '0.2.1', agents: [agentOff('claude'), agentOff('codex')], project: '', ...agents }
  const linkTo = (target: string, s: 'none' | 'apps') => {
    state = { ...state, links: state.links.map((l) => (l.target === target ? { ...l, state: s } : l)) }
  }
  // a run does what it says, so the status read after it shows it
  const apply = (r: AgentRequest) => {
    status = {
      ...status,
      agents: status.agents.map((a) => {
        if (a.agent !== r.agent) return a
        const b = { ...a, plugin: { ...a.plugin }, mcp: { ...a.mcp } }
        if (r.part === 'plugin') {
          if (r.undo) delete b.plugin[r.scope]
          else b.plugin[r.scope] = { version: status.plugin, state: 'ok' }
        } else if (r.undo) delete b.mcp[r.scope]
        else b.mcp[r.scope] = { command: '/Applications/Stickypane.app/Contents/Helpers/stickypane' }
        b.state = Object.keys(b.plugin).length ? 'ok' : 'off'
        b.marketplace = r.part === 'plugin' && !r.undo ? { source: 'github', where: 'LeeSwallow/stickypane' } : b.marketplace
        return b
      }),
    }
  }
  const api: SetupApi = {
    command: vi.fn(async () => state),
    install: vi.fn(async (target) => {
      linkTo(target, 'apps')
      return { state }
    }),
    uninstall: vi.fn(async () => {
      linkTo('user', 'none')
      linkTo('system', 'none')
      return { state }
    }),
    agents: vi.fn(async () => status),
    agentPlan: vi.fn(async (r: AgentRequest) => ({ command: `stickypane setup --agents ${r.agent} --dry-run`, output: planOf(r) })),
    agentRun: vi.fn(async (r: AgentRequest) => {
      apply(r)
      return { command: `stickypane setup --agents ${r.agent} --yes`, output: planOf(r).replace('stickypane setup runs:\n', '') }
    }),
    initHere: vi.fn(async () => ({ command: 'stickypane init', output: 'Wrote the guide to AGENTS.md' })),
    ...apiOver,
  }
  return {
    api,
    get state() {
      return state
    },
    get status() {
      return status
    },
  }
}
