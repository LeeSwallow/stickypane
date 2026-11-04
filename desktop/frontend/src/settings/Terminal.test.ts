// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage, t } from '../i18n'
import { agentOff, fakeSetup } from './fakeSetup'
import Terminal from './Terminal.svelte'

beforeEach(() => {
  setLanguage('en')
  // the dialogs of the design system measure themselves
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver
})
afterEach(cleanup)

const text = () => document.body.textContent ?? ''

describe('the terminal command', () => {
  it('installs into ~/.local/bin without a password, then offers to remove it', async () => {
    const f = fakeSetup()
    render(Terminal, { props: { setup: f.api } })
    await fireEvent.click(await screen.findByRole('button', { name: t('cli.install') }))
    expect(f.api.install).toHaveBeenCalledWith('user', false)
    await screen.findByRole('button', { name: t('cli.uninstall') })
    expect(text()).toContain('/Users/min/.local/bin/stickypane')
    await fireEvent.click(screen.getByRole('button', { name: t('cli.uninstall') }))
    await screen.findByRole('button', { name: t('cli.install') })
  })

  it('/usr/local/bin only when picked', async () => {
    const f = fakeSetup()
    render(Terminal, { props: { setup: f.api } })
    await fireEvent.click(await screen.findByRole('button', { name: t('cli.installSystem') }))
    expect(f.api.install).toHaveBeenCalledWith('system', false)
  })

  it('says the line to add when ~/.local/bin is not on the PATH', async () => {
    const f = fakeSetup({ onPath: false, links: [{ target: 'user', path: '/Users/min/.local/bin/stickypane', state: 'apps' }] })
    render(Terminal, { props: { setup: f.api } })
    await waitFor(() => expect(text()).toContain('/Users/min/.zshrc'))
    expect(text()).toContain('export PATH="$HOME/.local/bin:$PATH"')
  })

  it('names an older copy that comes first, with its version and the fix', async () => {
    const f = fakeSetup({
      first: { path: '/opt/homebrew/bin/stickypane', real: '/opt/homebrew/Caskroom/stickypane/0.1.1/stickypane', version: '0.1.1', apps: false },
      shadow: { kind: 'cask', command: 'brew uninstall --cask stickypane', before: '/Users/min/.local/bin', after: '/opt/homebrew/bin' },
    })
    render(Terminal, { props: { setup: f.api } })
    await waitFor(() => expect(text()).toContain('/opt/homebrew/bin/stickypane'))
    expect(text()).toContain('0.1.1')
    expect(text()).toContain('brew uninstall --cask stickypane')
    expect(text()).toContain(t('cli.fix.order', { before: '/Users/min/.local/bin', after: '/opt/homebrew/bin' }))
  })

  it('asks before it replaces a program the app did not put there', async () => {
    const f = fakeSetup({}, {})
    let calls = 0
    f.api.install = async (target, replace) => {
      calls++
      if (!replace) return { state: f.state, occupied: '/Users/min/.local/bin/stickypane' }
      return { state: { ...f.state, links: [{ target, path: '/Users/min/.local/bin/stickypane', state: 'apps' }] } }
    }
    render(Terminal, { props: { setup: f.api } })
    await fireEvent.click(await screen.findByRole('button', { name: t('cli.install') }))
    await fireEvent.click(await screen.findByRole('button', { name: t('cli.replace') }))
    await screen.findByRole('button', { name: t('cli.uninstall') })
    expect(calls).toBe(2)
  })

  it('a closed password dialog changes nothing, and says so', async () => {
    const f = fakeSetup({}, { install: async () => ({ state: fakeSetup().state, cancelled: true }) })
    render(Terminal, { props: { setup: f.api } })
    await fireEvent.click(await screen.findByRole('button', { name: t('cli.installSystem') }))
    await waitFor(() => expect(text()).toContain(t('cli.cancelled')))
  })

  it('a build without its own stickypane has nothing to install', async () => {
    render(Terminal, { props: { setup: fakeSetup({ bundled: '' }).api } })
    await waitFor(() => expect(text()).toContain(t('cli.none')))
    expect(screen.queryByRole('button', { name: t('cli.install') })).toBeNull()
  })
})

