// @vitest-environment jsdom
// The scenes of docs/desktop/01 §1.7 on the app's side, over what a real
// bridge sent while an agent worked from another process
// (desktop/internal/bridgeclient/scenes_test.go; `go test ./internal/bridgeclient -run
// Scene -update` writes fixtures/). What the window draws and sends from
// those payloads is checked here; the Go test checks the files and the
// agent's side.
import { cleanup, fireEvent, render, screen, within } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { boardContext, type BoardCtx } from '../board/context'
import { glance } from '../board/glance'
import { findNote, type Board, type Note } from '../board/model'
import { setLanguage } from '../i18n'
import { asChange, planChange } from '../lib/motion/plan'
import { CSP, SANDBOX } from '../shapes/html'
import Shape from '../shapes/Shape.svelte'
import a from './fixtures/scene-a.json'
import b from './fixtures/scene-b.json'
import c from './fixtures/scene-c.json'

globalThis.ResizeObserver ??= class {
  observe() {}
  unobserve() {}
  disconnect() {}
} as unknown as typeof ResizeObserver

beforeEach(() => setLanguage('en'))
afterEach(cleanup)

const board = (x: unknown) => x as Board
const note = (x: unknown, name: string): Note => {
  const n = findNote(board(x).tabs, name)
  if (!n) throw new Error(`no ${name}`)
  return n
}
const latest = (x: unknown) => new Date(Math.max(...board(x).tabs.flatMap((t) => t.notes.map((n) => Date.parse(n.modTime)))) + 60_000)

function ctx(now: Date, over: Partial<BoardCtx> = {}) {
  const c: BoardCtx = {
    find: () => null, navigate: () => {}, act: async () => {}, arrange: async () => {}, zoom: () => {}, back: () => {},
    canBack: () => false, flip: () => {}, pageOf: () => 0, changed: () => false, seen: () => {}, filter: () => null,
    now: () => now, ...over,
  }
  return new Map<unknown, unknown>([[boardContext, c]])
}

describe('scene A: three agents on one board', () => {
  const before = board(a.before)
  const after = board(a.after)

  it('the glance line names each agent, and the agent of the card that has not moved is stalled', () => {
    const tags = glance(before.tabs, latest(a.before))
    expect(tags.map((t) => t.who).sort()).toEqual(['claude-1', 'codex-1', 'codex-2'])
    expect(tags.find((t) => t.who === 'codex-2')).toMatchObject({ stalled: true })
    expect(tags.find((t) => t.who === 'codex-1')).toMatchObject({ stalled: false })
  })

  it("an agent's move from the terminal carries the card, in that agent's colour", () => {
    const plan = planChange(before, after, asChange(a.changed))
    const carry = plan.steps.find((s) => s.effect === 'carry')
    expect(carry?.target).toMatchObject({ part: 'card', card: 'payments' })
    expect(carry?.who).toBe('codex-1')
  })

  it('after the move the card is drawn in Review', () => {
    render(Shape, { props: { note: note(a.after, 'work.md') }, context: ctx(latest(a.after)) })
    const review = screen.getByText('Review').closest('section, [role="group"], .column, li, div') as HTMLElement
    expect(within(review.parentElement as HTMLElement).getByText('payments')).toBeTruthy()
  })
})

describe('scene B: the agent asks, the user answers', () => {
  it('the form says it waits for an answer, and the clicks send choose and submit', async () => {
    const act = vi.fn(async () => {})
    render(Shape, { props: { note: note(b.waiting, 'ship.md') }, context: ctx(latest(b.waiting), { act }) })
    expect(screen.getByRole('status').textContent).toMatch(/waiting/i)
    await fireEvent.click(screen.getByText('Homebrew too'))
    await fireEvent.click(screen.getByRole('button', { name: 'Ship' }))
    const sent = act.mock.calls.map((call) => (call as unknown[])[1])
    expect(sent[0]).toMatchObject({ name: 'choose' })
    expect(sent.at(-1)).toEqual({ name: 'submit', args: { label: 'Ship' } })
  })

  it('once answered it says which button, and the agent waiting got the same answer', () => {
    render(Shape, { props: { note: note(b.answered, 'ship.md') }, context: ctx(latest(b.answered)) })
    expect(screen.getByRole('status').textContent).toContain('Ship')
    expect(b.wait).toMatchObject({ submitted: true, button: 'Ship', answers: [{ values: ['Homebrew too'] }] })
  })
})

describe('scene C: a report as an html file', () => {
  it('arrives as an html note and is drawn in a sandboxed frame that loads nothing', () => {
    const n = note(c.after, '03-log/bench.html')
    expect(n.kind).toBe('html')
    const { container } = render(Shape, { props: { note: n }, context: ctx(latest(c.after)) })
    const frame = container.querySelector('iframe') as HTMLIFrameElement
    expect(frame.getAttribute('sandbox')).toBe(SANDBOX)
    expect(frame.getAttribute('sandbox')).not.toContain('allow-same-origin')
    const doc = frame.getAttribute('srcdoc') ?? ''
    expect(doc).toContain('<h1>Bench</h1>')
    expect(doc.indexOf(CSP)).toBeGreaterThanOrEqual(0)
    expect(doc.indexOf(CSP)).toBeLessThan(doc.indexOf('<h1>Bench</h1>'))
  })
})
