// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import { THEMES } from '../demo/themes'
import type { Board } from '../board/model'
import type { AppApi, AppPrefs } from '../services/settings'
import { fakeSetup } from './fakeSetup'
import SettingsWindow from './SettingsWindow.svelte'

const dark = THEMES.find((t) => t.name === 'stickypane-dark')!

function fakeBoard(over: Partial<Board['settings']> = {}, ready = true) {
  let settings = { theme: 'stickypane-dark', style: 'doodle', view: 'overlay', language: { choice: 'auto', resolved: 'ko' }, tab: '', time: '', editor: '', stale: '' , ...over }
  return {
    status: vi.fn(async () => ({ window: 3, path: '/p/my-app', state: ready ? 'ready' : 'closed', pinned: false, version: '0.4.0' })),
    load: vi.fn(async () => ({ title: 'my-app', tabs: [], settings, palette: dark }) as unknown as Board),
    themes: vi.fn(async () => THEMES),
    setSettings: vi.fn(async (p: Record<string, unknown>) => {
      settings = { ...settings, ...(p as object), language: p.language ? { choice: p.language as string, resolved: 'ko' } : settings.language }
    }),
    layout: vi.fn(async () => ({ mode: 'canvas' as const, snap: true, places: {}, torn: {} })),
    setMode: vi.fn(async () => {}),
  }
}

function fakeApp(over: Partial<AppApi> = {}) {
  let prefs: AppPrefs = { hotkey: 'Option+Space', language: 'ko', mode: 'dark', motion: 'reduce', pinNew: false }
  const api: AppApi = {
    prefs: vi.fn(async () => prefs),
    setPrefs: vi.fn(async (p: AppPrefs) => void (prefs = p)),
    onPrefs: vi.fn(() => () => {}),
    checkHotkey: vi.fn(async (accel: string) => (accel === 'Ctrl+Option+K' ? { ok: false, accel, code: 'taken' } : { ok: true, accel })),
    autostart: vi.fn(async () => ({ supported: false, enabled: false })),
    setAutostart: vi.fn(async (on: boolean) => ({ supported: true, enabled: on })),
    about: vi.fn(async () => ({ app: '0.1.0', stickypane: { found: true, bridge: true, path: '/x/stickypane', version: '0.4.0' } })),
    onBoard: vi.fn(() => () => {}),
    openURL: vi.fn(async () => {}),
    ...over,
  }
  return { api, get prefs() { return prefs } }
}

beforeEach(() => {
  setLanguage('ko')
  globalThis.ResizeObserver ??= class {
    observe() {}
    unobserve() {}
    disconnect() {}
  } as unknown as typeof ResizeObserver
  window.matchMedia = ((q: string) => ({ matches: true, media: q, addEventListener: () => {}, removeEventListener: () => {} })) as unknown as typeof window.matchMedia
})
afterEach(cleanup)

const open = (board = fakeBoard(), app = fakeApp(), section: 'board' | 'app' | 'cli' | 'about' = 'board') =>
  render(SettingsWindow, { board, app: app.api, setup: fakeSetup().api, section })

