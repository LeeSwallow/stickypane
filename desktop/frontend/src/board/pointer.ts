// What the board's drags (a grid note's edges, a canvas note's handles and
// tape) share about the pointer.

/** MOVED is how far, in px, a press goes before it is a drag and not a click. */
export const MOVED = 3

/**
 * capture keeps the pointer of e on the element it went down on, when the
 * webview lets it: WebKit refuses a pointer it does not count as active,
 * and a drag must go on without it, from the window's events (#66).
 */
export function capture(e: PointerEvent): void {
  try {
    ;(e.currentTarget as Element | null)?.setPointerCapture?.(e.pointerId)
  } catch {
    // the drag follows the window's pointer events instead
  }
}
