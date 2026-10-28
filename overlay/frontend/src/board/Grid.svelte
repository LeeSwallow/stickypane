<script lang="ts">
  // The grid (03 §3.6): 12 columns, one below 720px. It is only placement:
  // given the notes in the board's order and the board's width, it puts
  // each where place() says and draws it with `item` (a NoteFrame by
  // default). A free canvas is another component with the same inputs.
  import type { Snippet } from 'svelte'
  import { columnsFor, place, type Placement } from './layout'
  import type { Note } from './model'
  import NoteFrame from './NoteFrame.svelte'

  let { notes, width, item }: { notes: Note[]; width: number; item?: Snippet<[Note, Placement]> } = $props()
  const placed = $derived(place(notes, width))
  const columns = $derived(columnsFor(width))
</script>

<div class="grid" style:--cols={columns}>
  {#each notes as note, i (note.name)}
    {@const p = placed[i]}
    <div class="slot" data-slot={note.name} style:grid-column="{p.column} / span {p.span}" style:grid-row={p.row}>
      {#if item}
        {@render item(note, p)}
      {:else}
        <NoteFrame {note} small={p.small} narrow={columns === 1} {width} />
      {/if}
    </div>
  {/each}
</div>

<style>
  .grid {
    display: grid;
    grid-template-columns: repeat(var(--cols), minmax(0, 1fr));
    gap: var(--space-6) var(--space-4);
    align-items: start;
  }
  .slot {
    min-width: 0;
  }
</style>
