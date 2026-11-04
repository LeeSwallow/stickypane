// The terminal board's keys, for the settings' 단축키: the app's own
// stickypane lists them (`stickypane keys --json`) and changes keys.json
// (`keys set`, `keys reset`), through the shell's SetupService
// (desktop/internal/shell/keys.go). The board checks every change: a key
// it cannot read, or one another action has, comes back as the error.

import { Call } from '@wailsio/runtime'
import { setup } from '../lib/shell'

/** One action of the terminal board. defaults and group are left out by a stickypane older than them. */
export interface Binding {
  name: string
  keys: string[]
  words: string
  changed: boolean
  defaults?: string[]
  group?: string
}

/** What the board's keys are; missing: no stickypane to ask. */
export interface Keys {
  bindings: Binding[]
  problems?: string[]
  missing?: boolean
}

export interface KeysApi {
  /** every action, its words in lang */
  keys(lang: string): Promise<Keys>
  /** gives action keys; none leaves it without one */
  setKeys(lang: string, action: string, keys: string[]): Promise<Keys>
  /** takes action's change back; '' takes every change back */
  resetKeys(lang: string, action: string): Promise<Keys>
}

const call = <T>(method: string, ...args: unknown[]) => Call.ByName(setup(method), ...args) as Promise<T>

export const keysService: KeysApi = {
  keys: (lang) => call('Keys', lang),
  setKeys: (lang, action, keys) => call('SetKeys', lang, action, keys),
  resetKeys: (lang, action) => call('ResetKeys', lang, action),
}

/** The groups, in the order the list shows them; an action of none goes last. */
export const GROUPS = ['move', 'notes', 'arrange', 'size', 'scroll', 'tabs', 'board'] as const

/**
 * showKey is a key as the board shows it: a sequence's characters side by
 * side ("g g" is gg), a named key in a sequence in angle brackets
 * ("ctrl+w j" is <ctrl+w>j). A key of its own is as it is.
 */
export function showKey(key: string): string {
  const parts = key.trim().split(/\s+/)
  if (parts.length < 2) return key
  return parts.map((p) => ([...p].length === 1 ? p : `<${p}>`)).join('')
}

/** formatKeys writes keys for the editor: commas between keys, spaces inside a sequence. */
export const formatKeys = (keys: string[]): string => keys.join(', ')

/**
 * parseKeys reads what the editor holds: keys apart by commas, the keys
 * of a sequence by spaces ("g g, ctrl+d" is two keys). A comma that is
 * the whole key, or ends a key with +, is the comma key: ", <" is the
 * keys , and <; "ctrl+," is one key.
 */
export function parseKeys(text: string): string[] {
  const out: string[] = []
  let seq: string[] = []
  for (const word of text.trim().split(/\s+/).filter(Boolean)) {
    let cur = ''
    for (const ch of word) {
      if (ch !== ',') cur += ch
      else if (cur.endsWith('+') || (cur === '' && seq.length === 0)) cur += ','
      else {
        if (cur) seq.push(cur)
        if (seq.length) out.push(seq.join(' '))
        cur = ''
        seq = []
      }
    }
    if (cur === ',' && !seq.length) {
      // the comma key is a key of its own: ", <" is , and <
      out.push(cur)
      continue
    }
    if (cur) seq.push(cur)
  }
  if (seq.length) out.push(seq.join(' '))
  return out
}

/**
 * keyName is a key pressed in the DOM as the terminal names it, or '' for
 * one that is only a modifier, or that the terminal does not see (⌘).
 * A key of the Korean 2-set layout is the Latin key under it.
 */
export function keyName(e: Pick<KeyboardEvent, 'key' | 'code' | 'ctrlKey' | 'altKey' | 'shiftKey' | 'metaKey' | 'isComposing'>): string {
  if (e.metaKey) return ''
  let k = e.key
  if (e.isComposing || k === 'Process' || /^[ㄱ-ㆎ]$/.test(k)) {
    const m = /^Key([A-Z])$/.exec(e.code)
    if (!m) return ''
    k = e.shiftKey ? m[1] : m[1].toLowerCase()
  }
  if (e.altKey && /^Key[A-Z]$/.test(e.code)) k = e.shiftKey ? e.code.slice(3) : e.code.slice(3).toLowerCase() // ⌥ makes another character on a Mac
  const named: Record<string, string> = {
    Enter: 'enter', Tab: 'tab', Backspace: 'backspace', Escape: 'esc', ' ': 'space',
    ArrowUp: 'up', ArrowDown: 'down', ArrowLeft: 'left', ArrowRight: 'right',
    Home: 'home', End: 'end', PageUp: 'pgup', PageDown: 'pgdown', Insert: 'insert', Delete: 'delete',
  }
  if (['Shift', 'Control', 'Alt', 'Meta', 'CapsLock', 'Dead', 'Unidentified'].includes(k)) return ''
  let base = named[k] ?? (/^F\d{1,2}$/.test(k) ? k.toLowerCase() : [...k].length === 1 ? k : '')
  if (!base) return ''
  const mods: string[] = []
  if (e.ctrlKey) mods.push('ctrl')
  if (e.altKey) mods.push('alt')
  const char = [...base].length === 1
  if (e.shiftKey && !char) mods.push('shift')
  if (char && mods.length && base !== base.toLowerCase()) {
    // ctrl+X is ctrl+shift+x, as the terminal says it
    base = base.toLowerCase()
    mods.push('shift')
  }
  return [...mods, base].join('+')
}
