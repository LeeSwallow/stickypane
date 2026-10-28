// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage, t } from '../i18n'
import type { Settings } from '../board/model'
import { setAppPref, type AppPrefs, type SettingsApi } from '../services/settings'
import SettingsDialog from './SettingsDialog.svelte'

const settings: Settings = {
  theme: 'nord', style: 'doodle', view: 'both', language: { choice: 'en', resolved: 'en' }, tab: '', time: '', editor: '', stale: '2h',
}

function fakeApi(over: Partial<SettingsApi> = {}) {
  const board: unknown[] = []
  let prefs: AppPrefs = { hotkey: 'Option+Space', language: 'ko', mode: '' }
  const saved: AppPrefs[] = []
  const api: SettingsApi = {
    settings: vi.fn(async () => settings),
    setSettings: vi.fn(async (p) => void board.push(p)),
    appPrefs: vi.fn(async () => prefs),
    setAppPrefs: vi.fn(async (p) => {
      saved.push(p)
      prefs = p
    }),
    ...over,
  }
  return { api, board, saved }
}

beforeEach(() => {
  vi.stubGlobal(
    'ResizeObserver',
    class {
      observe() {}
      unobserve() {}
      disconnect() {}
    },
  )
  setLanguage('en')
})
afterEach(cleanup)

const show = (api: SettingsApi, over: Record<string, unknown> = {}) => {
  const onmode = vi.fn()
  const onchanged = vi.fn()
  render(SettingsDialog, { props: { open: true, api, settings, mode: 'system', hotkey: 'Option+Space', version: '0.4.0', onmode, onchanged, ...over } })
  return { onmode, onchanged }
}

describe('setAppPref', () => {
  it('changes one of the app\'s settings and keeps the rest', async () => {
    const { api, saved } = fakeApi()
    expect(await setAppPref(api, { mode: 'dark' })).toEqual({ hotkey: 'Option+Space', language: 'ko', mode: 'dark' })
    expect(saved).toEqual([{ hotkey: 'Option+Space', language: 'ko', mode: 'dark' }])
  })
})

describe('the settings', () => {
  it('a board setting goes to the board, through the bridge, and the board is read again', async () => {
    const { api, board, saved } = fakeApi()
    const { onchanged } = show(api)
    await fireEvent.click(screen.getByRole('radio', { name: t('settings.style.neat') }))
    await waitFor(() => expect(board).toEqual([{ style: 'neat' }]))
    expect(onchanged).toHaveBeenCalled()
    await fireEvent.change(screen.getByLabelText(t('settings.theme')), { target: { value: 'dracula' } })
    await waitFor(() => expect(board).toContainEqual({ theme: 'dracula' }))
    expect(saved).toEqual([])
  })

  it('light or dark is the app\'s, shown at once and kept in overlay.json', async () => {
    const { api, board, saved } = fakeApi()
    const { onmode } = show(api)
    const modes = within(screen.getByRole('radiogroup', { name: t('settings.mode') }))
    await fireEvent.click(modes.getByRole('radio', { name: t('settings.mode.dark') }))
    expect(onmode).toHaveBeenCalledWith('dark')
    await waitFor(() => expect(saved).toEqual([{ hotkey: 'Option+Space', language: 'ko', mode: 'dark' }]))
    await fireEvent.click(modes.getByRole('radio', { name: t('settings.mode.system') }))
    await waitFor(() => expect(saved.at(-1)?.mode).toBe(''))
    expect(board).toEqual([])
  })

  it('a hotkey the system will not give is said under the field', async () => {
    const { api } = fakeApi({ setAppPrefs: async () => Promise.reject(new Error('Error: Cmd+Q is taken')) })
    show(api)
    const field = screen.getByLabelText(t('settings.hotkey'))
    await fireEvent.input(field, { target: { value: 'Cmd+Q' } })
    await fireEvent.click(screen.getByRole('button', { name: t('settings.save') }))
    await waitFor(() => expect(document.body.textContent).toContain('Cmd+Q is taken'))
    expect(field.getAttribute('aria-invalid')).toBe('true')
  })

  it('names the bridge\'s version, and the terminal buttons wait for the next version', () => {
    show(fakeApi().api)
    expect(document.body.textContent).toContain('stickypane 0.4.0')
    expect((screen.getByRole('button', { name: t('settings.installCli') }) as HTMLButtonElement).disabled).toBe(true)
  })
})
