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

/**
 * follow is a drag of the pointer e went down with: move hears each move
 * of it, and end hears it let go (keep) or taken away by the system (not
 * keep), once, after which nothing more is heard. It listens on the
 * window, not the element: a webview may refuse or drop the capture, and
 * the pointer leaves a thin edge or a handle as soon as it moves (#66).
 */
export function follow(
  e: PointerEvent,
  move: (m: PointerEvent) => void,
  end: (keep: boolean) => void,
  on: Pick<EventTarget, 'addEventListener' | 'removeEventListener'> = window,
): void {
  capture(e)
  const mine = (f: (m: PointerEvent) => void) => (m: Event) => (m as PointerEvent).pointerId === e.pointerId && f(m as PointerEvent)
  const moved = mine(move)
  const stop = (keep: boolean) =>
    mine(() => {
      on.removeEventListener('pointermove', moved)
      on.removeEventListener('pointerup', up)
      on.removeEventListener('pointercancel', cancel)
      end(keep)
    })
  const up = stop(true)
  const cancel = stop(false)
  on.addEventListener('pointermove', moved)
  on.addEventListener('pointerup', up)
  on.addEventListener('pointercancel', cancel)
}
