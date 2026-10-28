<script lang="ts">
  // One note on the board (03 §3.6–3.8, 10 §10.1): its frame, its header
  // (title, changed dot, link, summary, book pages, size, pin, colour,
  // close, zoom or back) and its body, which the shape draws. The bottom
  // edge drags the height (rows). Everything it does goes through the
  // board context, so the same frame works zoomed, on a canvas or alone.
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import { NOTE_NAMES } from '../lib/theme/palette'
  import { tr } from '../i18n'
  import Shape from '../shapes/Shape.svelte'
  import { noteHex, useBoard } from './context'
  import { heightOfRows, rowsFromHeight, SIZES } from './layout'
  import type { Note } from './model'

  let { note, zoomed = false, small = false, narrow = false }: { note: Note; zoomed?: boolean; small?: boolean; narrow?: boolean } =
    $props()
  const board = useBoard()

  const pages = $derived(note.pages ?? [])
  const page = $derived(pages.length ? board.pageOf(note.name, pages.length) : 0)
  const shown = $derived(pages.length ? pages[page] : note)
  const color = $derived(noteHex(note.color, look.palette))
  const changed = $derived(board.changed(note) || pages.some((p) => board.changed(p)))
  const sizeIcon: Record<string, string> = { card: '▢', half: '▭', page: '▬' }

  // the height being dragged, in px; null when not dragging
  let dragH: number | null = $state(null)
  let body: HTMLElement | undefined = $state()
  const fixed = $derived(zoomed ? null : (dragH ?? heightOfRows(note.rows)))

  let editing = $state(false)
  let draft = $state('')
  let colours = $state(false)

  function startDrag(e: PointerEvent) {
    if (!body) return
    e.preventDefault()
    const el = e.currentTarget as HTMLElement
    el.setPointerCapture(e.pointerId)
    const y0 = e.clientY
    const h0 = body.clientHeight
    const move = (m: PointerEvent) => (dragH = Math.max(0, h0 + m.clientY - y0))
    const up = () => {
      el.removeEventListener('pointermove', move)
      el.removeEventListener('pointerup', up)
      el.removeEventListener('pointercancel', up)
      if (dragH !== null) {
        const rows = rowsFromHeight(dragH)
        dragH = null
        if (rows !== note.rows) board.arrange(note.name, { rows })
      }
    }
    el.addEventListener('pointermove', move)
    el.addEventListener('pointerup', up)
    el.addEventListener('pointercancel', up)
  }

  function rename() {
    draft = note.title
    editing = true
  }
  function commit() {
    editing = false
    const t = draft.trim()
    if (t && t !== note.title) board.arrange(note.name, { title: t })
  }

  // a double-click on the note (not on its controls or its title) zooms it
  function dbl(e: MouseEvent) {
    const t = e.target as HTMLElement
    if (zoomed || t.closest('button, input, a, select, .edge, .title')) return
    board.zoom(note.name)
  }
</script>

<!-- svelte-ignore a11y_no_noninteractive_tabindex a11y_no_static_element_interactions -->
<div
  class="frame"
  class:zoomed
  class:small
  data-note={note.name}
  tabindex="0"
  role="group"
  aria-label={note.title}
  ondblclick={dbl}
  onfocusin={() => board.seen(shown.name)}
  onpointerenter={() => board.seen(shown.name)}
