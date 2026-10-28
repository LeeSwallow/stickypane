// @vitest-environment jsdom
// Every shape drawn from what a real bridge says of it (fixtures/board.json),
// and what each does through the board context: the acts it sends, the
// run and the send it asks the window for.
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import Shape from './Shape.svelte'
import { context, fixture, sample } from './fixtures/fixtures'

// jsdom measures nothing: bind:clientWidth needs the observer to exist
globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver

beforeEach(() => setLanguage('en'))
afterEach(cleanup)

const draw = (name: string, over = {}) => render(Shape, { props: { note: sample(name) }, context: context(over) })

it('the fixture has a note of every shape', () => {
  const kinds = new Set(fixture.tabs.flatMap((t) => t.notes.map((n) => n.kind)))
  for (const k of ['note', 'board', 'checklist', 'log', 'chat', 'chart', 'form', 'script', 'api', 'html']) expect(kinds, k).toContain(k)
})

it('no shape of the fixture falls back to its summary', () => {
  for (const t of fixture.tabs)
    for (const n of t.notes) {
      const { container, unmount } = render(Shape, { props: { note: n }, context: context() })
      expect(container.querySelector('p.plain'), n.name).toBeNull()
      expect(container.querySelector('.unreadable'), n.name).toBeNull()
      unmount()
    }
})

describe('form', () => {
  it('draws questions, radios, checkboxes, the field, the buttons and that it waits', () => {
    draw('deploy.md')
    expect(screen.getByRole('heading', { name: 'Where to deploy?' })).toBeTruthy()
    expect(screen.getByRole('radio', { name: 'staging' }).getAttribute('aria-checked')).toBe('true')
    expect(screen.getByRole('radio', { name: 'production' }).getAttribute('aria-checked')).toBe('false')
    expect(screen.getByRole('checkbox', { name: 'tests' }).getAttribute('aria-checked')).toBe('true')
    expect(screen.getByRole('checkbox', { name: 'lint' }).getAttribute('aria-checked')).toBe('false')
    expect(screen.getByText('the one people use')).toBeTruthy()
    expect((screen.getByLabelText('Note') as HTMLInputElement).value).toBe('after lunch')
    expect(screen.getByRole('button', { name: 'Deploy' })).toBeTruthy()
    expect(screen.getByRole('button', { name: 'Cancel' })).toBeTruthy()
    expect(screen.getByRole('status').textContent).toContain('Waiting for an answer')
  })

  it('says a submitted form is answered, with its button', () => {
    draw('ready.md')
    expect(screen.getByRole('status').textContent).toContain('Answered: Go ✓')
    expect(screen.getByRole('button', { name: 'Go' }).getAttribute('aria-pressed')).toBe('true')
  })

  it('choose, fill and submit go to note.act with the line of the data', async () => {
    const act = vi.fn(async () => {})
    draw('deploy.md', { act })
    const lines = (sample('deploy.md').data as { lines: { text: string }[] }).lines
    const at = (text: string) => String(lines.findIndex((l) => l.text === text))
    await fireEvent.click(screen.getByRole('radio', { name: 'production' }))
    expect(act).toHaveBeenLastCalledWith(sample('deploy.md'), { name: 'choose', args: { line: at('production') } })
    await vi.waitFor(() => expect((screen.getByRole('checkbox', { name: 'lint' }) as HTMLButtonElement).disabled).toBe(false))
    await fireEvent.click(screen.getByRole('checkbox', { name: 'lint' }))
    expect(act).toHaveBeenLastCalledWith(sample('deploy.md'), { name: 'choose', args: { line: at('lint') } })
    await vi.waitFor(() => expect((screen.getByLabelText('Note') as HTMLInputElement).disabled).toBe(false))
    const field = screen.getByLabelText('Note') as HTMLInputElement
    field.focus()
    await fireEvent.input(field, { target: { value: ' after the standup ' } })
    await fireEvent.keyDown(field, { key: 'Enter' })
    await fireEvent.blur(field)
    expect(act).toHaveBeenLastCalledWith(sample('deploy.md'), { name: 'fill', args: { line: at('after lunch'), text: 'after the standup' } })
    await vi.waitFor(() => expect((screen.getByRole('button', { name: 'Deploy' }) as HTMLButtonElement).disabled).toBe(false))
    await fireEvent.click(screen.getByRole('button', { name: 'Deploy' }))
    expect(act).toHaveBeenLastCalledWith(sample('deploy.md'), { name: 'submit', args: { label: 'Deploy' } })
  })

  it('a field left as it was, or let go with Escape, writes nothing', async () => {
    const act = vi.fn(async () => {})
    draw('deploy.md', { act })
    const field = screen.getByLabelText('Note') as HTMLInputElement
    await fireEvent.blur(field)
    await fireEvent.input(field, { target: { value: 'changed' } })
    await fireEvent.keyDown(field, { key: 'Escape' })
    await fireEvent.blur(field)
    expect(act).not.toHaveBeenCalled()
  })
})

