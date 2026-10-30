<script lang="ts">
  // The free canvas (03 §3.6.1): the open notes of a tab, each where it was
  // left on this computer. Drag a note by its tape or the empty part of its
  // head; resize it by its right edge, its bottom edge or the corner; the
  // note touched last comes to the top. Notes land on an 8px grid unless
  // snap is off (Alt flips it while dragging). ⌘ + wheel zooms from 50% to
  // 200%. A note never placed starts where the grid would draw it, and that
  // place is kept. Every change is handed to onplace; the shell keeps it in
  // overlay.json, never in the board's files.
  import { tr } from '../i18n'
  import type { Place } from '../services/canvas'
  import { extent, firstPlaces, moved, raise, resized, samePlace, zoomed, type Edge } from './free'
  import { fitHeight } from './arrange'
  import { CARD_PX } from './layout'
  import type { Note } from './model'
  import NoteFrame from './NoteFrame.svelte'
  import { capture } from './pointer'

  let {
    notes,
    places,
    snap = true,
    scale = $bindable(1),
    onplace,
  }: {
    notes: Note[]
    places: Record<string, Place>
    snap?: boolean
    scale?: number
    /** keep these places (one note moved, raised or resized; or notes placed for the first time) */
    onplace: (places: Record<string, Place>) => void
  } = $props()

  let width = $state(0)
  // the note being dragged and where it is now; null when none is
  let drag = $state<{ name: string; at: Place } | null>(null)

  const fresh = $derived(width > 0 ? firstPlaces(notes, places, width) : {})
  const all = $derived<Record<string, Place>>({ ...fresh, ...places })
  const shown = (name: string): Place | undefined => (drag?.name === name ? drag.at : all[name])
  const size = $derived(extent(notes.map((n) => shown(n.name)).filter((p): p is Place => !!p)))

  // the notes seen for the first time get their places kept; once drawn,
  // one with no height of its own (rows) takes the height of what it holds
  // (#63: the welcome note is not 280px of mostly empty paper)
  let layer: HTMLElement | undefined = $state()
  $effect(() => {
    const f = fresh
    if (!Object.keys(f).length) return
    onplace(f)
    const names = notes.filter((n) => f[n.name] && !n.rows).map((n) => n.name)
    if (names.length) requestAnimationFrame(() => requestAnimationFrame(() => fit(names)))
  })

  /** fit measures each note drawn at its natural height once and keeps that height. */
  function fit(names: string[]) {
    const out: Record<string, Place> = {}
    for (const name of names) {
      const el = layer?.querySelector<HTMLElement>(`[data-place="${CSS.escape(name)}"]`)
      const p = all[name]
      if (!el || !p) continue
      const was = el.style.height
      el.style.height = 'auto'
      const natural = el.offsetHeight
      el.style.height = was
      const h = fitHeight(natural, p.h)
      if (h !== p.h) out[name] = { ...p, h }
    }
    if (Object.keys(out).length) onplace(out)
  }

  function toTop(name: string) {
    const up = raise(all, name)
    if (up) onplace({ [name]: up })
  }

  /**
   * follow follows the pointer from e until it is let go (keep the place
   * next gives) or taken away (keep only the raise). The note is on top of
   * the stack while it moves.
   */
  function follow(e: PointerEvent, name: string, next: (start: Place, dx: number, dy: number, snapOn: boolean) => Place) {
    const start = all[name]
    if (!start || e.button !== 0) return
    e.preventDefault()
    e.stopPropagation()
    // the window's events, not the handle's: the webview may refuse the
    // capture, and the pointer leaves the handle as it moves (#66)
    capture(e)
    const x0 = e.clientX
    const y0 = e.clientY
    const up = raise(all, name)
    const z = (up ?? start).z
    drag = { name, at: { ...start, z } }
    const mine = (f: (m: PointerEvent) => void) => (m: PointerEvent) => m.pointerId === e.pointerId && f(m)
    const move = mine((m) => (drag = { name, at: { ...next(start, m.clientX - x0, m.clientY - y0, m.altKey ? !snap : snap), z } }))
    const end = (keep: boolean) =>
      mine(() => {
        window.removeEventListener('pointermove', move)
        window.removeEventListener('pointerup', done)
        window.removeEventListener('pointercancel', cancel)
        const at = drag?.at
        drag = null
        if (keep && at && !samePlace(at, start)) onplace({ [name]: at })
        else if (up) onplace({ [name]: up })
      })
    const done = end(true)
    const cancel = end(false)
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', done)
    window.addEventListener('pointercancel', cancel)
  }

  const startMove = (e: PointerEvent, name: string) => follow(e, name, (s, dx, dy, on) => moved(s, dx, dy, scale, on))
  const startResize = (e: PointerEvent, name: string, edge: Edge) => follow(e, name, (s, dx, dy, on) => resized(s, edge, dx, dy, scale, on))

  // a note is dragged by its tape, or by the empty part of its head (never
  // by its title, which renames, or its buttons)
  function grab(e: PointerEvent, name: string) {
    if (e.button !== 0) return
    const t = e.target as HTMLElement
    if (t.closest('[data-grip]') || (t.closest('.head') && !t.closest('button, input, select, a, .title, [role="menu"]'))) startMove(e, name)
    else toTop(name)
  }

  function wheel(e: WheelEvent) {
    if (!e.metaKey && !e.ctrlKey) return
    e.preventDefault()
    scale = zoomed(scale, e.deltaY)
  }