// one agent's card, by its name
const card = async (name: string) => within(await screen.findByRole('region', { name }))

describe('the agents, each on its own', () => {
  it('a row each, with its own status', async () => {
    const f = fakeSetup({}, {}, {
      agents: [
        agentOff('claude', { state: 'ok', plugin: { user: { version: '0.2.1', state: 'ok' } }, marketplace: { source: 'github', where: 'LeeSwallow/stickypane' } }),
        agentOff('codex', { found: false, state: 'missing', path: undefined }),
      ],
    })
    render(Terminal, { props: { setup: f.api } })
    const claude = await card('Claude Code')
    expect(claude.getByText(t('agent.ok', { version: 'v0.2.1' }))).toBeTruthy()
    expect(claude.getByRole('button', { name: t('agent.remove') })).toBeTruthy()
    const codex = await card('Codex')
    expect(codex.getByText(t('agent.missing'))).toBeTruthy()
    expect(codex.getByText(t('agent.missingHint', { cmd: 'codex' }))).toBeTruthy()
    expect(codex.queryByRole('button')).toBeNull()
    expect(screen.queryByRole('button', { name: /connect…|연결…/i })).toBeNull()
  })

  it('an older plugin offers its update, a broken one its repair', async () => {
    const f = fakeSetup({}, {}, {
      plugin: '0.3.0',
      agents: [
        agentOff('claude', { state: 'broken', plugin: { user: { version: '0.2.1', state: 'broken', error: 'Marketplace stickypane failed to load: cache-miss' } }, marketplace: { source: 'directory', where: '/Users/min/Projects/stickypane', gone: true } }),
        agentOff('codex', { state: 'outdated', plugin: { user: { version: '0.2.1', state: 'outdated' } } }),
      ],
    })
    render(Terminal, { props: { setup: f.api } })
    const claude = await card('Claude Code')
    expect(claude.getByText(t('agent.broken'))).toBeTruthy()
    expect(text()).toContain(t('agent.gone', { where: '/Users/min/Projects/stickypane' }))
    await fireEvent.click(claude.getByRole('button', { name: t('agent.repair') }))
    expect(f.api.agentPlan).toHaveBeenCalledWith({ agent: 'claude', part: 'plugin', scope: 'user', undo: false })
    const codex = await card('Codex')
    expect(codex.getByText(t('agent.outdated'))).toBeTruthy()
    expect(codex.getByText(t('agent.newer', { version: 'v0.3.0' }))).toBeTruthy()
    expect(codex.getByRole('button', { name: t('agent.update') })).toBeTruthy()
  })

  it('connects one agent: the plan first, then Run, then init in the project', async () => {
    const f = fakeSetup()
    render(Terminal, { props: { setup: f.api, project: true } })
    const claude = await card('Claude Code')
    await fireEvent.click(claude.getByRole('button', { name: t('agent.connect') }))
    await waitFor(() => expect(text()).toContain('claude plugin install board@stickypane --scope user'))
    expect(f.api.agentRun).not.toHaveBeenCalled()
    await fireEvent.click(claude.getByRole('button', { name: t('agents.run') }))
    expect(f.api.agentRun).toHaveBeenCalledWith({ agent: 'claude', part: 'plugin', scope: 'user', undo: false })
    await waitFor(() => expect(claude.getByText(t('agent.ok', { version: 'v0.2.1' }))).toBeTruthy())
    // Codex was left alone
    expect((await card('Codex')).getByText(t('agents.notConnected'))).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: t('agents.init') }))
    await waitFor(() => expect(text()).toContain('Wrote the guide to AGENTS.md'))
  })

  it('the MCP switch registers and unregisters one agent, plan first', async () => {
    const f = fakeSetup()
    render(Terminal, { props: { setup: f.api } })
    const codex = await card('Codex')
    await fireEvent.click(codex.getByRole('switch', { name: t('agent.mcpOn') }))
    expect(f.api.agentPlan).toHaveBeenCalledWith({ agent: 'codex', part: 'mcp', scope: 'user', undo: false })
    await waitFor(() => expect(text()).toContain('codex mcp add stickypane'))
    // cancelled, the switch goes back
    await fireEvent.click(codex.getByRole('button', { name: t('ui.cancel') }))
    expect(codex.getByRole('switch', { name: t('agent.mcpOn') }).getAttribute('aria-checked')).toBe('false')
    await fireEvent.click(codex.getByRole('switch', { name: t('agent.mcpOn') }))
    await fireEvent.click(await codex.findByRole('button', { name: t('agents.run') }))
    await waitFor(() => expect(codex.getByText(t('agent.mcpAt', { command: '/Applications/Stickypane.app/Contents/Helpers/stickypane' }))).toBeTruthy())
    await fireEvent.click(codex.getByRole('switch', { name: t('agent.mcpOn') }))
    expect(f.api.agentPlan).toHaveBeenLastCalledWith({ agent: 'codex', part: 'mcp', scope: 'user', undo: true })
  })

  it('in a project, Claude Code offers this project and local; Codex the user only', async () => {
    const f = fakeSetup({}, {}, { project: '/Users/min/work/app' })
    render(Terminal, { props: { setup: f.api, project: true } })
    const claude = await card('Claude Code')
    await fireEvent.click(claude.getByRole('radio', { name: t('scope.project') }))
    expect(text()).toContain(t('scope.projectHint'))
    await fireEvent.click(claude.getByRole('button', { name: t('agent.connect') }))
    expect(f.api.agentPlan).toHaveBeenCalledWith({ agent: 'claude', part: 'plugin', scope: 'project', undo: false })
    const codex = await card('Codex')
    expect(codex.queryByRole('radio')).toBeNull()
    expect(codex.getByText(t('agent.codexUserOnly'))).toBeTruthy()
  })

  it('without a project, the user scope only', async () => {
    render(Terminal, { props: { setup: fakeSetup().api } })
    const claude = await card('Claude Code')
    expect(claude.queryByRole('radio')).toBeNull()
  })

  it('never asks for a setup of more than one agent', async () => {
    const f = fakeSetup({}, {}, { project: '/Users/min/work/app' })
    render(Terminal, { props: { setup: f.api, project: true } })
    for (const name of ['Claude Code', 'Codex']) {
      const c = await card(name)
      await fireEvent.click(c.getByRole('button', { name: t('agent.connect') }))
      await fireEvent.click(await c.findByRole('button', { name: t('agents.run') }))
      await waitFor(() => expect(c.getByRole('button', { name: t('agent.remove') })).toBeTruthy())
      await fireEvent.click(c.getByRole('switch', { name: t('agent.mcpOn') }))
      await fireEvent.click(await c.findByRole('button', { name: t('agents.run') }))
    }
    const calls = [...vi.mocked(f.api.agentPlan).mock.calls, ...vi.mocked(f.api.agentRun).mock.calls].map((c) => c[0])
    expect(calls.length).toBe(8)
    for (const r of calls) expect(['claude', 'codex']).toContain(r.agent)
    expect(new Set(vi.mocked(f.api.agentRun).mock.calls.map((c) => c[0].agent))).toEqual(new Set(['claude', 'codex']))
  })

  it('a failure is said in that agent', async () => {
    const f = fakeSetup({}, { agentRun: async () => Promise.reject(new Error('claude: exit status 1')) })
    render(Terminal, { props: { setup: f.api, project: true } })
    const claude = await card('Claude Code')
    await fireEvent.click(claude.getByRole('button', { name: t('agent.connect') }))
    await fireEvent.click(await claude.findByRole('button', { name: t('agents.run') }))
    await waitFor(() => expect(claude.getByRole('alert').textContent).toContain('claude: exit status 1'))
    expect(screen.queryByRole('button', { name: t('agents.init') })).toBeNull()
  })
})
