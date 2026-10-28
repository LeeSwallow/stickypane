<script lang="ts">
  // One project: a taped note (it holds no controls of its own, the whole
  // note is the button) with its name, who is on it as Chips, what is
  // running as Badges, when it last moved, and its path on one line.
  // The agents come from the board, which the home does not read; they
  // show only when the row carries them.
  import { language, relativeTime, tr, type Lang } from '../i18n'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor, noteIndexOf } from '../lib/theme/tokens'
  import Badge from '../lib/ui/Badge.svelte'
  import Chip from '../lib/ui/Chip.svelte'
  import type { Row } from './api'

  let { row, now, active = false, open }: { row: Row; now: Date; active?: boolean; open: (path: string) => void } = $props()

  const notes = $derived(look.palette?.notes ?? [])
  const color = $derived(notes.length ? notes[noteIndexOf(row.Path) % notes.length] : 'currentColor')
  const name = $derived(row.Title || row.Path.split('/').filter(Boolean).pop() || row.Path)
  const when = $derived.by(() => {
    const d = new Date(row.LastAt)
    const lang: Lang = $language
    if (Number.isNaN(d.getTime())) return ''
    const time = active
      ? relativeTime(d, now, lang)
      : new Intl.DateTimeFormat(lang === 'ko' ? 'ko-KR' : 'en-US', { hour: '2-digit', minute: '2-digit', hour12: false }).format(d)
    return active ? time : $tr('sessions.last', { time })
  })
  const kinds = $derived([...new Set((row.Live ?? []).map((s) => s.Kind))])
</script>

<button type="button" class="row" aria-label={$tr('sessions.open', { name })} onclick={() => open(row.Path)}>
  <NoteFrame {color} id={row.Path}>
    <span class="top">
      <span class="dot" class:live={active} aria-hidden="true"></span>
      <span class="title" class:neat={look.style === 'neat'}>{name}</span>
      {#each row.Agents ?? [] as a (a)}
        <Chip name={a} color={agentColor(a, notes)} />
      {/each}
      {#each kinds as k (k)}
        <Badge tone="accent">{k === 'tui' ? $tr('sessions.terminalBoard') : k}</Badge>
      {/each}
      <span class="when">{when}</span>
    </span>
    <span class="path" title={row.Path}>{row.Path}</span>
  </NoteFrame>
</button>

<style>
  .row {
    display: block;
    width: 100%;
    padding: 0;
    border: 0;
    border-radius: var(--radius-sm);
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .row:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .row:hover :global(.note) {
    background: var(--hover);
  }
  .top {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    flex-wrap: wrap;
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
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  .title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .when {
    margin-left: auto;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .path {
    display: block;
    margin-top: var(--space-1);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ink-3);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
</style>
