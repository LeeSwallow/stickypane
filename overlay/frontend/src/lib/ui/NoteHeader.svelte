<script lang="ts" module>
  export type NoteSize = 'card' | 'half' | 'page'
</script>

<script lang="ts">
  // The head of a note: title (hand-written, in the note's colour, one line),
  // a changed dot, the summary (3/6, 6 cards), a book's pager, and the
  // note's controls — size ▢▭▬, pin, tear-off, more ⋯. It holds controls,
  // so it never leans; the controls stay quiet until the note is hovered.
  import { tr } from '../../i18n'
  import { look } from '../doodle/look.svelte'
  import { noteTitleColor } from '../theme/tokens'
  import IconButton from './IconButton.svelte'
  import Menu from './Menu.svelte'
  import type { MenuItem } from './menu'
  import SegmentedControl from './SegmentedControl.svelte'
  import Toolbar from './Toolbar.svelte'

  let {
    title,
    color,
    summary,
    updated = false,
    size = $bindable('half'),
    pinned = false,
    book,
    menu = [],
    menuOpen = $bindable(false),
    onsize,
    onpin,
    ontearoff,
    onmenu,
  }: {
    title: string
    color: string
    summary?: string
    updated?: boolean
    size?: NoteSize
    pinned?: boolean
    book?: { page: number; total: number; onprev: () => void; onnext: () => void }
    menu?: MenuItem[]
    menuOpen?: boolean
    onsize?: (s: NoteSize) => void
    onpin?: (on: boolean) => void
    ontearoff?: () => void
    onmenu?: (id: string) => void
  } = $props()

  const ink = $derived(look.palette ? noteTitleColor(color, look.palette, look.dark) : color)
  const ORDER: NoteSize[] = ['card', 'half', 'page']
  function step() {
    size = ORDER[(ORDER.indexOf(size) + 1) % ORDER.length]
    onsize?.(size)
  }
  const sizes = $derived([
    { value: 'card', label: $tr('ui.size.card'), icon: 'size-card' as const },
    { value: 'half', label: $tr('ui.size.half'), icon: 'size-half' as const },
    { value: 'page', label: $tr('ui.size.page'), icon: 'size-page' as const },
  ])
</script>

<div class="wrap">
<header class="head">
  <h3 class="title" class:neat={look.style === 'neat'} style:color={ink} title={title}>{title}</h3>
  {#if updated}<span class="dot" style:background={color} role="img" aria-label={$tr('ui.updated')}></span>{/if}
  {#if summary}<span class="summary">{summary}</span>{/if}
  <span class="grow"></span>
  {#if book}
    <span class="pager">
      <IconButton icon="book-prev" size="sm" label={$tr('ui.page.prev')} shortcut="[" disabled={book.page <= 1} onclick={book.onprev} />
      <span class="page" aria-label={$tr('ui.page.of', { n: book.page, total: book.total })}>{book.page} / {book.total}</span>
      <IconButton icon="book-next" size="sm" label={$tr('ui.page.next')} shortcut="]" disabled={book.page >= book.total} onclick={book.onnext} />
    </span>
  {/if}
  <span class="tools">
    <Toolbar label={$tr('ui.noteActions')}>
      <span class="sizes wide">
        <SegmentedControl options={sizes} bind:value={size} label={$tr('ui.size')} iconOnly size="sm" onchange={(v) => onsize?.(v as NoteSize)} />
      </span>
      <!-- in a narrow note the three become one button that steps card → half → page -->
      <span class="sizes narrow">
        <IconButton icon={`size-${size}`} size="sm" label="{$tr('ui.size')}: {$tr(`ui.size.${size}`)}" shortcut="=" onclick={step} />
      </span>
      <IconButton
        icon="pin"
        size="sm"
        label={pinned ? $tr('ui.unpin') : $tr('ui.pin')}
        pressed={pinned}
        crayon={pinned}
        onclick={() => onpin?.(!pinned)}
      />
      <IconButton icon="tear-off" size="sm" label={$tr('ui.tearOff')} onclick={ontearoff} />
      <span class="anchor">
        <IconButton icon="more" size="sm" label={$tr('ui.more')} aria-haspopup="menu" aria-expanded={menuOpen} align="end" onclick={() => (menuOpen = !menuOpen)} />
        <Menu items={menu} bind:open={menuOpen} label={$tr('ui.noteActions')} onselect={onmenu} />
      </span>
    </Toolbar>
  </span>
</header>
</div>

<style>
  .wrap {
    container-type: inline-size;
    min-width: 0;
  }
  .sizes {
    display: inline-flex;
  }
  .narrow {
    display: none;
  }
  @container (max-width: 460px) {
    .wide {
      display: none;
    }
    .narrow {
      display: inline-flex;
    }
  }
  .head {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-height: var(--control-md);
    min-width: 0;
  }
  .title {
    flex: 0 1 auto;
    min-width: min(9em, 45%);
    margin: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  .title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .dot {
    width: 6px;
    height: 6px;
    flex: none;
    border-radius: 50%;
  }
  .summary {
    flex: none;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .grow {
    flex: 1;
  }
  .pager {
    display: inline-flex;
    align-items: center;
    gap: 2px;
    flex: none;
  }
  .page {
    min-width: 36px;
    text-align: center;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .tools {
    flex: none;
    opacity: 0.7;
    transition: opacity var(--t-short) var(--ease);
  }
  .head:hover .tools,
  .head:focus-within .tools {
    opacity: 1;
  }
  .anchor {
    position: relative;
    display: inline-flex;
  }
</style>
