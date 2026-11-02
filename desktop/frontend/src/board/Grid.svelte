<script lang="ts">
  // The grid (03 §3.6): on a board of columns (the `columns` setting) that
  // many equal columns, drawn on 24 tracks (columns.ts); with `columns:
  // auto`, 12 columns, one below 720px, where the sizes decide. It is only
  // placement: given the notes in the board's order and the board's width,
  // it puts each where the layout says and draws it with `item` (a
  // NoteFrame by default). A free canvas is another component with the
  // same inputs.
  //
  // A note is moved by its grip, or by the empty part of its head, onto
  // another place: the rows reflow as it goes, the note it would land on
  // is marked, and the order is written once it is let go (onorder).
  import type { Snippet } from 'svelte'
  import { tr } from '../i18n'
  import { dropAt, moveTo, placeColumns, tracksFor, type Slot } from './columns'
  import { columnsFor, place, type Placement } from './layout'
  import type { Note } from './model'
  import NoteFrame from './NoteFrame.svelte'
  import { capture, MOVED } from './pointer'

  let {
    notes,
    width,
    columns = 0,
    item,
    onorder,
  }: {
    notes: Note[]
    width: number
    /** the columns this width draws (the setting, fewer when the window is narrow), or 0 when the sizes decide */
    columns?: number
    item?: Snippet<[Note, Placement]>
    /** the open notes in a new order, after a drag */
    onorder?: (names: string[]) => void
  } = $props()

  // while a note is dragged, the order it would make
  let preview = $state<string[] | null>(null)
  let dragging = $state<string | null>(null)
  const shown = $derived(preview ? preview.map((n) => notes.find((x) => x.name === n)).filter((x): x is Note => !!x) : notes)
  const placed = $derived(columns > 0 ? placeColumns(shown, columns) : place(shown, width))
  const tracks = $derived(tracksFor(columns, width))
  const narrow = $derived(columnsFor(width) === 1)
  let grid: HTMLElement | undefined = $state()

  function slots(): Slot[] {
    return [...(grid?.querySelectorAll<HTMLElement>('[data-slot]') ?? [])].map((el) => {
      const r = el.getBoundingClientRect()
      return { name: el.dataset.slot ?? '', left: r.left, top: r.top, right: r.right, bottom: r.bottom }
    })
  }

  // a note is moved by its grip, or by the empty part of its head (never
  // its title, which renames, or its buttons), as on the canvas
  function grab(e: PointerEvent, name: string) {
    if (!onorder || e.button !== 0 || notes.length < 2) return
    const t = e.target as HTMLElement
    if (!t.closest('[data-grip]') && !(t.closest('.head') && !t.closest('button, input, select, a, .title, [role="menu"]'))) return
    e.preventDefault()
    capture(e)
    const x0 = e.clientX
    const y0 = e.clientY
    const start = notes.map((n) => n.name)
    const mine = (f: (m: PointerEvent) => void) => (m: PointerEvent) => m.pointerId === e.pointerId && f(m)
    const move = mine((m) => {
      if (!dragging && Math.hypot(m.clientX - x0, m.clientY - y0) < MOVED) return
      dragging = name
      const to = dropAt(slots(), name, m.clientX, m.clientY)
      if (to) preview = moveTo(preview ?? start, name, to.target, to.after)
    })
    const end = (keep: boolean) =>
      mine(() => {
        window.removeEventListener('pointermove', move)
        window.removeEventListener('pointerup', done)
        window.removeEventListener('pointercancel', cancel)
        const order = preview
        preview = null
        dragging = null
        if (keep && order && order.some((n, i) => n !== start[i])) onorder?.(order)
      })
    const done = end(true)
    const cancel = end(false)
    window.addEventListener('pointermove', move)
    window.addEventListener('pointerup', done)
    window.addEventListener('pointercancel', cancel)
  }
</script>

<div class="grid" class:sorting={!!dragging} bind:this={grid} style:--cols={tracks}>
  {#each shown as note, i (note.name)}
    {@const p = placed[i]}
    <div
      class="slot"
      class:lifted={dragging === note.name}
      data-slot={note.name}
      style:grid-column="{p.column} / span {p.span}"
      style:grid-row={p.row}
      onpointerdown={(e) => grab(e, note.name)}
      role="presentation"
    >
      {#if onorder && notes.length > 1}<span class="grip" data-grip title={$tr('canvas.move')} aria-hidden="true"></span>{/if}
      {#if item}
        {@render item(note, p)}
      {:else}
        <NoteFrame {note} small={p.small} {narrow} {width} {columns} />
      {/if}
    </div>
  {/each}
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(var(--cols), minmax(0, 1fr));
    gap: var(--space-6) var(--space-4);
    align-items: start;
  }
  .slot {
    position: relative;
    min-width: 0;
  }
  /* the tape a note is carried by, as on the canvas; it shows while the
     note is hovered or holds the focus */
  .grip {
    position: absolute;
    top: calc(-1 * var(--space-2));
    left: 50%;
    z-index: 2;
    width: 48px;
    height: var(--space-3);
    transform: translateX(-50%) rotate(-2deg);
    border-radius: 2px;
    background: color-mix(in srgb, var(--ink-3) 22%, transparent);
    cursor: grab;
    touch-action: none;
    opacity: 0;
    transition: opacity var(--t-short) var(--ease);
  }
  .slot:hover .grip,
  .slot:focus-within .grip,
  .lifted .grip {
    opacity: 1;
  }
  .sorting,
  .sorting .grip {
    cursor: grabbing;
  }
  /* the note being carried stands where it would land: an outline in the
     accent marks the place, the paper under it is let through */
  .lifted {
    border-radius: 6px;
    outline: 2px dashed var(--focus);
    outline-offset: 3px;
  }
  .lifted > :global(*:not(.grip)) {
    opacity: 0.55;
  }
  @media (prefers-reduced-motion: reduce) {
    .grip {
      transition: none;
    }
  }
</style>