</script>

<div class="canvas" bind:clientWidth={width} onwheel={wheel} role="region" aria-label={$tr('ui.view.canvas')} data-canvas>
  <div class="sizer" style:width="{size.w * scale}px" style:height="{size.h * scale}px">
    <div class="layer" bind:this={layer} style:width="{size.w}px" style:height="{size.h}px" style:transform="scale({scale})">
      {#each notes as note (note.name)}
        {@const p = shown(note.name)}
        {#if p}
          <div
            class="item"
            class:moving={drag?.name === note.name}
            data-place={note.name}
            style:left="{p.x}px"
            style:top="{p.y}px"
            style:width="{p.w}px"
            style:height="{p.h}px"
            style:z-index={p.z}
            style:--tilt="{p.tilt}deg"
            onpointerdown={(e) => grab(e, note.name)}
            onfocusin={() => drag || toTop(note.name)}
            role="presentation"
          >
            <span class="grip" data-grip title={$tr('canvas.move')} aria-hidden="true"></span>
            <NoteFrame {note} free small={p.w < CARD_PX} narrow={p.w < CARD_PX} width={p.w} />
            {#if !note.error}
              {#each ['e', 's', 'se'] as const as edge (edge)}
                <span
                  class="handle {edge}"
                  title={$tr('canvas.resize')}
                  aria-hidden="true"
                  onpointerdown={(e) => startResize(e, note.name, edge)}
                ></span>
              {/each}
            {/if}
          </div>
        {/if}
      {/each}
    </div>
  </div>
  {#if scale !== 1}
    <button class="zoom" type="button" title={$tr('canvas.zoomReset')} onclick={() => (scale = 1)}>{Math.round(scale * 100)}%</button>
  {/if}
</div>

<style>
  .canvas {
    position: relative;
    flex: 1;
    min-height: 100%;
  }
  .sizer {
    position: relative;
    min-width: 100%;
    min-height: 100%;
  }
  .layer {
    position: absolute;
    left: 0;
    top: 0;
    transform-origin: 0 0;
  }
  .item {
    position: absolute;
    box-sizing: border-box;
    transform: rotate(var(--tilt));
    transition: transform var(--dur-fast, 120ms) ease;
  }
  .item:focus-within {
    transform: rotate(0deg);
  }
  .item.moving {
    transform: rotate(0deg) scale(1.01);
    filter: drop-shadow(0 6px 10px rgb(0 0 0 / 0.18));
    transition: none;
  }
  .item > :global(.frame) {
    height: 100%;
  }
  /* the tape a note is carried by */
  .grip {
    position: absolute;
    top: calc(-1 * var(--space-2));
    left: 50%;
    z-index: 2;
    width: 56px;
    height: var(--space-4);
    transform: translateX(-50%) rotate(-2deg);
    border-radius: 2px;
    background: color-mix(in srgb, var(--ink-3) 22%, transparent);
    cursor: grab;
    touch-action: none;
  }
  .moving .grip {
    cursor: grabbing;
  }
  .handle {
    position: absolute;
    z-index: 2;
    touch-action: none;
  }
  /* 12px across, centred on the border drawn (#66); a mark on each shows
     while the note is hovered, in the accent while its handle is */
  .handle.e {
    top: 0;
    bottom: var(--space-4);
    right: -6px;
    width: 12px;
    cursor: ew-resize;
  }
  .handle.s {
    left: 0;
    right: var(--space-4);
    bottom: -6px;
    height: 12px;
    cursor: ns-resize;
  }
  .handle.se {
    right: -6px;
    bottom: -6px;
    width: 22px;
    height: 22px;
    cursor: nwse-resize;
  }
  .handle::after {
    content: '';
    position: absolute;
    border-radius: 1px;
    background: var(--ink-3);
    opacity: 0;
  }
  .handle.e::after {
    left: 5px;
    top: calc(50% - 12px);
    width: 2px;
    height: 24px;
  }
  .handle.s::after {
    top: 5px;
    left: calc(50% - 12px);
    width: 24px;
    height: 2px;
  }
  .handle.se::after {
    right: 6px;
    bottom: 6px;
    width: var(--space-2);
    height: var(--space-2);
    border-radius: 0;
    background: none;
    border-right: var(--stroke-emphasis, 2px) solid var(--ink-3);
    border-bottom: var(--stroke-emphasis, 2px) solid var(--ink-3);
  }
  .item:hover .handle::after {
    opacity: 0.6;
  }
  .item .handle:hover::after,
  .moving .handle::after {
    opacity: 1;
    background: var(--accent);
  }
  .item .handle.se:hover::after,
  .moving .handle.se::after {
    background: none;
    border-color: var(--accent);
  }
  .zoom {
    position: sticky;
    left: calc(100% - 64px);
    bottom: var(--space-3);
    float: right;
    margin: var(--space-3);
    padding: var(--space-1) var(--space-2);
    border: var(--stroke-hairline) solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--paper);
    color: var(--ink-2);
    font: var(--weight-mono-md) var(--text-mono-md) / 1 var(--font-mono);
    cursor: pointer;
  }
</style>
