import { describe, expect, it } from 'vitest'
import { glyphs, keyOf, record, type KeyPress } from './keys'

const press = (code: string, mods: Partial<KeyPress> = {}): KeyPress => ({ key: '', code, metaKey: false, ctrlKey: false, altKey: false, shiftKey: false, ...mods })

describe('the hotkey recorder', () => {
  it('writes modifiers in macOS order, then the key, as the shell does', () => {
    expect(record(press('Space', { altKey: true }))).toEqual({ kind: 'done', accel: 'Option+Space' })
    expect(record(press('KeyK', { metaKey: true, shiftKey: true }))).toEqual({ kind: 'done', accel: 'Shift+Cmd+K' })
    expect(record(press('KeyN', { ctrlKey: true, altKey: true }))).toEqual({ kind: 'done', accel: 'Ctrl+Option+N' })
  })
  it('reads where the key is, not what Option makes it type', () => {
    expect(record({ ...press('KeyP', { altKey: true }), key: 'π' })).toEqual({ kind: 'done', accel: 'Option+P' })
  })
  it('Esc alone cancels; with a modifier it is a key', () => {
    expect(record(press('Escape'))).toEqual({ kind: 'cancel' })
    expect(record(press('Escape', { metaKey: true, altKey: true }))).toEqual({ kind: 'done', accel: 'Option+Cmd+Escape' })
  })
  it('a modifier alone is held, not a hotkey', () => {
    expect(record(press('AltLeft', { altKey: true }))).toEqual({ kind: 'held', mods: ['Option'] })
    expect(record(press('IntlRo', { metaKey: true }))).toEqual({ kind: 'held', mods: ['Cmd'] })
  })
  it('knows letters, digits, F-keys, arrows and punctuation', () => {
    expect(keyOf('KeyA')).toBe('A')
    expect(keyOf('Digit7')).toBe('7')
    expect(keyOf('F13')).toBe('F13')
    expect(keyOf('F21')).toBe('')
    expect(keyOf('ArrowUp')).toBe('Up')
    expect(keyOf('Enter')).toBe('Return')
    expect(keyOf('Equal')).toBe('=')
  })
  it('shows an accelerator as glyphs', () => {
    expect(glyphs('Option+Space')).toEqual(['⌥', 'Space'])
    expect(glyphs('Shift+Cmd+K')).toEqual(['⇧', '⌘', 'K'])
    expect(glyphs('alt+space')).toEqual(['⌥', 'space'])
    expect(glyphs('')).toEqual([])
  })
})
