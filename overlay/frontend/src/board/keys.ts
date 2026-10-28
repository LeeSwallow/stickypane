// The board's keys, as the terminal's (03 §3.10, 10 §10.1–10.2).

export type KeyAction =
  | { type: 'back' }
  | { type: 'zoom' }
  | { type: 'size'; dir: -1 | 1 }
  | { type: 'page'; delta: -1 | 1 }
  | { type: 'tab'; index: number }
  | { type: 'focus'; delta: -1 | 1 }
  | { type: 'order'; delta: -1 | 1 }
  | { type: 'edit' }

export interface KeyLike {
  key: string
  metaKey?: boolean
  ctrlKey?: boolean
  altKey?: boolean
  /** the key went to a text field: it is the field's */
  editing?: boolean
}

export interface KeyContext {
  zoomed: boolean
  canBack: boolean
}

/** keyAction is what a key does on the board, or null when it does nothing here. */
export function keyAction(e: KeyLike, ctx: KeyContext): KeyAction | null {
  if (e.editing || e.altKey) return null
  if (e.metaKey || e.ctrlKey) {
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
    case 'ArrowUp':
      return { type: 'focus', delta: -1 }
    case 'ArrowRight':
    case 'ArrowDown':
      return { type: 'focus', delta: 1 }
    case '{':
      return { type: 'order', delta: -1 }
    case '}':
      return { type: 'order', delta: 1 }
    case 'e':
    case 'E':
      return { type: 'edit' }
  }
  return null
}
