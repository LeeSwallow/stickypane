// What the intro needs of the page around it: when the window first
// shows, whether the notes are there yet, what day the intro was last
// played, and an animation that can be skipped. Kept apart from intro.ts,
// which is pure, and from Intro.svelte, which only draws.

import { INTRO_KEY, wriggle, type Timed } from './intro'

/** shown resolves once the page is visible (at once when it already is). */
export function shown(doc: Document = document): Promise<void> {
  if (doc.visibilityState !== 'hidden') return Promise.resolve()
  return new Promise((resolve) => {
    const on = () => {
      if (doc.visibilityState === 'hidden') return
      doc.removeEventListener('visibilitychange', on)
      resolve()
    }
    doc.addEventListener('visibilitychange', on)
  })
}

/** frames resolves with the notes on the page once there are some, or [] after `wait` ms. */
export function frames(root: ParentNode, wait: number): Promise<HTMLElement[]> {
  const now = () => [...root.querySelectorAll<HTMLElement>('[data-frame]')]
  if (now().length || typeof MutationObserver === 'undefined') return Promise.resolve(now())
  return new Promise((resolve) => {
    const done = () => {
      clearTimeout(timer)
      obs.disconnect()
      resolve(now())
    }
    const obs = new MutationObserver(() => now().length && done())
    obs.observe(root as Node, { childList: true, subtree: true })
    const timer = setTimeout(done, wait)
  })
}

/** lastDay is the day the intro last played on this computer, or null. */
export function lastDay(store: Storage | null | undefined): string | null {
  try {
    return store?.getItem(INTRO_KEY) ?? null
  } catch {
    return null
  }
}

/** remember keeps today as the day the intro played; a store that refuses is no harm. */
export function remember(store: Storage | null | undefined, day: string): void {
  try {
    store?.setItem(INTRO_KEY, day)
  } catch {
    // private window, blocked storage: the intro plays again, that is all
  }
}

export const storage = (): Storage | null => {
  try {
    return typeof localStorage === 'undefined' ? null : localStorage
  } catch {
    return null
  }
}

/** Reel is a set of animations that a key or a click ends at once. */
export class Reel {
  private all: Animation[] = []
  run(el: Element, t: Timed | null): Animation | null {
    if (!t || typeof (el as HTMLElement).animate !== 'function') return null
    const a = el.animate(t.keyframes, t.timing)
    this.all.push(a)
    return a
  }
  /** wriggle has each note wriggle once, from `at`, in the order drawn. */
  wriggle(notes: HTMLElement[], at: number, neat: boolean) {
    notes.forEach((n, i) => this.run(n, wriggle(i, at, neat)))
  }
  /** done resolves when every animation has ended (or was skipped). */
  done(): Promise<void> {
    return Promise.all(this.all.map((a) => a.finished.catch(() => undefined))).then(() => undefined)
  }
  /** skip ends them all at their end state. */
  skip() {
    for (const a of this.all.splice(0))
      try {
        a.finish()
      } catch {
        a.cancel()
      }
  }
}
