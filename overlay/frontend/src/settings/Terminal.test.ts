// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it } from 'vitest'
import { setLanguage, t } from '../i18n'
import { fakeSetup } from './fakeSetup'
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

describe('the agents', () => {
  it('shows the plan first, runs it on Run, then offers init in the project', async () => {
    const f = fakeSetup()
    render(Terminal, { props: { setup: f.api, project: true } })
    await fireEvent.click(screen.getByRole('button', { name: t('settings.connectAgents') }))
    expect(f.api.connectPlan).toHaveBeenCalledWith(true)
    await waitFor(() => expect(text()).toContain('claude mcp add'))
    expect(f.api.connect).not.toHaveBeenCalled()
    await fireEvent.click(screen.getByRole('button', { name: t('agents.run') }))
    expect(f.api.connect).toHaveBeenCalledWith(true)
    await waitFor(() => expect(text()).toContain(t('agents.done')))
    await fireEvent.click(screen.getByRole('button', { name: t('agents.init') }))
    await waitFor(() => expect(text()).toContain('Wrote the guide to AGENTS.md'))
  })

  it('without MCP, and a failure is said', async () => {
    const f = fakeSetup({}, { connect: async () => Promise.reject(new Error('claude: exit status 1')) })
    render(Terminal, { props: { setup: f.api } })
    await fireEvent.click(screen.getByRole('switch', { name: t('agents.mcp') }))
    await fireEvent.click(screen.getByRole('button', { name: t('settings.connectAgents') }))
    expect(f.api.connectPlan).toHaveBeenCalledWith(false)
    await fireEvent.click(await screen.findByRole('button', { name: t('agents.run') }))
    await waitFor(() => expect(text()).toContain('claude: exit status 1'))
    expect(screen.queryByRole('button', { name: t('agents.init') })).toBeNull()
  })
})