describe('chart', () => {
  it('bars: every label with its number as written', () => {
    const { container } = draw('tokens.md')
    expect(screen.getByText('Tokens by package')).toBeTruthy()
    const labels = [...container.querySelectorAll('.label')].map((e) => e.textContent)
    const values = [...container.querySelectorAll('.value')].map((e) => e.textContent)
    expect(labels).toEqual(['app', 'board', 'checklist', 'store'])
    expect(values).toEqual(['61', '31', '21', '16'])
  })

  it('heat: a calendar of days with its total and peak', () => {
    const { container } = draw('commits.md')
    const svg = container.querySelector('svg.heat')!
    expect(svg.getAttribute('aria-label')).toBe('A calendar of days: total 29, peak 8 on 2026-10-08')
    // seven days above zero, one zero shown as a dot
    expect(container.querySelectorAll('rect.cell')).toHaveLength(7)
    expect(container.querySelectorAll('circle.zero')).toHaveLength(1)
    expect(container.querySelector('rect.cell[data-level="3"] title')?.textContent).toBe('2026-10-08: 8')
    expect(container.querySelector('.legend')?.textContent).toContain('total 29 · peak 8 (2026-10-08)')
  })

  it('spark: a trend line box (drawn at its width)', () => {
    const { container } = draw('trend.md')
    expect(container.querySelector('.spark')).toBeTruthy()
  })
})

describe('log', () => {
  it('draws its lines and when it last grew', () => {
    const { container } = draw('worklog.md')
    expect([...container.querySelectorAll('.line')].map((e) => e.textContent)).toEqual(['03:14 tests passed', '03:14 started on review feedback', 'pushed the fix'])
    expect(screen.getByText(/^Last grew/)).toBeTruthy()
  })

  it('colours a run\'s command and how it ended', () => {
    const { container } = draw('build.log')
    expect(container.querySelector('.line.command')?.textContent).toMatch(/^\$ sh build.sh/)
    expect(container.querySelector('.line.end-good')?.textContent).toBe('[exit 0 · 0s]')
  })
})

describe('chat', () => {
  it('draws "@name HH:MM text" under its day, and says through note.act', async () => {
    const act = vi.fn(async () => {})
    const { container } = draw('chat.md', { act })
    const msgs = [...container.querySelectorAll('.msg')].map((e) => e.textContent?.replace(/\s+/g, ' ').trim())
    expect(msgs).toEqual(['@min 03:14 can you rerun the tests?', '@claude 03:14 41/41 pass'])
    expect(container.querySelector('.day')?.textContent).toBe('Today')
    const say = screen.getByLabelText('Say something')
    await fireEvent.input(say, { target: { value: 'thanks' } })
    await fireEvent.keyDown(say, { key: 'Enter' })
    // who is left to the bridge, which names the person as the terminal does
    expect(act).toHaveBeenCalledWith(sample('chat.md'), { name: 'say', args: { text: 'thanks' } })
  })
})

