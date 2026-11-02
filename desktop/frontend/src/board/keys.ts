// The board's keys, as the terminal's (03 §3.10, 10 §10.1–10.2).

export type KeyAction =
  | { type: 'back' }
  | { type: 'zoom' }
  | { type: 'size'; dir: -1 | 1 }
  | { type: 'page'; delta: -1 | 1 }
  | { type: 'tab'; index: number }
  | { type: 'focus'; delta: -1 | 1 }
  | { type: 'order'; delta: -1 | 1 }
  | { type: 'row'; delta: -1 | 1 }
  | { type: 'edit' }
  | { type: 'create' }
  | { type: 'settings' }
  | { type: 'archive' }
  | { type: 'trash' }
  | { type: 'undo' }
  | { type: 'tidy' }
  | { type: 'tearAll' }
  | { type: 'tearBoard' }

export interface KeyLike {
  key: string
  /** the physical key: with ⌥ held, key is the letter ⌥ makes */
  code?: string
  metaKey?: boolean
  ctrlKey?: boolean
  altKey?: boolean
  shiftKey?: boolean
  /** the key went to a text field: it is the field's */
  editing?: boolean
}

export interface KeyContext {
  zoomed: boolean
  canBack: boolean
  /** the note the keys act on is a book: left and right turn its pages */
  book?: boolean
}

/**
 * frameOf is the note whose frame holds el (data-frame), the note keys act
 * on. A link inside a note names another note (data-note) and is not it.
 */
export const frameOf = (el: Element | null): string | null =>
  (el?.closest('[data-frame]') as HTMLElement | null)?.dataset.frame ?? null

/** keyAction is what a key does on the board, or null when it does nothing here. */
export function keyAction(e: KeyLike, ctx: KeyContext): KeyAction | null {
  if (e.editing) return null
  if (e.altKey) {
    // ⌥⌘L tidies the canvas (03 §3.6.2); ⌥⌘T puts each note of the tab in its own window, ⇧⌥⌘T every tab's
    if (!e.metaKey || ctx.zoomed) return null
    if (e.code === 'KeyL') return { type: 'tidy' }
    if (e.code === 'KeyT') return { type: e.shiftKey ? 'tearBoard' : 'tearAll' }
    return null
  }
  if (e.metaKey || e.ctrlKey) {
    if (e.key === ',') return { type: 'settings' }
    if (e.key === 'z' && !e.shiftKey) return { type: 'undo' }
    return /^[1-9]$/.test(e.key) ? { type: 'tab', index: Number(e.key) - 1 } : null
  }
  switch (e.key) {
    case 'Escape':
      return ctx.canBack ? { type: 'back' } : null
    case 'Enter':
      return { type: 'zoom' }
    case '-':
      return { type: 'size', dir: -1 }
    case '=':
    case '+':
      return { type: 'size', dir: 1 }
    case '[':
      return { type: 'page', delta: -1 }
    case ']':
      return { type: 'page', delta: 1 }
    case 'ArrowLeft':
      return ctx.book ? { type: 'page', delta: -1 } : { type: 'focus', delta: -1 }
    case 'ArrowRight':
      return ctx.book ? { type: 'page', delta: 1 } : { type: 'focus', delta: 1 }
    case 'ArrowUp':
      return { type: 'focus', delta: -1 }
    case 'ArrowDown':
      return { type: 'focus', delta: 1 }
    // H L (and { }) move the note a place, K J a row (the terminal's keys)
    case '{':
    case 'H':
      return { type: 'order', delta: -1 }
    case '}':
    case 'L':
      return { type: 'order', delta: 1 }
    case 'K':
      return { type: 'row', delta: -1 }
    case 'J':
      return { type: 'row', delta: 1 }
    case 'e':
    case 'E':
      return { type: 'edit' }
    case 'a':
    case 'A':
      return { type: 'create' }
    case 'x':
      return { type: 'archive' }
    case 'D':
      return { type: 'trash' }
    case 'u':
      return { type: 'undo' }
  }
  return null
}
