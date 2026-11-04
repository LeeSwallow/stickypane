<script lang="ts">
  // 원격: the boards of the remote hosts (settings → 원격 호스트), as cards
  // like the boards of this computer: name, host and folder, and what the
  // board looked like when this computer last opened it. They are not in
  // this computer's index, so no time or session is said.
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { folderColor } from '../lib/theme/tokens'
  import { baseName, shortPath } from '../lib/path'
  import Badge from '../lib/ui/Badge.svelte'
  import type { Peek } from '../services/home'
  import { columnsFor, spans } from './grid'

  let {
    boards,
    open,
    peeks = {},
    query = '',
  }: {
    boards: { host: string; path: string }[]
    open: (host: string, path: string) => void
    peeks?: Record<string, Peek>
    query?: string
  } = $props()

  const words = $derived(query.toLowerCase().split(/\s+/).filter(Boolean))
  const shown = $derived(boards.filter((b) => words.every((w) => `${b.host} ${b.path}`.toLowerCase().includes(w))))
  const notes = $derived(look.palette?.notes ?? [])
  const key = (b: { host: string; path: string }) => `ssh://${b.host}${b.path}`
  const colorOf = (b: { host: string; path: string }) => folderColor(key(b), peeks[key(b)]?.colors, notes)
  let width = $state(900)
  const cols = $derived(columnsFor(width))
</script>

{#if shown.length}
  <section class="remote" bind:clientWidth={width} aria-label={$tr('home.remote')}>
    <h2>{$tr('home.remote')}</h2>
    <div class="cards">
      {#each shown as b, i (key(b))}
        {@const peek = peeks[key(b)]}
        <div class="cell" style:grid-column="span {spans(shown.length, cols)[i]}">
          <button type="button" class="card" aria-label={$tr('sessions.open', { name: `${baseName(b.path)} · ${b.host}` })} onclick={() => open(b.host, b.path)}>
            <NoteFrame color={colorOf(b)} id={key(b)} lean>
              <span class="inner">
                <span class="top">
                  <span class="title" class:neat={look.style === 'neat'}>{baseName(b.path)}</span>
                </span>
                <span class="facts">
                  <Badge tone="accent">{b.host}</Badge>
                  {#if peek}<span>{$tr('home.notes', { n: peek.notes })}</span>{/if}
                </span>
                <span class="path" title="{b.host}:{b.path}">{shortPath(b.path)}</span>
              </span>
            </NoteFrame>
          </button>
        </div>
      {/each}
    </div>
  </section>
{/if}

<style>
  section {
    margin-bottom: var(--space-8);
  }
  h2 {
    margin: 0 0 var(--space-3);
    color: var(--ink-2);
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
    letter-spacing: var(--hand-track, 0);
  }
  .cards {
    display: grid;
    grid-template-columns: repeat(6, minmax(0, 1fr));
    gap: var(--space-6) var(--space-5);
  }
  .cell {
    display: flex;
    min-width: 0;
  }
  .card {
    display: block;
    flex: 1;
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
  }
  .top {
    display: flex;
    align-items: center;
    min-width: 0;
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
  .facts {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1) var(--space-2);
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .path {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ink-3);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
</style>
