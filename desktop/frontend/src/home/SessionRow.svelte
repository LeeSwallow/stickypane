<script lang="ts">
  // One board as a card (08 §8.3): a strip of its palette's note colours,
  // its name, when it last moved, how many notes and how many stalled,
  // who is on it, what is running, and its folder. The whole card is the
  // button. What the board looks like comes from the last time this
  // computer opened it (the shell's peek); a board never opened here shows
  // its name, folder and time.
  import { language, relativeTime, tr, type Lang } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { agentColor, noteIndexOf } from '../lib/theme/tokens'
  import Badge from '../lib/ui/Badge.svelte'
  import Chip from '../lib/ui/Chip.svelte'
  import type { Peek, Row } from '../services/home'

  let {
    row,
    now,
    active = false,
    open,
    peek = null,
    day = '',
    home = '',
  }: {
    row: Row
    now: Date
    active?: boolean
    open: (path: string) => void
    peek?: Peek | null
    /** the day it was last used, said on a card of an earlier day */
    day?: string
    /** the person's home folder, written ~ in the path */
    home?: string
  } = $props()

  const notes = $derived(look.palette?.notes ?? [])
  // the board's own colours when this computer has seen it, else one from its folder
  const own = $derived(peek?.colors?.length === 6 ? peek.colors : null)
  const color = $derived(own ? own[noteIndexOf(row.Path) % 6] : (notes[noteIndexOf(row.Path) % Math.max(notes.length, 1)] ?? 'currentColor'))
  const name = $derived(row.Title || row.Path.split('/').filter(Boolean).pop() || row.Path)
  const path = $derived.by(() => {
    const p = home && row.Path.startsWith(home + '/') ? '~' + row.Path.slice(home.length) : row.Path
    // a long folder keeps its end, where its name is
    return p.length > 48 ? '…' + p.slice(-47) : p
  })
  const when = $derived.by(() => {
    const d = new Date(row.LastAt)
    const lang: Lang = $language
    if (Number.isNaN(d.getTime())) return ''
    if (active) return relativeTime(d, now, lang)
    const time = new Intl.DateTimeFormat(lang === 'ko' ? 'ko-KR' : 'en-US', { hour: '2-digit', minute: '2-digit', hour12: false }).format(d)
    return day ? `${day} ${time}` : time
  })
  const agents = $derived([...new Set([...(row.Agents ?? []), ...(peek?.agents ?? [])])].slice(0, 4))
  const kinds = $derived([...new Set((row.Live ?? []).map((s) => s.Kind))])
</script>

<button type="button" class="card" aria-label={$tr('sessions.open', { name })} onclick={() => open(row.Path)}>
  <NoteFrame {color} id={row.Path} lean>
    <span class="inner">
      <span class="top">
        <span class="dot" class:live={active} aria-hidden="true"></span>
        <span class="title" class:neat={look.style === 'neat'}>{name}</span>
      </span>
      <span class="facts">
        <span class="when">{when}</span>
        {#if peek}
          <span>· {$tr('home.notes', { n: peek.notes })}</span>
          {#if peek.stalled}<Badge tone="warn">{$tr('home.stalled', { n: peek.stalled })}</Badge>{/if}
        {:else}
          <span class="quiet">· {$tr('home.neverOpened')}</span>
        {/if}
        {#each kinds as k (k)}
          <Badge tone="accent">{k === 'tui' ? $tr('sessions.terminalBoard') : k}</Badge>
        {/each}
      </span>
      {#if agents.length}
        <span class="agents">
          {#each agents as a (a)}<Chip name={a} color={agentColor(a, notes)} />{/each}
        </span>
      {/if}
      <span class="foot">
        <span class="path" title={row.Path}>{path}</span>
        {#if own}
          <span class="swatches" aria-hidden="true">{#each own as c, i (i)}<span style:background={c}></span>{/each}</span>
        {/if}
      </span>
    </span>
  </NoteFrame>
</button>

<style>
  .card {
    display: block;
    width: 100%;
    min-width: 0;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
    transition: transform var(--t-short) var(--ease);
  }
  .card:hover {
    transform: translateY(-2px);
  }
  .card:hover :global(.note) {
    background: var(--hover);
  }
  .card:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
    border-radius: var(--radius-sm);
  }
  .card > :global(.note) {
    height: 100%;
    box-sizing: border-box;
  }
  .inner {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-height: 116px;
  }
  .top {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
  }
  .dot {
    width: 8px;
    height: 8px;
    flex: none;
    border-radius: 50%;
    border: var(--stroke-hairline) solid var(--ink-3);
  }
  .dot.live {
    border-color: var(--good);
    background: var(--good);
  }
  .title {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
    letter-spacing: var(--hand-track, 0);
  }
  .title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .when {
    color: var(--ink-2);
    font-variant-numeric: tabular-nums;
  }
  .facts,
  .agents {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1) var(--space-2);
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .quiet {
    color: var(--ink-3);
  }
  .foot {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-top: auto;
    min-width: 0;
  }
  .path {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ink-3);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .swatches {
    display: flex;
    flex: none;
    gap: 2px;
  }
  .swatches > span {
    width: 8px;
    height: 8px;
    border-radius: 2px;
    outline: 1px solid var(--line);
  }
</style>
