// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import { THEMES } from '../lib/theme/palettes'
import { STICKYPANE_DARK, STICKYPANE_LIGHT } from '../lib/theme/fallback'
import type { Groups, HomeApi, Row } from './api'
import Home from './Home.svelte'

const NOW = new Date(2026, 9, 28, 15, 0, 0)
const iso = (d: Date) => d.toISOString()
const at = (days: number, h = 10, m = 0) => new Date(2026, 9, 28 - days, h, m, 0)

const row = (id: number, title: string, last: Date, extra: Partial<Row> = {}): Row => ({
  ID: id,
  Path: `/home/min/${title}`,
  Title: title,
  LastAt: iso(last),
  Agents: null,
  Live: null,
  ...extra,
})

function fakeApi(groups: Groups, over: Partial<HomeApi> = {}): HomeApi {
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
    ...over,
  }
}

const empty: Groups = { Active: null, Days: null, Removed: null }

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
  window.matchMedia = ((q: string) => ({
    matches: false,
    media: q,
    addEventListener: () => {},
    removeEventListener: () => {},
  })) as unknown as typeof window.matchMedia
  setLanguage('ko')
})
afterEach(cleanup)

const mount = (api: HomeApi) => render(Home, { api, now: () => NOW })

describe('home', () => {
  it('home opens straight to the list', async () => {
    const api = fakeApi({
      Active: [row(1, 'stickypane', at(0, 14, 58), { Live: [{ PID: 1, Kind: 'tui', Started: iso(at(0, 14)) }] })],
      Days: null,
      Removed: null,
    })
    mount(api)
    expect(await screen.findByText('stickypane', { selector: '.title' })).toBeTruthy()
    expect(api.ensure).toHaveBeenCalled()
    expect(api.sessions).toHaveBeenCalled()
    expect(screen.getByText('/home/min/stickypane')).toBeTruthy()
    expect(screen.getByText('지금 움직이는 중')).toBeTruthy()
    // no landing, no empty state, no explanation
    expect(screen.queryByText('아직 붙은 보드가 없어요')).toBeNull()
  })

  it('clicking a row opens that project', async () => {
    const api = fakeApi({ Active: [row(1, 'stickypane', at(0, 14, 58))], Days: null, Removed: null })
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: 'stickypane 열기' }))
    expect(api.openProject).toHaveBeenCalledWith('/home/min/stickypane')
  })

  it('empty state offers one button', async () => {
    const api = fakeApi(empty, { pickFolder: vi.fn(async () => '/home/min/new') })
    mount(api)
    expect(await screen.findByText('아직 붙은 보드가 없어요')).toBeTruthy()
    const buttons = screen.getAllByRole('button').filter((b) => !b.closest('.language, [data-titlebar]'))
    expect(buttons.map((b) => b.textContent?.trim())).toEqual(['폴더 고르기'])
    await fireEvent.click(buttons[0])
    // no board is made until the person says so
    const ask = await screen.findByRole('alertdialog')
    expect(within(ask).getByText('이 폴더에 보드를 만들까요?')).toBeTruthy()
    expect(within(ask).getByText('/home/min/new')).toBeTruthy()
    expect(api.initProject).not.toHaveBeenCalled()
    await fireEvent.click(within(ask).getByRole('button', { name: '만들기' }))
    await waitFor(() => expect(api.initProject).toHaveBeenCalledWith('/home/min/new'))
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })

  it('cancelling the confirm makes nothing', async () => {
    const api = fakeApi(empty, { pickFolder: vi.fn(async () => '/home/min/new') })
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: '폴더 고르기' }))
    const ask = await screen.findByRole('alertdialog')
    await fireEvent.click(within(ask).getByRole('button', { name: '취소' }))
    expect(screen.queryByRole('alertdialog')).toBeNull()
    expect(api.initProject).not.toHaveBeenCalled()
    expect(api.openProject).not.toHaveBeenCalled()
  })

  it('a folder that already has a board opens without init', async () => {
    const api = fakeApi(empty, {
      pickFolder: vi.fn(async () => '/home/min/has'),
      inspect: vi.fn(async (dir: string) => ({ exists: true, path: dir })),
    })
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: '폴더 고르기' }))
    await waitFor(() => expect(api.openProject).toHaveBeenCalledWith('/home/min/has'))
    expect(api.initProject).not.toHaveBeenCalled()
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })

  it('a failed init shows a notice', async () => {
    const api = fakeApi(empty, {
      pickFolder: vi.fn(async () => '/home/min/new'),
      initProject: vi.fn(async () => {
        throw new Error('refused')
      }),
    })
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: '폴더 고르기' }))
    await fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '만들기' }))
    expect(await screen.findByText('보드를 만들지 못했어요 · refused')).toBeTruthy()
  })

  it('a failed undo shows a notice', async () => {
    const api = fakeApi(
      { Active: null, Days: null, Removed: [{ ID: 7, Path: '/gone/a', Title: 'a' }] },
      {
        restore: vi.fn(async () => {
          throw new Error('locked')
        }),
      },
    )
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: '되돌리기' }))
    expect(await screen.findByText('되돌리지 못했어요 · locked')).toBeTruthy()
  })

  it('a failed status check shows a notice and keeps the folder button off', async () => {
    const api = fakeApi(empty, {
      status: vi.fn(async () => {
        throw new Error('no shell')
      }),
    })
    mount(api)
    expect(await screen.findByText('stickypane을 확인하지 못했어요 · no shell')).toBeTruthy()
    expect((screen.getByRole('button', { name: '폴더 고르기' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('a failed notices read shows a notice', async () => {
    const api = fakeApi(empty, {
      notices: vi.fn(async () => {
        throw new Error('gone')
      }),
    })
    mount(api)
    expect(await screen.findByText('알림을 읽지 못했어요 · gone')).toBeTruthy()
  })

  it('the x hides the removed note and it stays hidden', async () => {
    const groups: Groups = { Active: null, Days: null, Removed: [{ ID: 7, Path: '/gone/a', Title: 'a' }] }
    localStorage.clear()
    mount(fakeApi(groups))
    await screen.findByText('사라진 프로젝트 1개를 정리했어요')
    await fireEvent.click(screen.getByRole('button', { name: '닫기' }))
    expect(screen.queryByText('사라진 프로젝트 1개를 정리했어요')).toBeNull()
    cleanup()
    mount(fakeApi(groups))
    await screen.findByText('아직 붙은 보드가 없어요')
    expect(screen.queryByText('사라진 프로젝트 1개를 정리했어요')).toBeNull()
    localStorage.clear()
  })

  it('a cancelled folder pick does nothing', async () => {
    const api = fakeApi(empty)
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: '폴더 고르기' }))
    await waitFor(() => expect(api.pickFolder).toHaveBeenCalled())
    expect(api.initProject).not.toHaveBeenCalled()
  })

  it('empty state shows the install line when stickypane is missing', async () => {
    const api = fakeApi(empty, { status: vi.fn(async () => ({ found: false, bridge: false })) })
    mount(api)
    expect(await screen.findByText('brew install --cask LeeSwallow/tap/stickypane')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: '복사' }))
    expect(api.copy).toHaveBeenCalledWith('brew install --cask LeeSwallow/tap/stickypane')
    expect(await screen.findByText('복사했어요')).toBeTruthy()
  })

  it('an old stickypane gets an update line, even when boards are listed', async () => {
    const api = fakeApi(
      { Active: [row(1, 'a', at(0, 14, 58))], Days: null, Removed: null },
      { status: vi.fn(async () => ({ found: true, bridge: false, version: '0.1.1' })) },
    )
    mount(api)
    expect(await screen.findByText(/오래되어/)).toBeTruthy()
    expect(screen.getByText('brew upgrade --cask LeeSwallow/tap/stickypane')).toBeTruthy()
  })

  it('sessions are grouped active, today, yesterday, by date', async () => {
    const api = fakeApi({
      Active: [row(1, 'alpha', at(0, 14, 58))],
      Days: [
        { Date: '2026-10-28', Rows: [row(2, 'bravo', at(0, 9, 5))] },
        { Date: '2026-10-27', Rows: [row(3, 'charlie', at(1, 11))] },
        { Date: '2026-10-20', Rows: [row(4, 'delta', at(8, 8))] },
      ],
      Removed: null,
    })
    mount(api)
    await screen.findByText('alpha', { selector: '.title' })
    const heads = screen.getAllByRole('heading', { level: 2 }).map((h) => h.textContent)
    expect(heads).toEqual(['지금 움직이는 중', '오늘', '어제', '10월 20일'])
    const titles = [...document.querySelectorAll('.title')].map((e) => e.textContent)
    expect(titles).toEqual(['alpha', 'bravo', 'charlie', 'delta'])
  })

  it('undo restores a removed project', async () => {
    const groups: Groups = {
      Active: null,
      Days: null,
      Removed: [
        { ID: 7, Path: '/gone/a', Title: 'a' },
        { ID: 9, Path: '/gone/b', Title: 'b' },
      ],
    }
    let restored = 0
    const api = fakeApi(groups, {
      sessions: vi.fn(async () => (restored ? { ...empty } : groups)),
      restore: vi.fn(async () => {
        restored++
      }),
    })
    mount(api)
    expect(await screen.findByText('사라진 프로젝트 2개를 정리했어요')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: '되돌리기' }))
    await waitFor(() => expect(api.restore).toHaveBeenCalledTimes(2))
    expect(api.restore).toHaveBeenCalledWith(7)
    expect(api.restore).toHaveBeenCalledWith(9)
    await waitFor(() => expect(screen.queryByText('사라진 프로젝트 2개를 정리했어요')).toBeNull())
    expect((api.sessions as ReturnType<typeof vi.fn>).mock.calls.length).toBeGreaterThan(1)
  })

  it('notices show as notes at the top', async () => {
    const api = fakeApi(empty, {
      notices: vi.fn(async () => [{ code: 'hotkey_failed', detail: 'Alt+Space: taken' }]),
    })
    mount(api)
    const note = await screen.findByRole('status')
    expect(within(note).getByText(/Alt\+Space: taken/)).toBeTruthy()
  })

  it('the one-time tray word shows with a way to quit, and can be put away', async () => {
    const api = fakeApi(empty, { notices: vi.fn(async () => [{ code: 'stayInTray', detail: '' }]) })
    mount(api)
    const note = await screen.findByRole('status')
    expect(within(note).getByText('트레이에 남아 있어요')).toBeTruthy()
    expect(within(note).getByRole('button', { name: 'stickypane 종료' })).toBeTruthy()
    await fireEvent.click(within(note).getByRole('button', { name: '닫기' }))
    expect(screen.queryByText('트레이에 남아 있어요')).toBeNull()
  })

  it('an index that cannot be read says so instead of an empty list', async () => {
    const api = fakeApi(empty, {
      sessions: vi.fn(async () => {
        throw new Error('index: locked')
      }),
    })
    mount(api)
    expect(await screen.findByText(/index: locked/)).toBeTruthy()
    expect(screen.queryByText('아직 붙은 보드가 없어요')).toBeNull()
  })

  it('draws in English when overlay.json says en', async () => {
    const api = fakeApi(empty, { prefs: vi.fn(async () => ({ hotkey: '', language: 'en', mode: '' })) })
    mount(api)
    expect(await screen.findByText('No boards attached yet')).toBeTruthy()
  })

  it('mode dark or light picks the stickypane palette', async () => {
    const api = fakeApi(empty, { prefs: vi.fn(async () => ({ hotkey: '', language: 'ko', mode: 'light' })) })
    mount(api)
    await screen.findByText('아직 붙은 보드가 없어요')
    expect(document.documentElement.dataset.mode).toBe('light')
  })
})

describe('fallback palettes', () => {
  it('match the stickypane pair of the theme list', () => {
    expect(THEMES[0]).toBe(STICKYPANE_DARK)
    expect(THEMES[1]).toBe(STICKYPANE_LIGHT)
  })
})
