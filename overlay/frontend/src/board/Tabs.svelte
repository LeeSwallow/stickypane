<script lang="ts">
  // The tabs (folders, as on the terminal) and, below them, the closed
  // notes of the tab as chips: a click opens one (10 §10.1). A tab with a
  // changed note carries the `•`.
  import Underline from '../lib/doodle/Underline.svelte'
  import { tr } from '../i18n'
  import type { Note, Tab } from './model'

  let {
    tabs,
    current,
    changed,
    closed,
    onselect,
    onopen,
  }: {
    tabs: Tab[]
    current: string
    changed: (tab: Tab) => boolean
    closed: Note[]
    onselect: (tab: string) => void
    onopen: (note: string) => void
  } = $props()
  const label = (t: Tab) => t.title || t.name.replace(/^\d+[-_. ]*/, '') || $tr('board.tabRoot')
</script>

{#if tabs.length > 1}
  <div class="tabs" role="tablist">
    {#each tabs as t, i (t.name)}
      <button role="tab" aria-selected={t.name === current} title={i < 9 ? `⌘${i + 1}` : ''} onclick={() => onselect(t.name)}>
        {label(t)}
        {#if changed(t)}<span class="changed" title={$tr('board.changedTab')}>•</span>{/if}
        {#if t.name === current}<Underline />{/if}
      </button>
    {/each}
  </div>
{/if}
{#if closed.length}
  <div class="shelf" aria-label={$tr('board.shelf')}>
    {#each closed as n (n.name)}
      <button class="chip" title={$tr('note.open', { title: n.title })} onclick={() => onopen(n.name)}>{n.icon} {n.title}</button>
    {/each}
  </div>
{/if}

<style>
  .tabs {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-4);
    padding: var(--space-1) var(--space-5);
  }
  [role='tab'] {
    position: relative;
    border: 0;
    background: none;
    color: var(--pencil);
    font: 700 var(--hand-lg) var(--font-hand);
    cursor: pointer;
    padding: var(--space-1) 0;
  }
  [role='tab'][aria-selected='true'] {
    color: var(--ink);
  }
  .changed {
    color: var(--accent);
  }
  .shelf {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-5) 0;
  }
  .chip {
    padding: var(--space-1) var(--space-3);
    border: var(--stroke) dashed var(--grid-major);
    border-radius: var(--radius);
    background: var(--paper);
    color: var(--pencil);
    font: var(--text-sm) var(--font-body);
    cursor: pointer;
  }
</style>
