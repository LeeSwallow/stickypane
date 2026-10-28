// @vitest-environment jsdom
import { cleanup, fireEvent, render, screen, waitFor, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage, t } from '../i18n'
import type { Board, Note } from '../board/model'
import type { Bus, Status } from '../services/board'
import { fakeProject } from '../demo/fakeProject'
import type { ProjectApi } from '../services/project'
import { THEMES } from '../lib/theme/palettes'
import TornNote from './TornNote.svelte'
import { fixture } from '../shapes/fixtures/fixtures'

const base = { tab: 'work', icon: '', open: true, rows: 0, color: 'yellow', pinned: false, version: 'v1', summary: '', modTime: '2026-10-28T10:00:00Z', linked: false, size: '1/2' }
const checklist = {
  ...base, name: 'work/release.md', title: 'Release', kind: 'checklist',
  data: { items: [{ text: 'write tests', checked: false }], prose: [], done: 0, total: 1 },
} as unknown as Note
const other = { ...base, name: 'work/b.md', title: 'Other', kind: 'note', data: { markdown: 'hi' } } as unknown as Note

const boardOf = (notes: Note[]): Board => ({
  title: 'stickypane',
  tabs: [{ name: 'work', title: 'work', notes }],
  settings: { theme: 'stickypane-light', style: 'doodle', view: 'overlay', language: { choice: 'en', resolved: 'en' }, tab: 'work', time: '', editor: '', stale: '2h' },
  palette: THEMES.find((x) => x.name === 'stickypane-light'),
})

// a bus that keeps its listeners, so a test can fire the shell's events
function fakeBus() {
  const on = new Map<string, (e: { data?: unknown }) => void>()
  const bus: Bus = { on: (name, fn) => (on.set(name, fn), () => on.delete(name)) }
  return { bus, fire: (name: string) => on.get(name)?.({}) }
}

function api(board: Board, over: Partial<ProjectApi> = {}): ProjectApi {
  const a = fakeProject(board, 'light', 'en', {}, 'work/release.md')
  const status = async (): Promise<Status> => ({ window: 7, path: '/p', state: 'ready', pinned: true, note: 'work/release.md' })
  return { ...a, status, act: vi.fn(async () => {}), putBack: vi.fn(async () => {}), pin: vi.fn(async () => {}), tearOff: vi.fn(async () => {}), ...over }
}

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  vi.stubGlobal('matchMedia', () => ({ matches: false, addEventListener() {}, removeEventListener() {} }))
  setLanguage('en')
})
afterEach(cleanup)

