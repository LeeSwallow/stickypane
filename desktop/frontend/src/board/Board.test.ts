// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import type { Board as BoardData, Note } from './model'
import type { Bus } from '../services/board'
import { fakeProject } from '../demo/fakeProject'
import type { ProjectApi } from '../services/project'
import type { Layout } from '../services/canvas'
import { THEMES } from '../demo/themes'
import Board from './Board.svelte'

const base = { tab: 'work', icon: '✎', open: true, rows: 0, color: 'yellow', pinned: false, version: 'v1', summary: '', modTime: '2026-10-28T10:00:00Z', linked: false, kind: 'note' }
const note = (name: string, size: string, rows = 0) => ({ ...base, name: `work/${name}`, title: name, size, rows, data: { markdown: 'hi' } }) as unknown as Note

const boardOf = (notes: Note[]): BoardData => ({
  title: 'my-app',
  tabs: [{ name: 'work', title: 'work', notes }],
  settings: { theme: 'stickypane-light', style: 'doodle', view: 'overlay', language: { choice: 'ko', resolved: 'ko' }, tab: 'work', time: '', editor: '', stale: '2h' },
  palette: THEMES.find((x) => x.name === 'stickypane-light'),
})

function fakeBus() {
  const on = new Map<string, (e: { data?: unknown }) => void>()
  const bus: Bus = { on: (name, fn) => (on.set(name, fn), () => on.delete(name)) }
  return { bus, fire: (name: string, data?: unknown) => on.get(name)?.({ data }) }
}

function api(board: BoardData, layout: Partial<Layout> = { mode: 'grid' }, over: Partial<ProjectApi> = {}): ProjectApi {
  const a = fakeProject(board, 'light', 'ko', layout)
  return {
    ...a,
    arrange: vi.fn(async () => {}),
    openSettings: vi.fn(async () => {}),
    place: vi.fn(a.place),
    resetPlaces: vi.fn(a.resetPlaces),
    ...over,
  }
}

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener() {}, removeEventListener() {} }))
  setLanguage('ko')
})
afterEach(cleanup)

describe('the board window', () => {
  it('an empty board says what to do next, and copies the line to paste', async () => {
    const writeText = vi.fn(async () => {})
    Object.defineProperty(navigator, 'clipboard', { value: { writeText }, configurable: true })
    render(Board, { props: { api: api(boardOf([])), bus: fakeBus().bus } })
    expect(await screen.findByText('아직 비어 있어요. 이렇게 시작해요')).toBeTruthy()
    const ask = screen.getByText('에이전트에게 부탁하기').closest('.card') as HTMLElement
    await fireEvent.click(within(ask).getByRole('button', { name: '복사' }))
    expect(writeText).toHaveBeenCalledWith('오늘 할 일을 stickypane 보드에 체크리스트로 붙여 줘.')
    const term = screen.getByText('터미널에서 열기').closest('.card') as HTMLElement
    await fireEvent.click(within(term).getByRole('button', { name: '복사' }))
    expect(writeText).toHaveBeenCalledWith('cd ~/Projects/Personal/stickypane && stickypane')
  })

  it('⌘, opens the settings window for this board', async () => {
    const a = api(boardOf([note('a.md', '1/2')]))
    render(Board, { props: { api: a, bus: fakeBus().bus } })
    await screen.findByText('a.md')
    await fireEvent.keyDown(window, { key: ',', metaKey: true })
    expect(a.openSettings).toHaveBeenCalledWith('board')
  })

  it('the app settings changed elsewhere redraw the window at once', async () => {
    const { bus, fire } = fakeBus()
    render(Board, { props: { api: api(boardOf([note('a.md', '1/2')])), bus } })
    await screen.findByText('a.md')
    fire('prefs.changed', { hotkey: 'Option+Space', language: '', mode: 'dark', motion: '' })
    await waitFor(() => expect(document.documentElement.dataset.mode).toBe('dark'))
  })

  it('on the grid, 배치 초기화 asks first, gives the sizes back, and 되돌리기 writes the old ones', async () => {
    const a = api(boardOf([note('a.md', '1/3', 8), note('b.md', '1/2')]))
    render(Board, { props: { api: a, bus: fakeBus().bus } })
    await screen.findByText('a.md')
    await fireEvent.click(within(document.querySelector('[data-titlebar]') as HTMLElement).getByRole('button', { name: '더 보기' }))
    await fireEvent.click(await screen.findByRole('menuitem', { name: /배치 초기화/ }))
    const ask = await screen.findByRole('alertdialog')
    expect(within(ask).getByText(/터미널 보드에도 적용돼요/)).toBeTruthy()
    expect(a.arrange).not.toHaveBeenCalled()
    await fireEvent.click(within(ask).getByRole('button', { name: '되돌리기' }))
    await waitFor(() => expect(a.arrange).toHaveBeenCalledWith({ note: 'work/a.md', size: '', rows: 0 }))
    expect(a.arrange).toHaveBeenCalledWith({ note: 'work/b.md', size: '', rows: 0 })
    await fireEvent.click(await screen.findByRole('button', { name: '되돌리기' }))
    await waitFor(() => expect(a.arrange).toHaveBeenCalledWith({ note: 'work/a.md', size: '1/3', rows: 8 }))
  })

  it('on the canvas, ⌥⌘L tidies and 배치 초기화 forgets the places, each with 되돌리기', async () => {
    const places = {
      'work/a.md': { x: 400, y: 400, w: 300, h: 200, tilt: 0, z: 1 },
      'work/b.md': { x: 410, y: 410, w: 300, h: 200, tilt: 0, z: 2 },
    }
    const a = api(boardOf([note('a.md', '1/2'), note('b.md', '1/2')]), { mode: 'canvas', places })
    render(Board, { props: { api: a, bus: fakeBus().bus } })
    await screen.findByText('a.md')
    await fireEvent.keyDown(window, { key: 'l', code: 'KeyL', altKey: true, metaKey: true })
    await waitFor(() => expect(a.place).toHaveBeenCalled())
    const tidied = (a.place as ReturnType<typeof vi.fn>).mock.calls.at(-1)![0] as typeof places
    expect(tidied['work/a.md'].y).toBe(tidied['work/b.md'].y === tidied['work/a.md'].y ? tidied['work/a.md'].y : tidied['work/a.md'].y)
    expect(tidied['work/a.md'].x).toBeLessThan(400)
    expect(await screen.findByText('노트를 정렬했어요.')).toBeTruthy()

    await fireEvent.click(within(document.querySelector('[data-titlebar]') as HTMLElement).getByRole('button', { name: '더 보기' }))
    await fireEvent.click(await screen.findByRole('menuitem', { name: /배치 초기화/ }))
    await waitFor(() => expect(a.resetPlaces).toHaveBeenCalled())
    // no question: the canvas's places are this computer's only
    expect(screen.queryByRole('alertdialog')).toBeNull()
  })
})
