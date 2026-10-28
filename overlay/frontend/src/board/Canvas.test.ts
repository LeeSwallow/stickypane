// @vitest-environment jsdom
import { cleanup, fireEvent, render } from '@testing-library/svelte'
import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { setLanguage } from '../i18n'
import { look } from '../lib/doodle/look.svelte'
import { THEMES } from '../demo/themes'
import type { Place } from '../services/canvas'
import Canvas from './Canvas.svelte'
import { boardContext, type BoardCtx } from './context'
import type { Note } from './model'

look.palette = THEMES[0]

const base = { tab: '', icon: '', open: true, rows: 0, color: 'yellow', pinned: false, version: 'v1', summary: '', modTime: '2026-10-28T10:00:00Z', linked: false, size: '1/2' }
const note = (name: string): Note => ({ ...base, name, title: name, kind: 'note', data: { markdown: 'hi' } }) as Note
const at = (x: number, y: number, z: number): Place => ({ x, y, w: 320, h: 200, tilt: 1, z })

function ctx(over: Partial<BoardCtx> = {}) {
  const c: BoardCtx = {
    find: () => null, navigate: () => {}, act: async () => {}, arrange: async () => {}, zoom: () => {}, back: () => {},
    canBack: () => false, flip: () => {}, pageOf: () => 0, changed: () => false, seen: () => {}, filter: () => null,
    now: () => new Date('2026-10-28T12:00:00Z'), ...over,
  }
  return new Map<unknown, unknown>([[boardContext, c]])
}

beforeEach(() => {
  vi.stubGlobal('ResizeObserver', class { observe() {} unobserve() {} disconnect() {} })
  setLanguage('en')
})
afterEach(cleanup)

const item = (name: string) => document.querySelector<HTMLElement>(`[data-place="${name}"]`)!

describe('the canvas', () => {
  it('draws each note where it was left, leaning, in its stack order', () => {
    render(Canvas, { props: { notes: [note('a.md'), note('b.md')], places: { 'a.md': at(40, 80, 2), 'b.md': at(400, 16, 1) }, onplace: vi.fn() }, context: ctx() })
    const a = item('a.md')
    expect(a.style.left).toBe('40px')
    expect(a.style.top).toBe('80px')
    expect(a.style.width).toBe('320px')
    expect(a.style.height).toBe('200px')
    expect(a.style.zIndex).toBe('2')
    expect(a.style.getPropertyValue('--tilt')).toBe('1deg')
    expect(item('b.md').style.left).toBe('400px')
    // a resize handle on the right, the bottom and the corner
    expect([...a.querySelectorAll('.handle')].map((h) => [...h.classList].filter((c) => !c.startsWith('svelte-') && c !== 'handle').join())).toEqual(['e', 's', 'se'])
    // the canvas sizes the note: no grid width steps
    expect(a.querySelector('[aria-label="Size"]')).toBeNull()
  })

  it('moves a note by its tape, on the grid, and keeps the place', async () => {
    const onplace = vi.fn()
    render(Canvas, { props: { notes: [note('a.md'), note('b.md')], places: { 'a.md': at(40, 80, 1), 'b.md': at(400, 16, 2) }, onplace }, context: ctx() })
    const grip = item('a.md').querySelector('[data-grip]')!
    await fireEvent.pointerDown(grip, { button: 0, clientX: 100, clientY: 100, pointerId: 1 })
    await fireEvent.pointerMove(item('a.md'), { clientX: 133, clientY: 90, pointerId: 1 })
    // it follows the pointer while it moves, on top
    expect(item('a.md').style.left).toBe('72px')
    expect(item('a.md').style.zIndex).toBe('3')
    await fireEvent.pointerUp(item('a.md'), { clientX: 133, clientY: 90, pointerId: 1 })
    expect(onplace).toHaveBeenLastCalledWith({ 'a.md': { x: 72, y: 72, w: 320, h: 200, tilt: 1, z: 3 } })
  })

  it('lands anywhere with snap off', async () => {
    const onplace = vi.fn()
    render(Canvas, { props: { notes: [note('a.md')], places: { 'a.md': at(40, 80, 1) }, snap: false, onplace }, context: ctx() })
    await fireEvent.pointerDown(item('a.md').querySelector('[data-grip]')!, { button: 0, clientX: 0, clientY: 0, pointerId: 1 })
    await fireEvent.pointerMove(item('a.md'), { clientX: 3, clientY: 5, pointerId: 1 })
    await fireEvent.pointerUp(item('a.md'), { pointerId: 1 })
    expect(onplace).toHaveBeenLastCalledWith({ 'a.md': expect.objectContaining({ x: 43, y: 85 }) })
  })

  it('resizes a note by its corner, and a cancelled drag keeps only the raise', async () => {
    const onplace = vi.fn()
    render(Canvas, { props: { notes: [note('a.md'), note('b.md')], places: { 'a.md': at(0, 0, 1), 'b.md': at(0, 0, 5) }, onplace }, context: ctx() })
    const corner = item('a.md').querySelector('.handle.se')!
    await fireEvent.pointerDown(corner, { button: 0, clientX: 0, clientY: 0, pointerId: 1 })
    await fireEvent.pointerMove(corner, { clientX: 80, clientY: 40, pointerId: 1 })
    await fireEvent.pointerUp(corner, { pointerId: 1 })
    expect(onplace).toHaveBeenLastCalledWith({ 'a.md': expect.objectContaining({ w: 400, h: 240, z: 6 }) })
    onplace.mockClear()
    await fireEvent.pointerDown(corner, { button: 0, clientX: 0, clientY: 0, pointerId: 2 })
    await fireEvent.pointerMove(corner, { clientX: 80, clientY: 40, pointerId: 2 })
    await fireEvent.pointerCancel(corner, { pointerId: 2 })
    expect(onplace).toHaveBeenLastCalledWith({ 'a.md': at(0, 0, 6) })
  })

  it('brings a touched note to the top', async () => {
    const onplace = vi.fn()
    render(Canvas, { props: { notes: [note('a.md'), note('b.md')], places: { 'a.md': at(0, 0, 1), 'b.md': at(0, 0, 4) }, onplace }, context: ctx() })
    await fireEvent.pointerDown(item('a.md').querySelector('.body')!, { button: 0, pointerId: 1 })
    expect(onplace).toHaveBeenCalledWith({ 'a.md': at(0, 0, 5) })
    onplace.mockClear()
    await fireEvent.pointerDown(item('b.md').querySelector('.body')!, { button: 0, pointerId: 1 })
    expect(onplace).not.toHaveBeenCalled()
  })

  it('zooms with ⌘ and the wheel, and only then', async () => {
    const r = render(Canvas, { props: { notes: [note('a.md')], places: { 'a.md': at(0, 0, 1) }, onplace: vi.fn() }, context: ctx() })
    const c = document.querySelector('[data-canvas]')!
    await fireEvent.wheel(c, { deltaY: -10 })
    expect(document.querySelector<HTMLElement>('.layer')!.style.transform).toBe('scale(1)')
    await fireEvent.wheel(c, { deltaY: -10, metaKey: true })
    expect(document.querySelector<HTMLElement>('.layer')!.style.transform).toBe('scale(1.1)')
    expect(r.container.textContent).toContain('110%')
  })
})
