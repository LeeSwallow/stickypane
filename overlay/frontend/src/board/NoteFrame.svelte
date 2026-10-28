<script lang="ts">
  // One note on the board (03 §3.6–3.8, 10 §10.1): its frame, its header
  // (the design system's NoteHeader: a way back when zoomed, the title with
  // rename, the changed dot, the link, the summary, a book's pages, the
  // width as a fraction, pin, ⋯ with zoom, colour and the editor, close)
  // and its body, which the shape draws. The bottom edge drags the height
  // (rows), the right edge the width (it lands on the nearest fraction).
  // Everything it does goes through the board context, so the same frame
  // works zoomed, on a canvas or alone. free: the canvas or a torn-off
  // note's window sizes it, so it fills its box and has no edges or width
  // steps of the grid's.
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import { NOTE_NAMES } from '../lib/theme/palette'
  import { fractionGlyph } from '../lib/sizes'
  import IconButton from '../lib/ui/IconButton.svelte'
  import NoteHeader from '../lib/ui/NoteHeader.svelte'
  import type { MenuItem } from '../lib/ui/menu'
  import { tr } from '../i18n'
  import Shape from '../shapes/Shape.svelte'
  import { noteHex, useBoard } from './context'
  import { HeightDrag, heightOfRows, sizeShown, WidthDrag, type Fraction } from './layout'
  import type { Note } from './model'
  import { manageItems, type Manage } from './manage'

  let {
    note,
    zoomed = false,
    small = false,
    narrow = false,
    free = false,
    width = 1200,
  }: {
    note: Note
    zoomed?: boolean
    small?: boolean
    narrow?: boolean
    /** sized from outside (the canvas, a torn-off window): fills its box */
    free?: boolean
    /** the board's width in px: what a fraction is a fraction of */
    width?: number
  } = $props()
  const board = useBoard()

  const pages = $derived(note.pages ?? [])
  const page = $derived(pages.length ? board.pageOf(note.name, pages.length) : 0)
  const shown = $derived(pages.length ? pages[page] : note)
  const color = $derived(noteHex(note.color, look.palette))
  const changed = $derived(board.changed(note) || pages.some((p) => board.changed(p)))
  // a note that cannot be read can only be zoomed and closed (10 §10.1)
  const readable = $derived(!note.error)
  const size = $derived(sizeShown(note.size, width))

  // the height being dragged, in px; null when not dragging
  let dragH: number | null = $state(null)
  // the width being dragged, in px, and the fraction it lands on
  let dragW: number | null = $state(null)
  let landing: Fraction | null = $state(null)
  let body: HTMLElement | undefined = $state()
  const fixed = $derived(zoomed || free ? null : (dragH ?? heightOfRows(note.rows)))

  let colours = $state(false)

  // the ⋯ items that act on the note as a whole, with the terminal's keys
  const manageItem = $derived<Record<Manage, MenuItem>>({
    run: { id: 'run', label: $tr('note.run'), icon: 'run' },
    send: { id: 'send', label: $tr('note.send'), icon: 'send' },
    move: { id: 'move', label: $tr('note.move'), icon: 'tab' },
    archive: { id: 'archive', label: $tr('note.archive'), icon: 'archive', shortcut: 'x' },
    trash: { id: 'trash', label: $tr('note.trash'), icon: 'trash', shortcut: 'D', danger: true },
  })

  const menu = $derived<MenuItem[]>([
    ...(zoomed || board.putBack ? [] : [{ id: 'zoom', label: $tr('note.zoom'), icon: 'zoom' as const, shortcut: '↵' }]),
    ...(board.putBack ? [{ id: 'putBack', label: $tr('note.putBack'), icon: 'restore' as const }] : []),
    ...(readable ? [{ id: 'color', label: $tr('note.color') }] : []),
    ...(readable && !pages.length && board.edit ? [{ id: 'edit', label: $tr('note.openInEditor'), icon: 'edit-external' as const, shortcut: 'E' }] : []),
    ...(board.manage ? [{ kind: 'separator' as const, id: 'sep-manage' }, ...manageItems(note).map((m) => manageItem[m])] : []),
  ])

  function choose(id: string) {
    if (id === 'zoom') board.zoom(note.name)
    else if (id === 'color') colours = !colours
    else if (id === 'edit') board.edit?.(note.name)
    else if (id === 'putBack') board.putBack?.(note.name)
    else if (id in manageItem) board.manage?.(note, id as Manage)
  }

  /** drag follows a pointer on an edge until it is let go (save) or taken away (cancel). */
  function drag(e: PointerEvent, move: (m: PointerEvent) => void, done: (save: boolean) => void) {
    e.preventDefault()
    const el = e.currentTarget as HTMLElement
    el.setPointerCapture(e.pointerId)
    const end = (save: boolean) => () => {
      el.removeEventListener('pointermove', move)
      el.removeEventListener('pointerup', up)
      el.removeEventListener('pointercancel', cancel)
      done(save)
    }
    const up = end(true)
    const cancel = end(false) // the system took the pointer: keep what it had
    el.addEventListener('pointermove', move)
    el.addEventListener('pointerup', up)
    el.addEventListener('pointercancel', cancel)
  }

  function startHeight(e: PointerEvent) {
    if (!body) return
    const d = new HeightDrag(body.clientHeight, e.clientY, note.rows)
    drag(
      e,
      (m) => (dragH = d.move(m.clientY)),
      (save) => {
        dragH = null
        const rows = save ? d.release() : d.cancel()
        if (rows !== null) board.arrange(note.name, { rows })
      },
    )
  }

  // the drag starts from the width the note asks for, not the width its
  // row stretched it to (layout.ts WidthDrag)
  function startWidth(e: PointerEvent) {
    const d = new WidthDrag(note.size, width, e.clientX)
    drag(
      e,
      (m) => {
        const at = d.move(m.clientX)
        dragW = at.px
        landing = at.landing
      },
      (save) => {
        dragW = null
        landing = null
        const to = save ? d.release() : d.cancel()
        if (to) board.arrange(note.name, { size: to })
      },
    )
  }

  // a double-click on the note (not on its controls or its title) zooms it
  function dbl(e: MouseEvent) {
    const t = e.target as HTMLElement
    if (zoomed || board.putBack || t.closest('button, input, a, select, .edge, .title')) return
    board.zoom(note.name)
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex a11y_no_static_element_interactions -->
<div
  class="frame"
  class:zoomed
  class:free
  class:small
  class:sizing={dragW !== null}
  style:width={dragW !== null ? `${dragW}px` : null}
  data-frame={note.name}
  tabindex="0"
  role="group"
  aria-label={note.title}
  ondblclick={dbl}
  onfocusin={() => board.seen(shown.name)}
  onpointerenter={() => board.seen(shown.name)}
>
  <NoteFrame {color} id={note.name}>
    <NoteHeader
      title={note.title}
      {color}
      summary={shown.summary}
      updated={changed}
      {size}
      pinned={note.pinned}
      linked={note.linked ? (note.target ?? '') : undefined}
      editable={readable}
      book={pages.length > 1
        ? {
            page: page + 1,
            total: pages.length,
            onprev: () => board.flip(note.name, -1),
            onnext: () => board.flip(note.name, 1),
            names: pages.map((p) => p.title),
            onpage: (i) => board.flip(note.name, 0, i),
          }
        : undefined}
      {menu}
      sizable={!free}
      onsize={(s) => board.arrange(note.name, { size: s })}
      onpin={(on) => board.arrange(note.name, { pin: on })}
      ontearoff={board.tear && readable && !zoomed ? () => board.tear?.(note.name) : undefined}
      onrename={(title) => board.arrange(note.name, { title })}
      onmenu={choose}
      onclose={board.putBack ? () => board.putBack?.(note.name) : () => board.arrange(note.name, { open: false })}
    >
      {#snippet lead()}
        {#if zoomed && board.canBack()}
          <IconButton icon="back" size="sm" label={$tr('board.back')} shortcut="Esc" align="start" onclick={() => board.back()} />
        {/if}
        {#if note.icon}<span class="icon" aria-hidden="true">{note.icon}</span>{/if}
      {/snippet}
    </NoteHeader>
    {#if colours && readable}
      <div class="swatches" role="menu" aria-label={$tr('note.color')}>
        {#each NOTE_NAMES as c (c)}
          <button
            class="swatch"
            role="menuitemradio"
            aria-checked={note.color === c}
            style:--swatch={noteHex(c, look.palette)}
            title={$tr(`note.color.${c}`)}
            aria-label={$tr(`note.color.${c}`)}
            onclick={() => {
              colours = false
              board.arrange(note.name, { color: c })
            }}
          ></button>
        {/each}
      </div>
    {/if}
    <div class="body" bind:this={body} class:fixed={fixed !== null} style:height={fixed !== null ? `${fixed}px` : null}>
      {#key shown.name}
        <Shape note={shown} narrow={narrow || small} />
      {/key}
    </div>
    {#if !zoomed && !free && readable}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="edge" title={$tr('note.height')} onpointerdown={startHeight} ondblclick={() => board.arrange(note.name, { rows: 0 })}></div>
      {#if !narrow}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <div class="edge side" title={$tr('note.width')} onpointerdown={startWidth}></div>
      {/if}
    {/if}
  </NoteFrame>
  {#if landing}<span class="landing" role="status" aria-label={$tr('ui.size.now', { size: landing })}>{fractionGlyph(landing)}</span>{/if}
</div>

<style>
  .frame {
    position: relative;
    outline: none;
    border-radius: var(--radius-sm);
  }
  .frame:focus-visible {
    box-shadow: 0 0 0 var(--focus-width) var(--focus);
  }
  .frame.sizing {
    z-index: var(--z-ring);
  }
  .frame.zoomed,
  .frame.free {
    height: 100%;
  }
  .frame.zoomed :global(.note),
  .frame.free :global(.note) {
    height: 100%;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
  }
  .frame.zoomed :global(.note > .body),
  .frame.free :global(.note > .body) {
    display: flex;
    flex-direction: column;
    min-height: 0;
    flex: 1;
  }
  .icon {
    flex: none;
    color: var(--ink-3);
    font: var(--weight-body-md) var(--text-body-md) / 1 var(--font-body);
  }
  .swatches {
    display: flex;
    gap: var(--space-2);
    margin: var(--space-1) 0 var(--space-2);
  }
  .swatch {
    width: var(--control-sm);
    height: var(--control-sm);
    padding: 0;
    border: var(--stroke-hairline) solid var(--line);
    border-radius: 50%;
    background: var(--swatch);
    cursor: pointer;
  }
  .swatch[aria-checked='true'] {
    box-shadow: 0 0 0 var(--focus-offset) var(--paper), 0 0 0 calc(var(--focus-offset) + var(--stroke-emphasis)) var(--swatch);
  }
  .swatch:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .body {
    margin-top: var(--space-2);
  }
  .body.fixed {
    overflow: auto;
  }
  .zoomed .body,
  .free .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .edge {
    position: absolute;
    left: 0;
    right: 0;
    bottom: calc(-1 * var(--space-3));
    height: var(--space-2);
    cursor: ns-resize;
    touch-action: none;
  }
  .edge:hover {
    border-bottom: var(--stroke-hairline) dashed var(--accent);
  }
  .edge.side {
    left: auto;
    top: 0;
    bottom: 0;
    right: calc(-1 * var(--space-3));
    width: var(--space-2);
    height: auto;
    cursor: ew-resize;
  }
  .edge.side:hover {
    border-bottom: 0;
    border-right: var(--stroke-hairline) dashed var(--accent);
  }
  .landing {
    position: absolute;
    top: var(--space-2);
    right: var(--space-2);
    z-index: var(--z-tooltip);
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--tip);
    color: var(--tip-ink);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    pointer-events: none;
  }
</style>
