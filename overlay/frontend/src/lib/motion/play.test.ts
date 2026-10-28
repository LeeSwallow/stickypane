// @vitest-environment jsdom
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import type { Look } from './keyframes'
import { keyOf, Player, snapshot } from './play'

const full: Look = { style: 'doodle', level: 'full' }

// jsdom lays nothing out and animates nothing: places come from data-y,
// and animate() is recorded.
type Call = { el: Element; keyframes: Keyframe[]; options: KeyframeAnimationOptions }
let calls: Call[] = []

function fakeAnimation(): Animation {
  const t = new EventTarget() as Animation
  ;(t as unknown as { finish: () => void }).finish = () => t.dispatchEvent(new Event('finish'))
  ;(t as unknown as { cancel: () => void }).cancel = () => t.dispatchEvent(new Event('cancel'))
  return t
}

beforeEach(() => {
  calls = []
  Element.prototype.animate = function (keyframes, options) {
    calls.push({ el: this, keyframes: keyframes as Keyframe[], options: options as KeyframeAnimationOptions })
    return fakeAnimation()
  }
  vi.spyOn(Element.prototype, 'getBoundingClientRect').mockImplementation(function (this: Element) {
    const y = Number((this as HTMLElement).dataset?.y ?? 0)
    return { left: 0, top: y, width: 100, height: 20, right: 100, bottom: y + 20, x: 0, y, toJSON() {} } as DOMRect
  })
})
afterEach(() => vi.restoreAllMocks())

function page(html: string): HTMLElement {
  const root = document.createElement('div')
  root.innerHTML = html
  document.body.replaceChildren(root)
  return root
}

describe('Player', () => {
  it('carries a moved card from its old place and slides the cards it pushed', () => {
    const root = page(`<div data-frame="work.md"><div data-card="ship" data-y="0"></div><div data-card="other" data-y="30"></div></div>`)
    const before = snapshot(root)
    // drawn again: ship went below, other moved up
    root.innerHTML = `<div data-frame="work.md"><div data-card="other" data-y="0"></div><div data-card="ship" data-y="80"></div></div>`
    new Player().play(root, before, { steps: [{ target: { part: 'card', frame: 'work.md', card: 'ship' }, effect: 'carry', delay: 0 }], flip: ['work.md'] }, { look: full })
    const ship = calls.find((c) => (c.el as HTMLElement).dataset.card === 'ship' && c.options.duration === 620)!
    expect(ship.keyframes[0].transform).toBe('translate(0px, -80px) scale(1)')
    const other = calls.find((c) => (c.el as HTMLElement).dataset.card === 'other')!
    expect(other.keyframes[0].transform).toBe('translate(0px, 30px)')
    // and the owner's highlight after it lands
    expect(root.querySelector('[data-card="ship"] .motion-glow')).not.toBeNull()
  })

  it('pushes in only the new last lines of a log', () => {
    const root = page(`<div data-frame="build.log"><pre class="log"><span class="line">a</span><span class="line">b</span><span class="line">c</span></pre></div>`)
    const before = snapshot(root)
    new Player().play(root, before, { steps: [{ target: { part: 'lines', frame: 'build.log', count: 2 }, effect: 'push', delay: 0 }], flip: [] }, { look: full })
    expect(calls.map((c) => c.el.textContent)).toEqual(['b', 'c'])
    expect(calls.map((c) => c.options.delay)).toEqual([0, 40])
  })

  it('shows a note that went as a copy that fades, then takes the copy away', () => {
    const root = page(`<div data-frame="gone.md" data-y="10">bye</div>`)
    const before = snapshot(root, new Set([keyOf({ part: 'note', frame: 'gone.md' })!]))
    root.innerHTML = ''
    const p = new Player()
    p.play(root, before, { steps: [{ target: { part: 'note', frame: 'gone.md' }, effect: 'leave', delay: 0 }], flip: [] }, { look: full })
    const ghost = document.body.querySelector<HTMLElement>('[aria-hidden="true"]')!
    expect(ghost.textContent).toBe('bye')
    expect(ghost.style.top).toBe('10px')
    p.finish()
    expect(ghost.isConnected).toBe(false)
  })

  it('a new change ends what is still moving and its glows', () => {
    const root = page(`<div data-frame="a.md"></div>`)
    const p = new Player()
    p.play(root, snapshot(root), { steps: [{ target: { part: 'note', frame: 'a.md' }, effect: 'glow', delay: 0 }], flip: [] }, { look: full })
    expect(root.querySelector('.motion-glow')).not.toBeNull()
    expect(p.busy).toBe(true)
    p.play(root, snapshot(root), { steps: [], flip: [] }, { look: full })
    expect(root.querySelector('.motion-glow')).toBeNull()
    expect(p.busy).toBe(false)
  })

  it('moves nothing when motion is off, and skips parts that are not there', () => {
    const root = page(`<div data-frame="a.md"></div>`)
    const p = new Player()
    p.play(root, snapshot(root), { steps: [{ target: { part: 'note', frame: 'a.md' }, effect: 'drop', delay: 0 }], flip: [] }, { look: { style: 'doodle', level: 'off' } })
    p.play(root, snapshot(root), { steps: [{ target: { part: 'card', frame: 'a.md', card: 'x' }, effect: 'drop', delay: 0 }], flip: [] }, { look: full })
    expect(calls).toEqual([])
  })
})
