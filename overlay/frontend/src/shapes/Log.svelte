<script lang="ts">
  // A log (03 §3.8, 10 §10.4): its lines in the mono face, following the
  // end as the file grows until the person scrolls up, and when it last
  // grew. A script's output and a request's responses are logs too.
  import { useBoard } from '../board/context'
  import type { Note } from '../board/model'
  import { language, relativeTime, tr } from '../i18n'
  import { dataOf, type LogData } from './data'
  import { follow } from './follow'
  import LogLines from './LogLines.svelte'

  let { note }: { note: Note } = $props()
  const board = useBoard()
  const lines = $derived(dataOf<LogData>(note.data, { lines: [] }).lines ?? [])
</script>

{#if lines.length}
  <div use:follow={lines.length}>
    <LogLines {lines} label={note.title} />
  </div>
  <p class="grew">{$tr('log.grew', { when: relativeTime(new Date(note.modTime), board.now(), $language) })}</p>
{:else}
  <p class="empty">{$tr('log.empty')}</p>
{/if}

<style>
  .grew,
  .empty {
    margin: var(--space-1) 0 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .empty {
    margin: 0;
  }
</style>
