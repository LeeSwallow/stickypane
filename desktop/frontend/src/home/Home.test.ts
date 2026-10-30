// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage, t } from '../i18n'
import type { Groups, HomeApi, Row } from '../services/home'
import { fakeSetup } from '../settings/fakeSetup'
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
    peeks: vi.fn(async () => ({})),
    openSettings: vi.fn(async () => {}),
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

const quiet = () => () => {}
const mount = (api: HomeApi, setup = fakeSetup().api) => render(Home, { api, setup, now: () => NOW, onPrefs: quiet })

describe('home', () => {
  it('the ⋯ menu and ⌘, open the settings window (03 §3.12)', async () => {
    const api = fakeApi(empty)
    mount(api)
    await fireEvent.click(screen.getByRole('button', { name: t('ui.more') }))
    await fireEvent.click(await screen.findByRole('menuitem', { name: new RegExp(t('settings.open')) }))
    expect(api.openSettings).toHaveBeenCalledWith('app')
    await fireEvent.keyDown(window, { key: ',', metaKey: true })
    expect(api.openSettings).toHaveBeenCalledTimes(2)
  })

  it('the welcome says where each step stands, and does steps 2 and 3 (08 §8.2)', async () => {
    const f = fakeSetup({}, { connectPlan: vi.fn(async () => ({ command: 'stickypane setup --dry-run', output: 'Nothing to do: the board plugin is set up for every agent found.' })) })
    const api = fakeApi(empty)
    mount(api, f.api)
    expect(await screen.findByText('첫 보드까지 세 걸음')).toBeTruthy()
    // ② is done: the agents are connected
    expect(await screen.findByText('연결됨')).toBeTruthy()
    // ③ is not: install puts the command on the PATH
    await fireEvent.click(await screen.findByRole('button', { name: '설치' }))
    expect(f.api.install).toHaveBeenCalledWith('user', false)
    expect(await screen.findByText('됐어요')).toBeTruthy()
  })

  it('a step not done goes on in the settings window', async () => {
    const api = fakeApi(empty)
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: '연결…' }))
    expect(api.openSettings).toHaveBeenCalledWith('cli')
  })

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
    // the home folder is ~
    expect(screen.getByText('~/stickypane')).toBeTruthy()
    expect(screen.getByText('지금 작업 중')).toBeTruthy()
    // no welcome once there are boards
    expect(screen.queryByText('첫 보드까지 세 걸음')).toBeNull()
  })

  it('a card shows what the board looked like when last opened here', async () => {
    const api = fakeApi(
      { Active: [row(1, 'stickypane', at(0, 14, 58))], Days: [{ Date: '2026-10-28', Rows: [row(2, 'other', at(0, 9))] }], Removed: null },
      { peeks: vi.fn(async () => ({ '/home/min/stickypane': { notes: 6, stalled: 1, agents: ['claude'], colors: ['#1', '#2', '#3', '#4', '#5', '#6'] } })) },
    )
    mount(api)
    expect(await screen.findByText('노트 6')).toBeTruthy()
    expect(screen.getByText('멈춤 1')).toBeTruthy()
    expect(screen.getByText('@claude')).toBeTruthy()
    // a board never opened here says so
    expect(screen.getByText('아직 여기서 안 열어 봤어요')).toBeTruthy()
  })

  it('the search narrows the cards by name and folder', async () => {
    const api = fakeApi({ Active: [row(1, 'alpha', at(0, 14, 58)), row(2, 'bravo', at(0, 14, 50))], Days: null, Removed: null })
    mount(api)
    await screen.findByText('alpha', { selector: '.title' })
    await fireEvent.input(screen.getByRole('searchbox', { name: '보드 찾기' }), { target: { value: 'brav' } })
    expect(screen.queryByText('alpha', { selector: '.title' })).toBeNull()
    expect(screen.getByText('bravo', { selector: '.title' })).toBeTruthy()
    await fireEvent.input(screen.getByRole('searchbox', { name: '보드 찾기' }), { target: { value: 'zzz' } })
    expect(screen.getByText('“zzz”에 맞는 보드가 없어요.')).toBeTruthy()
  })

  it('clicking a row opens that project', async () => {
    const api = fakeApi({ Active: [row(1, 'stickypane', at(0, 14, 58))], Days: null, Removed: null })
    mount(api)
    await fireEvent.click(await screen.findByRole('button', { name: 'stickypane 열기' }))
    expect(api.openProject).toHaveBeenCalledWith('/home/min/stickypane')
  })

  it('step 1 picks a folder and asks before it makes a board there', async () => {
    const api = fakeApi(empty, { pickFolder: vi.fn(async () => '/home/min/new') })
    mount(api)
    expect(await screen.findByText('첫 보드까지 세 걸음')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: '폴더 고르기' }))
    // no board is made until the person says so
    const ask = await screen.findByRole('alertdialog')
    expect(within(ask).getByText('이 폴더에 보드를 만들까요?')).toBeTruthy()
    // what making it does, then where
    expect(within(ask).getByText(/\.sticky\/ 폴더를 만들어요[\s\S]*\/home\/min\/new$/)).toBeTruthy()
    expect(api.initProject).not.toHaveBeenCalled()
    await fireEvent.click(within(ask).getByRole('button', { name: '보드 만들기' }))
    // the welcome note in the language the window draws in
    await waitFor(() => expect(api.initProject).toHaveBeenCalledWith('/home/min/new', 'ko'))
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
    await fireEvent.click(within(await screen.findByRole('alertdialog')).getByRole('button', { name: '보드 만들기' }))
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
    await screen.findByText('없어진 프로젝트 1개를 목록에서 뺐어요')
    await fireEvent.click(screen.getByRole('button', { name: '닫기' }))
    expect(screen.queryByText('없어진 프로젝트 1개를 목록에서 뺐어요')).toBeNull()
    cleanup()
    mount(fakeApi(groups))
    await screen.findByText('첫 보드까지 세 걸음')
    expect(screen.queryByText('없어진 프로젝트 1개를 목록에서 뺐어요')).toBeNull()
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

  it('boards are grouped now, today and earlier; an earlier card names its day', async () => {
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
    expect(heads).toEqual(['지금 작업 중', '오늘', '이전'])
    const titles = [...document.querySelectorAll('.title')].map((e) => e.textContent)
    expect(titles).toEqual(['alpha', 'bravo', 'charlie', 'delta'])
    expect(screen.getByText(/^어제 11:00$/)).toBeTruthy()
    expect(screen.getByText(/^10월 20일 08:00$/)).toBeTruthy()
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
    expect(await screen.findByText('없어진 프로젝트 2개를 목록에서 뺐어요')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: '되돌리기' }))
    await waitFor(() => expect(api.restore).toHaveBeenCalledTimes(2))
    expect(api.restore).toHaveBeenCalledWith(7)
    expect(api.restore).toHaveBeenCalledWith(9)
    await waitFor(() => expect(screen.queryByText('없어진 프로젝트 2개를 목록에서 뺐어요')).toBeNull())
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
    expect(within(note).getByText('창을 닫아도 stickypane은 계속 실행돼요')).toBeTruthy()
    expect(within(note).getByRole('button', { name: 'stickypane 종료' })).toBeTruthy()
    await fireEvent.click(within(note).getByRole('button', { name: '닫기' }))
    expect(screen.queryByText('창을 닫아도 stickypane은 계속 실행돼요')).toBeNull()
  })

  it('an index that cannot be read says so instead of an empty list', async () => {
    const api = fakeApi(empty, {
      sessions: vi.fn(async () => {
        throw new Error('index: locked')
      }),
    })
    mount(api)
    expect(await screen.findByText(/index: locked/)).toBeTruthy()
    expect(screen.queryByText('첫 보드까지 세 걸음')).toBeNull()
  })

  it('draws in English when overlay.json says en', async () => {
    const api = fakeApi(empty, { prefs: vi.fn(async () => ({ hotkey: '', language: 'en', mode: '' })) })
    mount(api)
    expect(await screen.findByText('Three steps to your first board')).toBeTruthy()
  })

  it('mode dark or light picks the stickypane palette', async () => {
    const api = fakeApi(empty, { prefs: vi.fn(async () => ({ hotkey: '', language: 'ko', mode: 'light' })) })
    mount(api)
    await screen.findByText('첫 보드까지 세 걸음')
    expect(document.documentElement.dataset.mode).toBe('light')
  })
})