describe('a torn-off note', () => {
  it('draws its one note, and acts on it as the board does', async () => {
    const a = api(boardOf([checklist, other]))
    const { bus } = fakeBus()
    render(TornNote, { props: { api: a, bus, note: 'work/release.md' } })
    await screen.findByText('Release')
    expect(screen.queryByText('Other')).toBeNull()
    await fireEvent.click(screen.getByRole('checkbox', { name: /write tests/ }))
    expect(a.act).toHaveBeenCalledWith('work/release.md', 'v1', expect.anything())
    expect(document.title).toBe('Release')
  })

  it('is put back from its bar and from the note\'s own close', async () => {
    const a = api(boardOf([checklist]))
    render(TornNote, { props: { api: a, bus: fakeBus().bus, note: 'work/release.md' } })
    await screen.findByText('Release')
    const backs = screen.getAllByRole('button', { name: t('note.putBack') })
    await fireEvent.click(backs[0])
    expect(a.putBack).toHaveBeenCalledTimes(1)
    await fireEvent.click(screen.getByRole('button', { name: t('ui.close') }))
    expect(a.putBack).toHaveBeenCalledTimes(2)
  })

  it('keeps its window on top, or lets it go', async () => {
    const a = api(boardOf([checklist]))
    render(TornNote, { props: { api: a, bus: fakeBus().bus, note: 'work/release.md' } })
    await screen.findByText('Release')
    await fireEvent.click(screen.getByRole('button', { name: t('window.unpin') }))
    expect(a.pin).toHaveBeenCalledWith(false)
  })

  it('follows board.changed, and says so when its note is gone', async () => {
    let notes = [checklist]
    const a = api(boardOf(notes), { load: async () => boardOf(notes) })
    const { bus, fire } = fakeBus()
    render(TornNote, { props: { api: a, bus, note: 'work/release.md' } })
    await screen.findByText('Release')
    notes = [{ ...checklist, title: 'Release v2' }]
    fire('board.changed:7')
    await screen.findByText('Release v2')
    notes = []
    fire('board.changed:7')
    await screen.findByText(t('torn.gone'))
  })

  it('opens a linked note torn off too', async () => {
    const linking = { ...other, name: 'work/b.md', title: 'Links', data: { markdown: 'see [[release]]' } } as unknown as Note
    const a = api(boardOf([linking, checklist]), { status: async () => ({ window: 7, path: '/p', state: 'ready', pinned: true, note: 'work/b.md' }) })
    render(TornNote, { props: { api: a, bus: fakeBus().bus, note: 'work/b.md' } })
    const link = await waitFor(() => {
      const l = document.querySelector<HTMLElement>('[data-note]')
      if (!l) throw new Error('no link yet')
      return l
    })
    await fireEvent.click(link)
    expect(a.tearOff).toHaveBeenCalledWith('work/release.md')
  })
})

// #20's shapes, drawn from a real bridge's board.load, act the same torn off
describe('a torn-off note of every shape', () => {
  const real = fixture as Board
  const torn = (name: string, over: Partial<ProjectApi> = {}) => {
    const a = api(real, { status: async () => ({ window: 7, path: '/p', state: 'ready', pinned: true, note: name }), run: vi.fn(async () => ({ code: 0, took: 5 })), send: vi.fn(async () => ({})), ...over })
    render(TornNote, { props: { api: a, bus: fakeBus().bus, note: name } })
    return a
  }

  it('answers a form', async () => {
    const a = torn('deploy.md')
    await fireEvent.click(await screen.findByRole('button', { name: 'Deploy' }))
    expect(a.act).toHaveBeenCalledWith('deploy.md', expect.any(String), { name: 'submit', args: { label: 'Deploy' } })
  })

  it('says something in a chat', async () => {
    const a = torn('chat.md')
    const say = await screen.findByLabelText('Say something')
    await fireEvent.input(say, { target: { value: 'thanks' } })
    await fireEvent.keyDown(say, { key: 'Enter' })
    expect(a.act).toHaveBeenCalledWith('chat.md', expect.any(String), { name: 'say', args: { text: 'thanks' } })
  })

  it('runs a script after the question', async () => {
    const a = torn('build.sh')
    await fireEvent.click(await screen.findByRole('button', { name: 'Run' }))
    expect(a.run).not.toHaveBeenCalled()
    const ask = await screen.findByRole('alertdialog')
    await fireEvent.click(within(ask).getByRole('button', { name: t('run.button') }))
    await waitFor(() => expect(a.run).toHaveBeenCalledWith('build.sh'))
    await screen.findByText(t('run.done', { code: 0, took: 5 }))
  })

  it('sends the picked request of a .http note after the question', async () => {
    const a = torn('api.http')
    await screen.findByRole('button', { name: 'Send' })
    await fireEvent.click(document.querySelectorAll('.req')[1])
    await fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    const ask = await screen.findByRole('alertdialog')
    await fireEvent.click(within(ask).getByRole('button', { name: t('send.button') }))
    await waitFor(() => expect(a.send).toHaveBeenCalledWith('api.http', 2))
  })

  it('has no tear-off of its own', async () => {
    torn('plan.md')
    await screen.findByText('Plan')
    expect(screen.queryByRole('button', { name: t('ui.tearOff') })).toBeNull()
  })
})