describe('script', () => {
  it('shows what it runs and the tail of its log, with how it ended', () => {
    const { container } = draw('build.sh')
    expect(container.querySelector('.code')?.textContent).toContain('echo building')
    const out = within(screen.getByRole('region', { name: 'Log' }))
    expect(out.getByText('building')).toBeTruthy()
    expect(container.querySelector('.out h4')?.textContent).toContain('[exit 0 · 0s]')
  })

  it('Run asks the window, which asks the person first', async () => {
    const manage = vi.fn()
    draw('build.sh', { manage })
    await fireEvent.click(screen.getByRole('button', { name: 'Run' }))
    expect(manage).toHaveBeenCalledWith(sample('build.sh'), 'run')
  })

  it('without a window to ask, there is no Run', () => {
    draw('build.sh')
    expect((screen.getByRole('button', { name: 'Run' }) as HTMLButtonElement).disabled).toBe(true)
  })
})

describe('api', () => {
  it('lists its requests and shows the picked one\'s response from its log', async () => {
    const { container } = draw('api.http')
    const reqs = [...container.querySelectorAll('.req')]
    expect(reqs.map((r) => r.querySelector('.title')?.textContent)).toEqual(['Hello', 'Missing'])
    expect(reqs[0].querySelector('.method')?.textContent).toBe('GET')
    expect(reqs[0].querySelector('.meta')?.textContent).toBe('1 ✓')
    const response = () => within(screen.getByRole('region', { name: 'Response' }))
    expect(response().getByText('[200 OK · 1ms · Hello · 1/1 ✔]').className).toContain('end-good')
    await fireEvent.click(reqs[1])
    expect(response().getByText(/^\[404 File not found/).className).toContain('end-bad')
    expect(response().getByText('✘ response.statusCode == 200 · got 404').className).toContain('bad')
  })

  it('Send asks the window to send the picked request, by its place and title', async () => {
    const manage = vi.fn()
    const { container } = draw('api.http', { manage })
    await fireEvent.click(container.querySelectorAll('.req')[1])
    await fireEvent.click(screen.getByRole('button', { name: 'Send' }))
    expect(manage).toHaveBeenCalledWith(sample('api.http'), 'send', { at: 2, title: 'Missing' })
  })

  it('a bridge that does not list requests yet shows the file', () => {
    const n = sample('api.http')
    render(Shape, { props: { note: { ...n, data: { source: '### x\nGET https://a.example\n' } } }, context: context() })
    expect(screen.getByRole('region', { name: n.title }).textContent).toContain('GET https://a.example')
  })
})

describe('html', () => {
  it('draws the page in a sandboxed frame under its badge', () => {
    const { container } = draw('report.html')
    expect(screen.getByText('HTML · view only')).toBeTruthy()
    const f = container.querySelector('iframe')!
    expect(f.getAttribute('sandbox')).toBe('allow-scripts')
    expect(f.getAttribute('srcdoc')).toContain('<h1>Benchmark</h1>')
    expect(f.getAttribute('srcdoc')).toMatch(/^<!doctype html><html><head><meta charset="utf-8"><meta http-equiv="Content-Security-Policy"/)
    expect(f.hasAttribute('src')).toBe(false)
  })

  it('takes its height only from its own frame', async () => {
    const { container } = draw('report.html')
    const f = container.querySelector('iframe')!
    const post = (source: unknown, origin: string, data: unknown) =>
      window.dispatchEvent(new MessageEvent('message', { source: source as Window, origin, data }))
    post(window, 'null', { type: 'stickypane:height', height: 999 })
    post(f.contentWindow, 'http://wails.localhost', { type: 'stickypane:height', height: 999 })
    await Promise.resolve()
    expect(f.style.height).toBe('80px')
    post(f.contentWindow, 'null', { type: 'stickypane:height', height: 321.2 })
    await vi.waitFor(() => expect(f.style.height).toBe('322px'))
  })
})
