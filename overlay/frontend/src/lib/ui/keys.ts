// Keyboard movement shared by the composite controls (tabs, toolbars,
// segmented controls, menus): one tab stop, arrows move inside it.

export type Orientation = 'horizontal' | 'vertical'

const MODIFIERS: Record<string, string> = { alt: '⌥', option: '⌥', cmd: '⌘', command: '⌘', ctrl: '⌃', control: '⌃', shift: '⇧' }

/** hotkeyLabel writes a hotkey from overlay.json ("alt+space") as macOS does ("⌥Space"). */
export const hotkeyLabel = (hotkey: string): string =>
  hotkey
    .split('+')
    .filter(Boolean)
    .map((k) => MODIFIERS[k.toLowerCase()] ?? k.charAt(0).toUpperCase() + k.slice(1))
    .join('')

/**
 * nextIndex is where an arrow, Home or End key moves from index i among n
 * items, skipping disabled ones and wrapping; null when the key does not move.
 */
export function nextIndex(
  key: string,
  i: number,
  n: number,
  orientation: Orientation = 'horizontal',
  disabled: (i: number) => boolean = () => false,
): number | null {
  if (n <= 0) return null
  const fwd = orientation === 'horizontal' ? 'ArrowRight' : 'ArrowDown'
  const back = orientation === 'horizontal' ? 'ArrowLeft' : 'ArrowUp'
  let step: number
  let from: number
  if (key === fwd) [step, from] = [1, i]
  else if (key === back) [step, from] = [-1, i]
  else if (key === 'Home') [step, from] = [1, -1]
  else if (key === 'End') [step, from] = [-1, n]
  else return null
  for (let k = 1; k <= n; k++) {
    const j = (((from + step * k) % n) + n) % n
    if (!disabled(j)) return j
  }
  return null
}

const FOCUSABLE =
  'button:not([disabled]), [href], input:not([disabled]), select:not([disabled]), textarea:not([disabled]), [tabindex]:not([tabindex="-1"])'

/** focusables lists what Tab can reach inside an element, in order. */
export const focusables = (el: HTMLElement): HTMLElement[] =>
  [...el.querySelectorAll<HTMLElement>(FOCUSABLE)].filter((e) => !e.hasAttribute('inert') && e.offsetParent !== null)

/**
 * roving is a Svelte action for a toolbar: arrows move focus between the
 * controls inside it. A control that handles its own arrows (a segmented
 * control, a menu) calls preventDefault, and the toolbar leaves it be.
 */
export function roving(el: HTMLElement, orientation: Orientation = 'horizontal') {
  const onkey = (e: KeyboardEvent) => {
    if (e.defaultPrevented) return
    const items = [...el.querySelectorAll<HTMLElement>('button:not([disabled]):not([tabindex="-1"]), [tabindex="0"]')]
    const i = items.indexOf(document.activeElement as HTMLElement)
    if (i < 0) return
    const j = nextIndex(e.key, i, items.length, orientation)
    if (j === null) return
    e.preventDefault()
    items[j].focus()
  }
  el.addEventListener('keydown', onkey)
  return { destroy: () => el.removeEventListener('keydown', onkey) }
}
