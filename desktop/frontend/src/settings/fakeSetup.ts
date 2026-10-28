// Tests only: a SetupApi over a made-up machine. Nothing is run or linked.
import { vi } from 'vitest'
import type { CommandState, SetupApi } from '../services/install'

export function fakeSetup(over: Partial<CommandState> = {}, apiOver: Partial<SetupApi> = {}) {
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
  const linkTo = (target: string, s: 'none' | 'apps') => {
    state = { ...state, links: state.links.map((l) => (l.target === target ? { ...l, state: s } : l)) }
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
    connectPlan: vi.fn(async (mcp) => ({ command: 'stickypane setup --dry-run', output: 'stickypane setup runs:\n  claude plugin install board@stickypane --scope user' + (mcp ? '\n  claude mcp add' : '') })),
    connect: vi.fn(async () => ({ command: 'stickypane setup --yes', output: '  claude plugin install board@stickypane --scope user' })),
    initHere: vi.fn(async () => ({ command: 'stickypane init', output: 'Wrote the guide to AGENTS.md' })),
    ...apiOver,
  }
  return { api, get state() { return state } }
}
