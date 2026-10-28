import { describe, expect, test } from 'vitest'
import { canGoBack, flip, followTab, goBack, initView, isChanged, navigate, pageOf, seeAll, selectTab, zoomIn } from './view'
import { keyAction } from './keys'

describe('zoom and back', () => {
  test('zoom fills the window and Esc returns', () => {
    let v = initView('')
    expect(canGoBack(v)).toBe(false)
    v = zoomIn(v, 'plan.md')
    expect(v.zoom).toBe('plan.md')
    expect(canGoBack(v)).toBe(true)
    const esc = keyAction({ key: 'Escape' }, { zoomed: true, canBack: canGoBack(v) })
    expect(esc).toEqual({ type: 'back' })
    v = goBack(v)
    expect(v.zoom).toBeNull()
    expect(canGoBack(v)).toBe(false)
    expect(goBack(v)).toBe(v) // nothing to go back to: unchanged
  })
  test('a link opens its note zoomed on its tab, and back walks the way home', () => {
    let v = zoomIn(initView(''), 'a.md')
    v = navigate(v, 'work', 'work/b.md')
    expect([v.tab, v.zoom]).toEqual(['work', 'work/b.md'])
    v = navigate(v, '', 'c.md')
    v = goBack(v)
    expect([v.tab, v.zoom]).toEqual(['work', 'work/b.md'])
    v = goBack(v)
    expect([v.tab, v.zoom]).toEqual(['', 'a.md'])
    v = goBack(v)
    expect([v.tab, v.zoom]).toEqual(['', null])
  })
  test('choosing a tab leaves the zoom', () => {
    const v = selectTab(zoomIn(initView(''), 'a.md'), 'work')
    expect([v.tab, v.zoom, canGoBack(v)]).toEqual(['work', null, false])
  })
  test('the terminal moving its tab moves the window; the window never writes it', () => {
    let v = initView('')
    v = selectTab(v, 'notes')
    expect(followTab(v, '', '').tab).toBe('notes') // unchanged setting: the window keeps its own tab
    expect(followTab(v, '', 'work').tab).toBe('work')
  })
})

describe('book pages', () => {
  test('book pages flip with [ and ]', () => {
    let v = initView('')
    expect(pageOf(v, 'book', 3)).toBe(0)
    const next = keyAction({ key: ']' }, { zoomed: false, canBack: false })
    const prev = keyAction({ key: '[' }, { zoomed: false, canBack: false })
    expect(next).toEqual({ type: 'page', delta: 1 })
    expect(prev).toEqual({ type: 'page', delta: -1 })
    v = flip(v, 'book', 1, 3)
    v = flip(v, 'book', 1, 3)
    expect(pageOf(v, 'book', 3)).toBe(2)
    v = flip(v, 'book', 1, 3) // the last page stays
    expect(pageOf(v, 'book', 3)).toBe(2)
    v = flip(v, 'book', -1, 3)
    expect(pageOf(v, 'book', 3)).toBe(1)
    expect(pageOf(v, 'book', 1)).toBe(0) // a book that lost pages clamps
  })
})

describe('keys', () => {
  const ctx = { zoomed: false, canBack: false }
  test('the terminal keys', () => {
    expect(keyAction({ key: 'Enter' }, ctx)).toEqual({ type: 'zoom' })
    expect(keyAction({ key: '-' }, ctx)).toEqual({ type: 'size', dir: -1 })
    expect(keyAction({ key: '=' }, ctx)).toEqual({ type: 'size', dir: 1 })
    expect(keyAction({ key: '+' }, ctx)).toEqual({ type: 'size', dir: 1 })
    expect(keyAction({ key: '2', metaKey: true }, ctx)).toEqual({ type: 'tab', index: 1 })
    expect(keyAction({ key: 'ArrowRight' }, ctx)).toEqual({ type: 'focus', delta: 1 })
    expect(keyAction({ key: '{' }, ctx)).toEqual({ type: 'order', delta: -1 })
    expect(keyAction({ key: 'e' }, ctx)).toEqual({ type: 'edit' })
    expect(keyAction({ key: 'Escape' }, ctx)).toBeNull() // nothing to go back to
    expect(keyAction({ key: 'Backspace', metaKey: false }, { zoomed: true, canBack: true })).toBeNull()
  })
  test('typing in a field is not a key for the board', () => {
    expect(keyAction({ key: '-', editing: true }, ctx)).toBeNull()
    expect(keyAction({ key: 'Escape', editing: true }, { zoomed: true, canBack: true })).toBeNull()
  })
})

describe('the changed marker', () => {
  test('a note is changed when its file moved on since the window saw it', () => {
    const a = { name: 'a', modTime: '2026-10-28T10:00:00Z' }
    let seen = seeAll({}, [a])
    expect(isChanged(seen, a)).toBe(false)
    const a2 = { name: 'a', modTime: '2026-10-28T10:05:00Z' }
    expect(isChanged(seen, a2)).toBe(true)
    seen = seeAll(seen, [a2]) // a later load does not clear it
    expect(isChanged(seen, a2)).toBe(true)
    seen = { ...seen, a: a2.modTime } // looked at
    expect(isChanged(seen, a2)).toBe(false)
    const b = { name: 'b', modTime: '2026-10-28T11:00:00Z' }
    expect(isChanged(seeAll(seen, [b]), b)).toBe(false) // a new note is new, not changed
  })
})
