// How a project window hears its bridge: it asks the shell for its window id
// (again, while that fails) and subscribes to the window's events as soon
// as it knows it, whether or not the first board load worked. Closing the
// window stops the asking and removes what was subscribed, even when the
// answer comes after.

import { changedEvent, restartedEvent, statusEvent } from '../lib/shell'

export interface Bus {
  on(name: string, fn: (e: { data?: unknown }) => void): () => void
}

export interface Session {
  status: () => Promise<{ window: number }>
  bus: Bus
  onChanged: (data: unknown) => void
  onRestarted: () => void
  onStatus: () => void
  /** waits between asks; a test passes one that does not wait */
  sleep?: (ms: number) => Promise<void>
}

const wait = (ms: number) => new Promise<void>((r) => setTimeout(r, ms))

/** connect subscribes the window to its events; the function it returns undoes it. */
export function connect(s: Session): () => void {
  const sleep = s.sleep ?? wait
  let closed = false
  const offs: (() => void)[] = []
  ;(async () => {
    for (let delay = 250; !closed; delay = Math.min(delay * 2, 5000)) {
      let id: number
      try {
        id = (await s.status()).window
      } catch {
        await sleep(delay)
        continue
      }
      if (closed) return
      offs.push(s.bus.on(changedEvent(id), (e) => s.onChanged(e?.data)))
      offs.push(s.bus.on(restartedEvent(id), () => s.onRestarted()))
      offs.push(s.bus.on(statusEvent(id), () => s.onStatus()))
      return
    }
  })()
  return () => {
    closed = true
    offs.splice(0).forEach((off) => off())
  }
}
