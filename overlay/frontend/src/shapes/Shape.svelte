<script lang="ts">
  // A note's body, drawn by its shape. It needs only the note and the board
  // context, so it draws the same in the grid, zoomed, on a canvas or in a
  // torn-off window. Shapes not drawn yet (A8) show their summary.
  import type { Note as NoteT } from '../board/model'
  import { tr } from '../i18n'
  import Checklist from './Checklist.svelte'
  import Kanban from './Kanban.svelte'
  import Note from './Note.svelte'

  let { note, depth = 0, narrow = false }: { note: NoteT; depth?: number; narrow?: boolean } = $props()
</script>

{#if note.error}
  <p class="unreadable" role="alert">{$tr('note.unreadable', { error: note.error })}</p>
{:else if note.kind === 'note'}
  <Note {note} {depth} />
{:else if note.kind === 'checklist'}
  <Checklist {note} />
{:else if note.kind === 'board'}
  <Kanban {note} {narrow} />
{:else}
  <p class="plain">{$tr('note.plain', { summary: note.summary || note.kind })}</p>
{/if}

<style>
  .unreadable {
    margin: 0;
    color: var(--bad);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .plain {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
</style>
