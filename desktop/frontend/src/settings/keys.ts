// The hotkey recorder's rules (03 §3.12): a key pressed becomes an
// accelerator written as the shell writes it (internal/hotkey):
// modifiers in macOS's order, Ctrl, Option, Shift, Cmd, then the key.
// The key is read from where it is on the keyboard (KeyboardEvent.code),
// because Option changes the letter a key types on a Mac. Pure.

export interface KeyPress {
  key: string
  code: string
  metaKey: boolean
  ctrlKey: boolean
  altKey: boolean
  shiftKey: boolean
}

/** What a key press means to the recorder. */
export type Recorded =
  | { kind: 'cancel' }
  | { kind: 'held'; mods: string[] }
  | { kind: 'done'; accel: string }

const MOD_CODES = new Set(['MetaLeft', 'MetaRight', 'ControlLeft', 'ControlRight', 'AltLeft', 'AltRight', 'ShiftLeft', 'ShiftRight', 'CapsLock', 'Fn'])

const NAMED: Record<string, string> = {
  Space: 'Space',
  Enter: 'Return',
  NumpadEnter: 'Return',
  Tab: 'Tab',
  Backspace: 'Backspace',
  Delete: 'Delete',
  Escape: 'Escape',
  ArrowLeft: 'Left',
  ArrowRight: 'Right',
  ArrowUp: 'Up',
  ArrowDown: 'Down',
  Home: 'Home',
  End: 'End',
  PageUp: 'Page Up',
  PageDown: 'Page Down',
  Minus: '-',
  Equal: '=',
  BracketLeft: '[',
  BracketRight: ']',
  Backslash: '\\',
  Semicolon: ';',
  Quote: "'",
  Comma: ',',
  Period: '.',
  Slash: '/',
  Backquote: '`',
}

/** keyOf is the key a code names, as the shell writes it; "" when it is not one a hotkey can use. */
export function keyOf(code: string): string {
  if (/^Key[A-Z]$/.test(code)) return code.slice(3)
  if (/^Digit[0-9]$/.test(code)) return code.slice(5)
  if (/^F([1-9]|1[0-9]|20)$/.test(code)) return code
  return NAMED[code] ?? ''
}

/** modsOf are the modifiers held, in macOS's order. */
export function modsOf(e: Pick<KeyPress, 'metaKey' | 'ctrlKey' | 'altKey' | 'shiftKey'>): string[] {
  const out: string[] = []
  if (e.ctrlKey) out.push('Ctrl')
  if (e.altKey) out.push('Option')
  if (e.shiftKey) out.push('Shift')
  if (e.metaKey) out.push('Cmd')
  return out
}

/**
 * record reads one key press: Esc alone cancels, a modifier alone is
 * held (shown while it is down), any other key is the accelerator.
 */
export function record(e: KeyPress): Recorded {
  const mods = modsOf(e)
  if (e.code === 'Escape' && mods.length === 0) return { kind: 'cancel' }
  if (MOD_CODES.has(e.code) || !e.code) return { kind: 'held', mods }
  const k = keyOf(e.code)
  if (!k) return { kind: 'held', mods }
  return { kind: 'done', accel: [...mods, k].join('+') }
}

const GLYPH: Record<string, string> = { Ctrl: '⌃', Option: '⌥', Shift: '⇧', Cmd: '⌘' }

/** glyphs writes an accelerator as macOS shows it: "Option+Space" is ["⌥", "Space"]. */
export function glyphs(accel: string): string[] {
  if (!accel) return []
  const parts = accel.split('+')
  // "Cmd++" ends with the plus key
  if (accel.endsWith('++')) parts.splice(parts.length - 2, 2, '+')
  return parts.filter(Boolean).map((p) => GLYPH[p] ?? (p.toLowerCase() === 'alt' ? '⌥' : p.length === 1 ? p.toUpperCase() : p))
}
