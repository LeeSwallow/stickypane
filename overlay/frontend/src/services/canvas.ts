// The canvas service: where a project's notes sit on this computer — the
// board window's free canvas and the notes torn off it into windows of
// their own (03 §3.6.1, ADR-0017). It is the shell's CanvasService
// (overlay/internal/shell), kept in ~/.stickypane/overlay.json; the notes'
// files and sticky.json never change because of it (RA13).

import { Call, Events } from '@wailsio/runtime'
import { canvas, canvasEvent } from '../lib/shell'

/** A note on the canvas, in the canvas's own px (before its zoom). */
export interface Place {
  x: number
  y: number
  w: number
  h: number
  /** degrees it leans */
  tilt: number
  /** its place in the stack: the higher, the nearer the top */
  z: number
}

/** A torn-off note's window on the screen. */
export interface Frame {
  screenX: number
  screenY: number
  w: number
  h: number
  pinned: boolean
}

/** the board window's layout: notes anywhere, or the terminal's grid */
export type LayoutMode = 'canvas' | 'grid'

export interface Layout {
  mode: LayoutMode
  snap: boolean
  places: Record<string, Place>
  torn: Record<string, Frame>
}

export interface CanvasApi {
  layout(): Promise<Layout>
  setMode(mode: LayoutMode): Promise<void>
  setSnap(on: boolean): Promise<void>
  /** keeps where notes are, by note name */
  place(places: Record<string, Place>): Promise<void>
  /** opens a note in a window of its own */
  tearOff(note: string): Promise<unknown>
  /** puts a torn-off note back on its board; from its own window, no name is its own */
  putBack(note?: string): Promise<void>
  /** onLayout calls fn when the window's layout changes outside it; it returns the way to stop */
  onLayout(window: number, fn: (l: Layout) => void): () => void
}

/** layoutOf reads what the shell answers, with the defaults where it says nothing. */
export function layoutOf(raw: Partial<Layout> | null | undefined): Layout {
  return {
    mode: raw?.mode === 'grid' ? 'grid' : 'canvas',
    snap: raw?.snap !== false,
    places: raw?.places ?? {},
    torn: raw?.torn ?? {},
  }
}

export const canvasService: CanvasApi = {
  layout: async () => layoutOf((await Call.ByName(canvas('Layout'))) as Partial<Layout>),
  setMode: (mode) => Call.ByName(canvas('SetMode'), mode) as Promise<void>,
  setSnap: (on) => Call.ByName(canvas('SetSnap'), on) as Promise<void>,
  place: (places) => Call.ByName(canvas('Place'), places) as Promise<void>,
  tearOff: (note) => Call.ByName(canvas('TearOff'), note),
  putBack: (note = '') => Call.ByName(canvas('PutBack'), note) as Promise<void>,
  onLayout: (window, fn) => Events.On(canvasEvent(window), (e) => fn(layoutOf(e.data as Partial<Layout>))),
}
