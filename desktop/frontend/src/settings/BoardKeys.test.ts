// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import { fakeKeys } from './fakeKeys'
import Keys from './BoardKeys.svelte'

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

const open = (f = fakeKeys(), goto = vi.fn()) => {
  render(Keys, { api: f.api, hotkey: 'Option+Space', goto, pause: 20 })
  return { f, goto }
}
const row = async (words: string) => (await screen.findByText(words)).closest('li') as HTMLElement

describe('단축키', () => {
  it('lists the actions by group, in the app language, with their keys as the board shows them', async () => {
    const { f } = open(fakeKeys({ 'note-archive': ['X'] }))
    expect(await screen.findByRole('heading', { name: '이동' })).toBeTruthy()
    expect(screen.getByRole('heading', { name: '탭' })).toBeTruthy()
    expect(f.state.calls[0]).toBe('keys ko')
    const first = await row('첫 노트')
    expect(within(first).getByText('gg')).toBeTruthy()
    const archive = await row('보관함으로')
    expect(within(archive).getByText('바꿈')).toBeTruthy()
    expect(within(archive).getByText('X')).toBeTruthy()
    expect(within(archive).getByRole('button', { name: '기본값으로 (x)' })).toBeTruthy()
    expect(within(first).queryByText('바꿈')).toBeNull()
    expect(within(first).queryByRole('button', { name: /기본값으로/ })).toBeNull()
  })

  it('names the app hotkey first, with the way to change it', async () => {
    const { goto } = open()
    expect(screen.getByText('⌥Space')).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: '이 컴퓨터로' }))
    expect(goto).toHaveBeenCalledWith('app')
  })

  it('finds an action by its words, its name or a key', async () => {
    open()
    await screen.findByText('첫 노트')
    const search = screen.getByLabelText('동작·키 찾기')
    await fireEvent.input(search, { target: { value: '탭' } })
    expect(screen.queryByText('첫 노트')).toBeNull()
    expect(screen.getByText('다음 탭')).toBeTruthy()
    await fireEvent.input(search, { target: { value: 'gg' } })
    expect(screen.getByText('첫 노트')).toBeTruthy()
    expect(screen.queryByText('다음 탭')).toBeNull()
    await fireEvent.input(search, { target: { value: 'zzz' } })
    expect(screen.getByText('맞는 동작이 없어요')).toBeTruthy()
  })

  it('changes an action’s keys through the board, and says why it refuses', async () => {
    const { f } = open()
    const archive = await row('보관함으로')
    await fireEvent.click(within(archive).getByRole('button', { name: '보관함으로 키 바꾸기' }))
    const field = within(archive).getByLabelText('키') as HTMLInputElement
    expect(field.value).toBe('x')
    await fireEvent.input(field, { target: { value: 'q' } })
    await fireEvent.click(within(archive).getByRole('button', { name: '저장' }))
    await waitFor(() => expect(within(archive).getByText("q는 이미 '끝내기' 키예요. 다른 키를 골라 주세요.")).toBeTruthy())
    expect(f.state.file['note-archive']).toBeUndefined()
    await fireEvent.input(field, { target: { value: 'X, d x' } })
    await fireEvent.click(within(archive).getByRole('button', { name: '저장' }))
    await waitFor(() => expect(within(archive).queryByLabelText('키')).toBeNull())
    expect(f.state.file['note-archive']).toEqual(['X', 'd x'])
    expect(within(archive).getByText('dx')).toBeTruthy()
    expect(within(archive).getByText('바꿈')).toBeTruthy()
  })

  it('takes one change back, or every change after asking', async () => {
    const { f } = open(fakeKeys({ 'note-archive': ['X'], quit: ['Q'] }))
    const archive = await row('보관함으로')
    await fireEvent.click(within(archive).getByRole('button', { name: '기본값으로 (x)' }))
    await waitFor(() => expect(within(archive).queryByText('바꿈')).toBeNull())
    expect(f.state.file).toEqual({ quit: ['Q'] })
    await fireEvent.click(screen.getByRole('button', { name: '모두 기본값으로' }))
    expect(f.state.calls).not.toContain('reset ')
    const dialog = await screen.findByRole('alertdialog')
    await fireEvent.click(within(dialog).getByRole('button', { name: '모두 기본값으로' }))
    await waitFor(() => expect(f.state.file).toEqual({}))
    expect(f.state.calls).toContain('reset ')
    expect((screen.getByRole('button', { name: '모두 기본값으로' }) as HTMLButtonElement).disabled).toBe(true)
  })

  it('leaves an action without a key only after asking', async () => {
    const { f } = open()
    const theme = await row('다음 테마')
    await fireEvent.click(within(theme).getByRole('button', { name: '다음 테마 키 바꾸기' }))
    await fireEvent.input(within(theme).getByLabelText('키'), { target: { value: '  ' } })
    await fireEvent.click(within(theme).getByRole('button', { name: '저장' }))
    expect(f.state.calls.some((c) => c.startsWith('set'))).toBe(false)
    const ask = await within(theme).findByRole('alertdialog')
    await fireEvent.click(within(ask).getByRole('button', { name: '키 빼기' }))
    await waitFor(() => expect(f.state.file['theme-next']).toEqual([]))
    expect(within(theme).getByText('없음')).toBeTruthy()
  })

  it('records what is pressed: a pause ends a key, keys before it are a sequence', async () => {
    open()
    const archive = await row('보관함으로')
    await fireEvent.click(within(archive).getByRole('button', { name: '보관함으로 키 바꾸기' }))
    await fireEvent.click(within(archive).getByRole('button', { name: '키 누르기' }))
    const field = within(archive).getByLabelText('키') as HTMLInputElement
    await fireEvent.keyDown(field, { key: 'd', code: 'KeyD' })
    await fireEvent.keyDown(field, { key: 'Shift', code: 'ShiftLeft', shiftKey: true })
    await fireEvent.keyDown(field, { key: 'X', code: 'KeyX', shiftKey: true })
    expect(field.value).toBe('d X')
    await new Promise((r) => setTimeout(r, 40))
    await fireEvent.keyDown(field, { key: 'ㅓ', code: 'KeyJ' })
    await fireEvent.keyDown(field, { key: 'd', code: 'KeyD', ctrlKey: true })
    expect(field.value).toBe('d X, j ctrl+d')
    await fireEvent.click(within(archive).getByRole('button', { name: '멈추기' }))
    expect(within(archive).getByRole('button', { name: '키 누르기' })).toBeTruthy()
  })

  it('without a stickypane, says how to get one', async () => {
    const { goto } = open(fakeKeys({}, { keys: async () => ({ bindings: [], missing: true }) }))
    expect(await screen.findByText(/stickypane을 설치하면/)).toBeTruthy()
    await fireEvent.click(screen.getByRole('button', { name: '터미널과 에이전트로' }))
    expect(goto).toHaveBeenCalledWith('cli')
  })

  it('an older answer, late, does not take the place of the newer one', async () => {
    const base = fakeKeys()
    const slow = fakeKeys({}, { keys: async (lang) => (lang === 'ko' && (await new Promise((r) => setTimeout(r, 60))), base.api.keys(lang)) })
    open(slow)
    setLanguage('en')
    expect(await screen.findByText('first note')).toBeTruthy()
    await new Promise((r) => setTimeout(r, 100))
    expect(screen.queryByText('첫 노트')).toBeNull()
  })

  it('asks again in the language the app turns to', async () => {
    const { f } = open()
    await screen.findByText('첫 노트')
    setLanguage('en')
    expect(await screen.findByText('first note')).toBeTruthy()
    expect(f.state.calls).toContain('keys en')
  })
})
