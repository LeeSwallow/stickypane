<script lang="ts">
  // The head of a note: what comes before the title (a way back, the kind's
  // icon), the title (hand-written, in the note's colour, one line; a
  // double-click renames it), a changed dot, a link mark, the summary
  // (3/6, 6 cards), a book's pager, and the note's controls — its width,
  // written as a fraction ("1/2", Full) with a tooltip in words, stepped by
  // buttons named for the width they go to, pin, tear-off (where
  // the window can tear it off), more ⋯ and close. It holds controls, so it
  // never leans; the controls stay quiet until the note is hovered.
  import type { Snippet } from 'svelte'
  import { tr } from '../../i18n'
  import { look } from '../doodle/look.svelte'
  import { FRACTIONS, isFraction, nextFraction, type Fraction } from '../sizes'
  import { noteTitleColor } from '../theme/tokens'
  import IconButton from './IconButton.svelte'
  import Icon from './icons/Icon.svelte'
  import Menu from './Menu.svelte'
  import type { MenuItem } from './menu'
  import Toolbar from './Toolbar.svelte'
  import Tooltip from './Tooltip.svelte'

  let {
    title,
    color,
    summary,
    updated = false,
    size = '1/2',
    pinned = false,
    book,
    linked,
    editable = true,
    sizable = true,
    lead,
    menu = [],
    menuOpen = $bindable(false),
    onsize,
    onpin,
    ontearoff,
    onrename,
    onmenu,
    onclose,
  }: {
    title: string
    color: string
    summary?: string
    updated?: boolean
    /** the fraction of the board the note is at */
    size?: string
    pinned?: boolean
    /** a book: its page (1-based), its count, and the names of its pages for a list */
    book?: { page: number; total: number; onprev: () => void; onnext: () => void; names?: string[]; onpage?: (i: number) => void }
    /** the real path of a linked file */
    linked?: string
    /** false: a note that cannot be read; it can only be closed */
    editable?: boolean
    /** false: something else sizes the note (the canvas); no width steps */
    sizable?: boolean
    lead?: Snippet
    menu?: MenuItem[]
    menuOpen?: boolean
    onsize?: (s: Fraction) => void
    onpin?: (on: boolean) => void
    ontearoff?: () => void
    onrename?: (title: string) => void
    onmenu?: (id: string) => void
    onclose?: () => void
  } = $props()

  const ink = $derived(look.palette ? noteTitleColor(color, look.palette, look.dark) : color)
  const at = $derived(FRACTIONS.indexOf(size as Fraction))
  const step = (dir: -1 | 1) => onsize?.(nextFraction(size, dir))
  // the width as it is read: "1/2", or "Full" for the whole board; and in
  // words for the tooltip and the ear: "1/2 of the board"
  const fraction = (f: string) => (f === '1' || !isFraction(f) ? $tr('ui.size.full') : f)
  const words = (f: string) => (f === '1' || !isFraction(f) ? $tr('ui.size.whole') : $tr('ui.size.share', { size: f }))

  const renamable = $derived(editable && !!onrename)
  let editing = $state(false)
  let draft = $state('')
  function rename() {
    if (!editable || !onrename) return
    draft = title
    editing = true
  }
  function commit() {
    if (!editing) return
    editing = false
    const t = draft.trim()
    if (t && t !== title) onrename?.(t)
  }
</script>

