import { describe, expect, it } from 'vitest'
import { formatKeys, keyName, parseKeys, showKey } from './keys'

describe('the board keys, written', () => {
  it('shows a sequence as the board does', () => {
    expect(showKey('g g')).toBe('gg')
    expect(showKey('ctrl+w j')).toBe('<ctrl+w>j')
    expect(showKey('ctrl+d')).toBe('ctrl+d')
    expect(showKey('?')).toBe('?')
  })
  it('reads commas between keys and spaces inside a sequence', () => {
    expect(parseKeys('g g, ctrl+d')).toEqual(['g g', 'ctrl+d'])
    expect(parseKeys('  X ,N,  d x ')).toEqual(['X', 'N', 'd x'])
    expect(parseKeys('')).toEqual([])
    expect(parseKeys(' , ')).toEqual([','])
  })
  it('keeps the comma key', () => {
    expect(parseKeys(',')).toEqual([','])
    expect(parseKeys(', <')).toEqual([',', '<'])
    expect(parseKeys(',, <')).toEqual([',', '<'])
    expect(parseKeys('ctrl+,, x')).toEqual(['ctrl+,', 'x'])
    expect(parseKeys('ctrl+,')).toEqual(['ctrl+,'])
  })
  it('reads back what it writes', () => {
    for (const keys of [['g g', 'ctrl+d'], [','], [',', '<'], ['>', '+'], ['ctrl+,', 'x']]) {
      expect(parseKeys(formatKeys(keys))).toEqual(keys)
    }
  })
})

const ev = (key: string, over: Partial<KeyboardEvent> = {}) =>
  ({ key, code: '', ctrlKey: false, altKey: false, shiftKey: false, metaKey: false, isComposing: false, ...over }) as KeyboardEvent

describe('a key pressed', () => {
  it('is named as the terminal names it', () => {
    expect(keyName(ev('x'))).toBe('x')
    expect(keyName(ev('X', { shiftKey: true }))).toBe('X')
    expect(keyName(ev('?', { shiftKey: true }))).toBe('?')
    expect(keyName(ev('d', { ctrlKey: true }))).toBe('ctrl+d')
    expect(keyName(ev('D', { ctrlKey: true, shiftKey: true }))).toBe('ctrl+shift+d')
    expect(keyName(ev('Tab', { shiftKey: true }))).toBe('shift+tab')
    expect(keyName(ev('PageDown'))).toBe('pgdown')
    expect(keyName(ev('ArrowUp', { shiftKey: true }))).toBe('shift+up')
    expect(keyName(ev(' '))).toBe('space')
    expect(keyName(ev('Escape'))).toBe('esc')
    expect(keyName(ev('F5'))).toBe('f5')
  })
  it('is the Latin key under a Korean jamo, and ⌥ does not change the letter', () => {
    expect(keyName(ev('ㅓ', { code: 'KeyJ' }))).toBe('j')
    expect(keyName(ev('Process', { code: 'KeyX', isComposing: true }))).toBe('x')
    expect(keyName(ev('∆', { code: 'KeyJ', altKey: true }))).toBe('alt+j')
  })
  it('is nothing for a modifier alone, or with ⌘', () => {
    expect(keyName(ev('Shift', { shiftKey: true }))).toBe('')
    expect(keyName(ev('k', { metaKey: true }))).toBe('')
  })
})
