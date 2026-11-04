// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import type { Host, HostCheck, HostsApi } from '../services/hosts'
import type { AppApi } from '../services/settings'
import { fakeSetup } from './fakeSetup'
import Hosts from './Hosts.svelte'
import SettingsWindow from './SettingsWindow.svelte'

function fakeHosts(start: Host[] = [], check: HostCheck = { ok: true, version: '0.9.0', bridge: true }) {
  let hosts = structuredClone(start)
  const api: HostsApi = {
    hosts: vi.fn(async () => structuredClone(hosts)),
    saveHost: vi.fn(async (h: Host, old: string) => {
      if (old) hosts = hosts.map((x) => (x.name === old ? h : x))
      else hosts = [...hosts, h]
    }),
    removeHost: vi.fn(async (name: string) => void (hosts = hosts.filter((h) => h.name !== name))),
    checkHost: vi.fn(async () => check),
    addBoard: vi.fn(async (name: string, path: string) => {
      hosts = hosts.map((h) => (h.name === name ? { ...h, boards: [...(h.boards ?? []), path] } : h))
    }),
    removeBoard: vi.fn(async (name: string, path: string) => {
      hosts = hosts.map((h) => (h.name === name ? { ...h, boards: (h.boards ?? []).filter((b) => b !== path) } : h))
    }),
    openRemote: vi.fn(async () => 7),
    aliases: vi.fn(async () => [{ name: 'work', resolved: 'min@10.0.0.9:22' }]),
  }
  return { api, get hosts() { return hosts } }
}

beforeEach(() => {
  setLanguage('ko')
  globalThis.ResizeObserver ??= class { observe() {} unobserve() {} disconnect() {} } as unknown as typeof ResizeObserver
  window.matchMedia = ((q: string) => ({ matches: true, media: q, addEventListener: () => {}, removeEventListener: () => {} })) as unknown as typeof window.matchMedia
})
afterEach(cleanup)

const dev: Host = { name: 'dev', target: 'min@dev.lan', boards: ['/srv/api'] }

