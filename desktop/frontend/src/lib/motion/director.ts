// The director: what the board window does with a board.changed. It
// measures the window, lets the board be read and drawn again, asks the
// plan what moved and has the player move it. The window gives it its
// parts as functions, so it holds no state of the board's own.

import type { Board } from '../../board/model'
import type { Look } from './keyframes'
import { asChange, leaving, planChange, type Plan } from './plan'
import { keyOf, Player, snapshot } from './play'

export interface Stage {
  /** the element the notes are drawn in, or null when none is */
  root(): HTMLElement | null
  /** the board drawn now */
  board(): Board | null
  /** reads the board again and draws it */
  refresh(): Promise<void>
  /** waits until what was read is on screen (Svelte's tick) */
  settle(): Promise<void>
  /** the frames on screen: the open notes of the tab, or the zoomed one */
  visible(): Set<string>
  look(): Look
  colorOf?(who: string): string | undefined
  /** the window is out of sight: nothing is played (04 §4.4) */
  hidden?(): boolean
  /** told the plan played, for the dev page */
  played?(p: Plan): void
}

export interface Director {
  changed(data: unknown): Promise<void>
  stop(): void
}

export function director(s: Stage): Director {
  const player = new Player()
  return {
    async changed(data) {
      const look = s.look()
      const root = s.root()
      const prev = s.board()
      if (look.level === 'off' || s.hidden?.() || !root || !prev) return s.refresh()
      const change = asChange(data)
      player.finish()
      const keys = new Set(leaving(prev, change).map(keyOf).filter((k): k is string => !!k))
      const before = snapshot(root, keys)
      await s.refresh()
      await s.settle()
      const next = s.board()
      const now = s.root()
      if (!next || !now) return
      const plan = planChange(prev, next, change, { visible: s.visible() })
      player.play(now, before, plan, { look: s.look(), colorOf: s.colorOf })
      s.played?.(plan)
    },
    stop: () => player.finish(),
  }
}