>
  <NoteFrame {color} id={note.name}>
    <header>
      {#if zoomed && board.canBack()}
        <button class="ctl" title={$tr('board.back')} aria-label={$tr('board.back')} onclick={() => board.back()}>←</button>
      {/if}
      <span class="icon" aria-hidden="true">{note.icon}</span>
      {#if editing}
        <!-- svelte-ignore a11y_autofocus -->
        <input
          class="title"
          bind:value={draft}
          autofocus
          onkeydown={(e) => {
            if (e.key === 'Enter') commit()
            if (e.key === 'Escape') {
              e.stopPropagation()
              editing = false
            }
          }}
          onblur={commit}
        />
      {:else}
        <!-- svelte-ignore a11y_no_static_element_interactions -->
        <span class="title" style:color title={$tr('note.rename')} ondblclick={rename}>{note.title}</span>
      {/if}
      {#if changed}<span class="changed" title={$tr('note.changed')} aria-label={$tr('note.changed')}>•</span>{/if}
      {#if note.linked}<span class="linked" title={$tr('note.linked', { target: note.target ?? '' })}>🔗</span>{/if}
      <span class="summary">{shown.summary}</span>
      {#if pages.length > 1}
        <span class="pages">
          <button class="ctl" disabled={page === 0} title={$tr('note.pagePrev')} aria-label={$tr('note.pagePrev')} onclick={() => board.flip(note.name, -1)}>‹</button>
          <select aria-label={$tr('note.pages')} value={page} onchange={(e) => board.flip(note.name, 0, Number(e.currentTarget.value))}>
            {#each pages as p, i (p.name)}<option value={i}>{$tr('note.page', { n: i + 1, m: pages.length })} · {p.title}</option>{/each}
          </select>
          <button class="ctl" disabled={page === pages.length - 1} title={$tr('note.pageNext')} aria-label={$tr('note.pageNext')} onclick={() => board.flip(note.name, 1)}>›</button>
        </span>
      {/if}
      <span class="tools">
        {#each SIZES as s (s)}
          <button
            class="ctl"
            aria-pressed={note.size === s}
            title={$tr(`note.size.${s}`)}
            aria-label={$tr(`note.size.${s}`)}
            onclick={() => board.arrange(note.name, { size: s })}>{sizeIcon[s]}</button
          >
        {/each}
        <button
          class="ctl"
          aria-pressed={note.pinned}
          title={$tr(note.pinned ? 'note.unpin' : 'note.pin')}
          aria-label={$tr(note.pinned ? 'note.unpin' : 'note.pin')}
          onclick={() => board.arrange(note.name, { pin: !note.pinned })}>📌</button
        >
        <button class="ctl dot" style:color title={$tr('note.color')} aria-label={$tr('note.color')} aria-expanded={colours} onclick={() => (colours = !colours)}>●</button>
        {#if !zoomed}
          <button class="ctl" title={$tr('note.zoom')} aria-label={$tr('note.zoom')} onclick={() => board.zoom(note.name)}>⤢</button>
        {/if}
        <button class="ctl" title={$tr('note.close')} aria-label={$tr('note.close')} onclick={() => board.arrange(note.name, { open: false })}>×</button>
      </span>
    </header>
    {#if colours}
      <div class="swatches" role="menu">
        {#each NOTE_NAMES as c (c)}
          <button
            class="ctl"
            role="menuitemradio"
            aria-checked={note.color === c}
            style:color={noteHex(c, look.palette)}
            title={$tr(`note.color.${c}`)}
            aria-label={$tr(`note.color.${c}`)}
            onclick={() => {
              colours = false
              board.arrange(note.name, { color: c })
            }}>●</button
          >
        {/each}
      </div>
    {/if}
    <div class="body" bind:this={body} class:fixed={fixed !== null} style:height={fixed !== null ? `${fixed}px` : null}>
      {#key shown.name}
        <Shape note={shown} narrow={narrow || small} />
      {/key}
    </div>
    {#if !zoomed}
      <!-- svelte-ignore a11y_no_static_element_interactions -->
      <div class="edge" title={$tr('note.height')} onpointerdown={startDrag} ondblclick={() => board.arrange(note.name, { rows: 0 })}></div>
    {/if}
  </NoteFrame>
</div>

<style>
  .frame {
    position: relative;
    outline: none;
    border-radius: var(--radius);
  }
  .frame:focus-visible {
    box-shadow: 0 0 0 2px var(--accent);
  }
  .frame.zoomed {
    height: 100%;
  }
  .frame.zoomed :global(.note) {
    height: 100%;
    box-sizing: border-box;
    display: flex;
    flex-direction: column;
  }
  .frame.zoomed :global(.note > .body) {
    display: flex;
    flex-direction: column;
    min-height: 0;
    flex: 1;
  }
  header {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1) var(--space-2);
    margin-bottom: var(--space-2);
  }
  .icon {
    color: var(--pencil);
  }
  .title {
    font: 700 var(--hand-title) var(--font-hand);
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  input.title {
    color: var(--ink);
    background: var(--paper);
    border: var(--stroke) solid var(--accent);
    border-radius: var(--radius);
  }
  .small .title {
    font-size: var(--hand-lg);
  }
  .changed {
    color: var(--accent);
    font-weight: 700;
  }
  .summary {
    margin-left: auto;
    color: var(--pencil);
    font: 700 var(--hand-md) var(--font-hand);
    font-variant-numeric: tabular-nums;
  }
  .pages,
  .tools {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
  }
  .pages select {
    font: var(--text-sm) var(--font-mono);
    color: var(--ink);
    background: var(--paper);
    border: var(--stroke) solid var(--grid-major);
    border-radius: var(--radius);
    max-width: 14em;
  }
  .ctl {
    min-width: var(--control);
    height: var(--control);
    padding: 0 var(--space-1);
    border: var(--stroke) solid transparent;
    border-radius: var(--radius);
    background: none;
    color: var(--pencil);
    font: var(--text-md) var(--font-body);
    cursor: pointer;
  }
  .ctl:hover:not(:disabled),
  .ctl:focus-visible {
    border-color: var(--grid-major);
    color: var(--ink);
  }
  .ctl[aria-pressed='true'] {
    color: var(--accent);
    border-color: var(--accent);
  }
  .ctl:disabled {
    opacity: var(--dim);
    cursor: default;
  }
  .dot {
    font-size: var(--hand-md);
  }
  .swatches {
    display: flex;
    gap: var(--space-1);
    margin: 0 0 var(--space-2);
  }
  .swatches .ctl {
    font-size: var(--hand-lg);
  }
  .body.fixed {
    overflow: auto;
  }
  .zoomed .body {
    flex: 1;
    min-height: 0;
    overflow: auto;
  }
  .edge {
    position: absolute;
    left: 0;
    right: 0;
    bottom: calc(-1 * var(--space-3));
    height: var(--edge);
    cursor: ns-resize;
    touch-action: none;
  }
  .edge:hover {
    border-bottom: var(--stroke) dashed var(--accent);
  }
</style>