describe('원격 호스트', () => {
  it('is a section of the settings window, saying what is kept', async () => {
    const f = fakeHosts([dev])
    const app = { prefs: vi.fn(async () => ({ hotkey: '', language: 'ko', mode: '' })), onPrefs: () => () => {}, onBoard: () => () => {}, autostart: vi.fn(async () => ({ supported: false, enabled: false })), about: vi.fn(async () => ({ app: '0.2.1', stickypane: { found: true, bridge: true } })) } as unknown as AppApi
    const board = { status: vi.fn(async () => ({ window: 0, path: '', state: 'closed', pinned: false })), load: vi.fn(), themes: vi.fn(async () => []), setSettings: vi.fn(), layout: vi.fn(), setMode: vi.fn() }
    render(SettingsWindow, { board, app, setup: fakeSetup().api, hosts: f.api, section: 'hosts' })
    expect(await screen.findByRole('heading', { name: '원격 호스트' })).toBeTruthy()
    expect(screen.getByText(/비밀번호는 저장하지 않아요/)).toBeTruthy()
    expect(await screen.findByRole('region', { name: 'dev' })).toBeTruthy()
  })

  it('with none, says so and offers to add one', async () => {
    render(Hosts, { api: fakeHosts().api })
    expect(await screen.findByText('아직 호스트가 없어요')).toBeTruthy()
    expect(screen.getByRole('button', { name: '호스트 추가' })).toBeTruthy()
  })

  it('adds a host by hand: name, ssh target, command', async () => {
    const f = fakeHosts()
    render(Hosts, { api: f.api })
    await fireEvent.click(await screen.findByRole('button', { name: '호스트 추가' }))
    await fireEvent.input(screen.getByLabelText('이름'), { target: { value: 'dev' } })
    await fireEvent.input(screen.getByLabelText('ssh 대상'), { target: { value: 'min@dev.lan' } })
    await fireEvent.input(screen.getByLabelText('원격 명령'), { target: { value: '~/go/bin/stickypane' } })
    await fireEvent.click(screen.getByRole('button', { name: '저장' }))
    await waitFor(() => expect(f.api.saveHost).toHaveBeenCalledWith({ name: 'dev', target: 'min@dev.lan', command: '~/go/bin/stickypane', boards: null }, ''))
    expect(await screen.findByRole('region', { name: 'dev' })).toBeTruthy()
  })

  it('연결 확인 says the version, a missing bridge, or what ssh said', async () => {
    const f = fakeHosts([dev])
    render(Hosts, { api: f.api })
    const host = await screen.findByRole('region', { name: 'dev' })
    await fireEvent.click(within(host).getByRole('button', { name: '연결 확인' }))
    expect(await within(host).findByText('연결됨 · stickypane 0.9.0')).toBeTruthy()
    expect(f.api.checkHost).toHaveBeenCalledWith('dev')
    cleanup()

    render(Hosts, { api: fakeHosts([dev], { ok: true, version: '0.3.0', bridge: false }).api })
    const old = await screen.findByRole('region', { name: 'dev' })
    await fireEvent.click(within(old).getByRole('button', { name: '연결 확인' }))
    expect(await within(old).findByText('bridge 없음: 원격 stickypane을 업데이트하세요')).toBeTruthy()
    cleanup()

    render(Hosts, { api: fakeHosts([dev], { ok: false, bridge: false, error: 'min@dev.lan: Permission denied (publickey).' }).api })
    const bad = await screen.findByRole('region', { name: 'dev' })
    await fireEvent.click(within(bad).getByRole('button', { name: '연결 확인' }))
    expect(await within(bad).findByText('min@dev.lan: Permission denied (publickey).')).toBeTruthy()
  })

  it('its boards: open one, take one out, add a folder', async () => {
    const f = fakeHosts([dev])
    render(Hosts, { api: f.api })
    const host = await screen.findByRole('region', { name: 'dev' })
    const boards = within(host).getByRole('list', { name: '보드' })
    expect(within(boards).getByText('/srv/api')).toBeTruthy()
    await fireEvent.click(within(boards).getByRole('button', { name: '열기' }))
    expect(f.api.openRemote).toHaveBeenCalledWith('dev', '/srv/api')

    await fireEvent.input(within(host).getByLabelText('원격 폴더'), { target: { value: '/home/min/web app' } })
    await fireEvent.click(within(host).getByRole('button', { name: '추가' }))
    await waitFor(() => expect(f.api.addBoard).toHaveBeenCalledWith('dev', '/home/min/web app'))
    expect(await within(host).findByText('/home/min/web app')).toBeTruthy()

    const row = within(host).getByText('/srv/api').closest('li') as HTMLElement
    await fireEvent.click(within(row).getByRole('button', { name: '빼기' }))
    await waitFor(() => expect(f.api.removeBoard).toHaveBeenCalledWith('dev', '/srv/api'))
  })

  it('삭제 asks first, then takes the host out', async () => {
    const f = fakeHosts([dev])
    render(Hosts, { api: f.api })
    const host = await screen.findByRole('region', { name: 'dev' })
    await fireEvent.click(within(host).getByRole('button', { name: '삭제' }))
    expect(f.api.removeHost).not.toHaveBeenCalled()
    const dialog = await screen.findByRole('alertdialog')
    await fireEvent.click(within(dialog).getByRole('button', { name: '삭제' }))
    await waitFor(() => expect(f.api.removeHost).toHaveBeenCalledWith('dev'))
    expect(await screen.findByText('아직 호스트가 없어요')).toBeTruthy()
  })

  it('imports a host from the ssh config, with where ssh would go', async () => {
    const f = fakeHosts()
    render(Hosts, { api: f.api })
    await fireEvent.click(await screen.findByRole('button', { name: 'ssh 설정에서 가져오기' }))
    const list = await screen.findByRole('list', { name: 'ssh 설정에서 가져오기' })
    expect(within(list).getByText('min@10.0.0.9:22')).toBeTruthy()
    await fireEvent.click(within(list).getByRole('button', { name: '추가' }))
    await waitFor(() => expect(f.api.saveHost).toHaveBeenCalledWith({ name: 'work', target: 'work' }, ''))
    expect(await screen.findByRole('region', { name: 'work' })).toBeTruthy()
  })

  it('a host is edited in place and keeps its boards', async () => {
    const f = fakeHosts([dev])
    render(Hosts, { api: f.api })
    const host = await screen.findByRole('region', { name: 'dev' })
    await fireEvent.click(within(host).getByRole('button', { name: '편집' }))
    const name = screen.getByLabelText('이름') as HTMLInputElement
    expect(name.value).toBe('dev')
    await fireEvent.input(screen.getByLabelText('ssh 대상'), { target: { value: 'dev-alias' } })
    await fireEvent.click(screen.getByRole('button', { name: '저장' }))
    await waitFor(() => expect(f.api.saveHost).toHaveBeenCalledWith({ name: 'dev', target: 'dev-alias', command: '', boards: ['/srv/api'] }, 'dev'))
  })
})
