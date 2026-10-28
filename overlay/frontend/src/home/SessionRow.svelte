<script lang="ts">
  // One project: a taped note with its name, path, who is on it and when it last moved.
  // The agents and the card summary come from the board, which the home does
  // not read; they show only when the row carries them.
  import { language, relativeTime, tr, type Lang } from '../i18n'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor, noteIndexOf } from '../lib/theme/tokens'
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
      <span class="dot" class:live={active} aria-hidden="true">{active ? '●' : '○'}</span>
      <span class="title">{name}</span>
      {#each row.Agents ?? [] as a (a)}
        <span class="agent" style:color={agentColor(a, notes)}>@{a}</span>
      {/each}
      {#each kinds as k (k)}
        <span class="kind">{k === 'tui' ? $tr('sessions.terminalBoard') : k}</span>
      {/each}
      <span class="when">{when}</span>
    </span>
    <span class="path">{row.Path}</span>
  </NoteFrame>
</button>

<style>
  .row {
    display: block;
    width: 100%;
    padding: 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
  }
  .top {
    display: flex;
    align-items: baseline;
    gap: 10px;
    flex-wrap: wrap;
  }
  .dot {
    color: var(--pencil);
  }
  .dot.live {
    color: var(--good);
  }
  .title {
    font-family: var(--font-hand);
    font-size: 22px;
    font-weight: 700;
  }
  .agent {
    font-family: var(--font-hand);
    font-size: 17px;
    font-weight: 700;
  }
  .kind {
    font-family: var(--font-hand);
    font-size: 16px;
    color: var(--pencil);
  }
  .when {
    margin-left: auto;
    font-family: var(--font-hand);
    font-size: 16px;
    color: var(--pencil);
  }
  .path {
    display: block;
    margin-top: 2px;
    font-family: var(--font-mono);
    font-size: 11.5px;
    color: var(--pencil);
    overflow-wrap: anywhere;
  }
</style>
