// Following the end of a log (10 §10.1): the note's scroller stays at the
// bottom as lines come in, until the person scrolls up; back at the
// bottom, it follows again. The scroller is the nearest ancestor that
// scrolls (the note's body when it has a height, the zoomed view), so a
// shape needs no scroller of its own.

/** scroller is the nearest ancestor of el that scrolls vertically, or null. */
export function scroller(el: HTMLElement): HTMLElement | null {
  for (let p = el.parentElement; p; p = p.parentElement) {
    const o = getComputedStyle(p).overflowY
    if (o === 'auto' || o === 'scroll') return p
  }
  return null
}

/** atEnd says whether a scroller shows its last line, give or take a few pixels. */
export const atEnd = (s: { scrollTop: number; clientHeight: number; scrollHeight: number }, slack = 4): boolean =>
  s.scrollHeight - s.scrollTop - s.clientHeight <= slack

/**
 * follow is a Svelte action: whenever `count` (the number of lines)
 * changes, the scroller goes to its end if it was there before.
 */
export function follow(el: HTMLElement, count: number) {
  let s: HTMLElement | null = null
  let pinned = true
  const onScroll = () => {
    if (s) pinned = atEnd(s)
  }
  const attach = () => {
    const next = scroller(el)
    if (next === s) return
    s?.removeEventListener('scroll', onScroll)
    s = next
    s?.addEventListener('scroll', onScroll, { passive: true })
  }
  const toEnd = () => {
    attach()
    if (s && pinned) s.scrollTop = s.scrollHeight
  }
  queueMicrotask(toEnd)
  return {
    update(n: number) {
      if (n !== count) {
        count = n
        requestAnimationFrame(toEnd)
      }
    },
    destroy() {
      s?.removeEventListener('scroll', onScroll)
    },
  }
}