<div class="wrap">
  <header class="head">
    {@render lead?.()}
    {#if editing}
      <!-- svelte-ignore a11y_autofocus -->
      <input
        class="title edit"
        class:neat={look.style === 'neat'}
        aria-label={$tr('note.rename')}
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
      <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
      <!-- svelte-ignore a11y_no_noninteractive_tabindex: a renamable title takes the focus so F2 or Enter renames it -->
      <h3
        class="title"
        class:neat={look.style === 'neat'}
        style:color={ink}
        title={renamable ? `${title} · ${$tr('note.rename')}` : title}
        tabindex={renamable ? 0 : undefined}
        ondblclick={rename}
        onkeydown={(e) => {
          if (!renamable || (e.key !== 'F2' && e.key !== 'Enter')) return
          // the title's own key: the board must not zoom the note too
          e.preventDefault()
          e.stopPropagation()
          rename()
        }}
      >
        {title}
      </h3>
    {/if}
    {#if updated}<span class="dot" style:background={color} role="img" aria-label={$tr('ui.updated')}></span>{/if}
    {#if linked}
      <Tooltip text={$tr('note.linked', { target: linked })} mono>
        <span class="link" role="img" aria-label={$tr('note.linked', { target: linked })}><Icon name="link" size={14} /></span>
      </Tooltip>
    {/if}
    {#if summary}<span class="summary">{summary}</span>{/if}
    <span class="grow"></span>
    {#if book && book.total > 1}
      <span class="pager">
        <IconButton icon="book-prev" size="sm" label={$tr('ui.page.prev')} shortcut="[" disabled={book.page <= 1} onclick={book.onprev} />
        {#if book.names && book.onpage}
          <select class="page" aria-label={$tr('note.pages')} value={book.page - 1} onchange={(e) => book.onpage?.(Number(e.currentTarget.value))}>
            {#each book.names as n, i (i)}<option value={i}>{i + 1} / {book.total} · {n}</option>{/each}
          </select>
        {:else}
          <span class="page" aria-label={$tr('ui.page.of', { n: book.page, total: book.total })}>{book.page} / {book.total}</span>
        {/if}
        <IconButton icon="book-next" size="sm" label={$tr('ui.page.next')} shortcut="]" disabled={book.page >= book.total} onclick={book.onnext} />
      </span>
    {/if}
    <span class="tools">
      <Toolbar label={$tr('ui.noteActions')}>
        {#if editable && sizable}
          <span class="size" role="group" aria-label={$tr('ui.size')}>
            <span class="step">
              <IconButton
                icon="narrower"
                size="sm"
                label={at === 0 ? $tr('ui.size.narrower') : $tr('ui.size.narrowerTo', { size: words(nextFraction(size, -1)) })}
                shortcut="-"
                disabled={at === 0}
                onclick={() => step(-1)}
              />
            </span>
            <Tooltip text={$tr('ui.size.now', { size: words(size) })} describe={false}>
              <span class="frac" role="status" aria-label={$tr('ui.size.now', { size: words(size) })}>{fraction(size)}</span>
            </Tooltip>
            <span class="step">
              <IconButton
                icon="wider"
                size="sm"
                label={at === FRACTIONS.length - 1 ? $tr('ui.size.wider') : $tr('ui.size.widerTo', { size: words(nextFraction(size, 1)) })}
                shortcut="="
                disabled={at === FRACTIONS.length - 1}
                onclick={() => step(1)}
              />
            </span>
          </span>
        {/if}
        {#if editable}
          <IconButton
            icon="pin"
            size="sm"
            label={pinned ? $tr('note.unpin') : $tr('note.pin')}
            pressed={pinned}
            crayon={pinned}
            onclick={() => onpin?.(!pinned)}
          />
          {#if ontearoff}
            <span class="tearoff"><IconButton icon="tear-off" size="sm" label={$tr('ui.tearOff')} onclick={ontearoff} /></span>
          {/if}
        {/if}
        {#if menu.length}
          <span class="anchor">
            <IconButton icon="more" size="sm" label={$tr('ui.more')} aria-haspopup="menu" aria-expanded={menuOpen} align="end" onclick={() => (menuOpen = !menuOpen)} />
            <Menu items={menu} bind:open={menuOpen} label={$tr('ui.noteActions')} onselect={onmenu} />
          </span>
        {/if}
        {#if onclose}
          <IconButton icon="close" size="sm" label={$tr('ui.close')} align="end" onclick={onclose} />
        {/if}
      </Toolbar>
    </span>
  </header>
</div>

<style>
  .wrap {
    container-type: inline-size;
    min-width: 0;
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
    min-width: min(6em, 40%);
    margin: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  .title:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
    border-radius: var(--radius-sm);
  }
  .title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .title.edit {
    flex: 1;
    min-width: 0;
    padding: 0 var(--space-1);
    color: var(--ink);
    background: var(--surface);
    border: var(--stroke-hairline) solid var(--focus);
    border-radius: var(--radius-sm);
    outline: none;
  }
  .dot {
    width: 6px;
    height: 6px;
    flex: none;
    border-radius: 50%;
  }
  .link {
    display: inline-flex;
    color: var(--ink-3);
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
    max-width: 12em;
    text-align: center;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  select.page {
    height: var(--control-sm);
    padding: 0 var(--space-1);
    border: var(--stroke-hairline) solid var(--line);
    border-radius: var(--radius-sm);
    background: var(--surface);
  }
  /* the width reads as a control, not as another count beside the summary */
  .size {
    display: inline-flex;
    align-items: center;
    margin-right: var(--space-1);
    border: var(--stroke-hairline) solid var(--line);
    border-radius: var(--radius-md);
  }
  .step,
  .tearoff {
    display: inline-flex;
  }
  .frac {
    display: inline-flex;
    align-items: center;
    gap: var(--space-1);
    min-width: 32px;
    padding: 0 var(--space-1);
    justify-content: center;
    color: var(--ink);
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
  /* a narrow note keeps its width, pin, ⋯ and close; the rest is a key or an edge away */
  @container (max-width: 400px) {
    .summary,
    .tearoff {
      display: none;
    }
  }
  /* a very narrow note gives its title the line; its controls come back
     when it is pointed at or holds the focus */
  @container (max-width: 320px) {
    .head:not(:hover):not(:focus-within) .tools {
      display: none;
    }
  }
  @container (max-width: 300px) {
    .step {
      display: none;
    }
    .frac {
      padding: 0 var(--space-1);
    }
  }
</style>
