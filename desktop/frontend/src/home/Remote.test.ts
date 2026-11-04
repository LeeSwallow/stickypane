// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import type { Groups, HomeApi } from '../services/home'
import type { Host } from '../services/hosts'
import { fakeSetup } from '../settings/fakeSetup'
import Home from './Home.svelte'

const empty: Groups = { Active: null, Days: null, Removed: null }
const local: Groups = { Active: null, Days: [{ Date: '2026-10-28', Rows: [{ ID: 1, Path: '/home/min/here', Title: 'here', LastAt: new Date(2026, 9, 28, 9).toISOString() }] }], Removed: null }

function fakeApi(groups: Groups, hosts: Host[]): HomeApi {
  return {
    ensure: vi.fn(async () => '/home/min/.stickypane'),
    sessions: vi.fn(async () => groups),
    restore: vi.fn(async () => {}),
    pickFolder: vi.fn(async () => ''),
    inspect: vi.fn(async (dir: string) => ({ exists: false, path: dir })),
    initProject: vi.fn(async () => {}),
    openProject: vi.fn(async () => 1),
    status: vi.fn(async () => ({ found: true, bridge: true, version: '0.2.0' })),
    notices: vi.fn(async () => []),
    prefs: vi.fn(async () => ({ hotkey: 'Alt+Space', language: 'ko', mode: '' })),
    setPrefs: vi.fn(async () => {}),
    onNotices: vi.fn(() => () => {}),
    copy: vi.fn(async () => {}),
    peeks: vi.fn(async () => ({ 'ssh://dev/srv/api': { notes: 4 } })),
    openSettings: vi.fn(async () => {}),
    hosts: vi.fn(async () => hosts),
    openRemote: vi.fn(async () => 2),
  }
}

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  window.matchMedia = ((q: string) => ({ matches: false, media: q, addEventListener: () => {}, removeEventListener: () => {} })) as unknown as typeof window.matchMedia
  setLanguage('ko')
})
afterEach(cleanup)

const mount = (api: HomeApi) => render(Home, { api, setup: fakeSetup().api, now: () => new Date(2026, 9, 28, 15), onPrefs: () => () => {} })
const hosts: Host[] = [
  { name: 'dev', target: 'min@dev', boards: ['/srv/api', '/home/min/web app'] },
  { name: 'box', target: 'box', boards: null },
]

describe('the home\'s 원격 group', () => {
  it('lists every host\'s boards as host · path, and opens one over ssh', async () => {
    const api = fakeApi(local, hosts)
    mount(api)
    const group = await screen.findByRole('region', { name: '원격' })
    const cards = within(group).getAllByRole('button')
    expect(cards).toHaveLength(2)
    expect(within(group).getByText('/srv/api')).toBeTruthy()
    expect(within(group).getByText('web app')).toBeTruthy()
    // what it looked like when this computer last opened it
    expect(within(group).getByText('노트 4장')).toBeTruthy()
    await fireEvent.click(cards[0])
    expect(api.openRemote).toHaveBeenCalledWith('dev', '/srv/api')
    expect(api.openProject).not.toHaveBeenCalled()
    // this computer's boards are listed as before
    expect(screen.getByText('here', { selector: '.title' })).toBeTruthy()
  })

  it('remote boards alone are a list, not the welcome', async () => {
    mount(fakeApi(empty, hosts))
    expect(await screen.findByRole('region', { name: '원격' })).toBeTruthy()
    expect(screen.queryByText('첫 보드까지 세 걸음')).toBeNull()
  })

  it('no group when no host has a board', async () => {
    mount(fakeApi(local, [{ name: 'box', target: 'box' }]))
    await screen.findByText('here', { selector: '.title' })
    expect(screen.queryByRole('region', { name: '원격' })).toBeNull()
  })

  it('the search narrows them by host and folder', async () => {
    mount(fakeApi(local, hosts))
    await screen.findByRole('region', { name: '원격' })
    const search = screen.getByRole('searchbox')
    await fireEvent.input(search, { target: { value: 'web' } })
    const group = screen.getByRole('region', { name: '원격' })
    expect(within(group).getAllByRole('button')).toHaveLength(1)
  })
})