describe('the settings window', () => {
  it('opening it changes nothing', async () => {
    const board = fakeBoard()
    const app = fakeApp()
    open(board, app)
    await screen.findByRole('radiogroup', { name: '테마' })
    await new Promise((r) => setTimeout(r, 300))
    expect(board.setSettings).not.toHaveBeenCalled()
    expect(board.setMode).not.toHaveBeenCalled()
    expect(app.api.setPrefs).not.toHaveBeenCalled()
  })

  it('the board: themes as cards in their own colours; picking one keeps the half the screen shows', async () => {
    const board = fakeBoard()
    open(board)
    const group = await screen.findByRole('radiogroup', { name: '테마' })
    const cards = within(group).getAllByRole('radio')
    // auto, then six families for the ten palettes
    expect(cards).toHaveLength(7)
    expect(within(group).getByRole('radio', { name: /stickypane/ }).getAttribute('aria-checked')).toBe('true')
    await fireEvent.click(within(group).getByRole('radio', { name: /Catppuccin/ }))
    await waitFor(() => expect(board.setSettings).toHaveBeenCalledWith({ theme: 'catppuccin-mocha' }))
    await waitFor(() => expect(within(group).getByRole('radio', { name: /Catppuccin/ }).getAttribute('aria-checked')).toBe('true'))
  })

  it('the arrows move through the theme cards and pick, as radios do', async () => {
    const board = fakeBoard()
    open(board)
    const group = await screen.findByRole('radiogroup', { name: '테마' })
    const chosen = within(group).getByRole('radio', { name: /stickypane/ })
    chosen.focus()
    await fireEvent.keyDown(group, { key: 'ArrowRight' })
    await waitFor(() => expect(board.setSettings).toHaveBeenCalledWith({ theme: 'catppuccin-mocha' }))
  })

  it('style, language, layout and where it opens each go where they are kept', async () => {
    const board = fakeBoard()
    open(board)
    await fireEvent.click(await screen.findByRole('radio', { name: /단정/ }))
    expect(board.setSettings).toHaveBeenCalledWith({ style: 'neat' })
    await fireEvent.click(screen.getByRole('radio', { name: 'English' }))
    expect(board.setSettings).toHaveBeenCalledWith({ language: 'en' })
    await fireEvent.click(screen.getByRole('radio', { name: '그리드' }))
    expect(board.setMode).toHaveBeenCalledWith('grid')
    await fireEvent.click(screen.getByRole('radio', { name: '둘 다' }))
    expect(board.setSettings).toHaveBeenCalledWith({ view: 'both' })
  })

  it('opened with no board, says how to get one', async () => {
    open(fakeBoard({}, false))
    expect(await screen.findByText('고를 보드가 없어요')).toBeTruthy()
    expect(screen.queryByRole('radiogroup', { name: '테마' })).toBeNull()
  })

  it('this computer: a change keeps every other setting', async () => {
    const app = fakeApp()
    open(fakeBoard(), app, 'app')
    await fireEvent.click(await screen.findByRole('radio', { name: /라이트/ }))
    await waitFor(() => expect(app.prefs).toMatchObject({ mode: 'light', hotkey: 'Option+Space', motion: 'reduce', language: 'ko' }))
    await fireEvent.click(screen.getByRole('switch', { name: '새 보드 창을 항상 위에' }))
    await waitFor(() => expect(app.prefs).toMatchObject({ pinNew: true, mode: 'light' }))
    await fireEvent.click(screen.getByRole('radio', { name: '끄기' }))
    await waitFor(() => expect(app.prefs.motion).toBe('off'))
  })

  it('the hotkey is recorded from the keys pressed, checked, and said when it cannot be had', async () => {
    const app = fakeApp()
    open(fakeBoard(), app, 'app')
    const field = await screen.findByRole('button', { name: /단축키/ })
    await fireEvent.click(field)
    await fireEvent.keyDown(field, { code: 'KeyK', key: 'k', ctrlKey: true, altKey: true })
    expect(await screen.findByRole('alert')).toHaveProperty('textContent', '⌃⌥K는 다른 앱이 이미 써요.')
    expect(app.api.setPrefs).not.toHaveBeenCalled()
    await fireEvent.click(field)
    await fireEvent.keyDown(field, { code: 'KeyJ', key: 'j', metaKey: true, shiftKey: true })
    await waitFor(() => expect(app.prefs.hotkey).toBe('Shift+Cmd+J'))
    expect(await screen.findByText('저장했어요. ⇧⌘J를 눌러 보세요.')).toBeTruthy()
    // Esc stops listening and keeps the hotkey
    await fireEvent.click(field)
    await fireEvent.keyDown(field, { code: 'Escape', key: 'Escape' })
    expect(app.api.checkHotkey).toHaveBeenCalledTimes(2)
  })

  it('opening at login is offered only where the desktop can', async () => {
    open(fakeBoard(), fakeApp(), 'app')
    await screen.findByText('단축키')
    expect(screen.queryByText('로그인할 때 열기')).toBeNull()
    cleanup()
    const app = fakeApp({ autostart: vi.fn(async () => ({ supported: true, enabled: false })) })
    open(fakeBoard(), app, 'app')
    await fireEvent.click(await screen.findByRole('switch', { name: '로그인할 때 열기' }))
    expect(app.api.setAutostart).toHaveBeenCalledWith(true)
  })

  it('the sections are a vertical tab list; the arrows move between them', async () => {
    open()
    const list = await screen.findByRole('tablist')
    await screen.findByRole('tab', { name: '보드' })
    expect(list.getAttribute('aria-orientation')).toBe('vertical')
    const tabs = within(list).getAllByRole('tab')
    expect(tabs.map((t) => t.textContent?.trim())).toEqual(['보드', '이 컴퓨터', '터미널과 에이전트', '정보'])
    tabs[0].focus()
    await fireEvent.keyDown(list, { key: 'ArrowDown' })
    expect(within(list).getByRole('tab', { name: '이 컴퓨터' }).getAttribute('aria-selected')).toBe('true')
    await fireEvent.keyDown(list, { key: 'End' })
    expect(await screen.findByText('0.1.0')).toBeTruthy()
    expect(screen.getByText('0.4.0')).toBeTruthy()
  })
})
