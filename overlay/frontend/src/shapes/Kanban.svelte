<script lang="ts">
  // A board note (03 §3.8): columns with a hand-written title and a count,
  // cards as memo sheets. In a narrow window the columns fold to two.
  // Moving cards is A9's; this draws.
  import { noteHex, useBoard } from '../board/context'
  import type { KanbanData, Note } from '../board/model'
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor } from '../lib/theme/tokens'
  import Card from './Card.svelte'

  let { note, narrow = false }: { note: Note; narrow?: boolean } = $props()
  const board = useBoard()
  const data = $derived((note.data as KanbanData | null) ?? { desc: [], columns: [] })
  const now = $derived(board.now())
  const filter = $derived(board.filter())
  const colorOf = (who: string[]) =>
    who.length && look.palette ? agentColor(who[0], look.palette.notes) : noteHex(note.color, look.palette)
</script>

{#each data.desc as line, i (i)}
  <p class="desc">{line}</p>
{/each}
<div class="cols" class:narrow style:--n={data.columns.length}>
  {#each data.columns as col, ci (ci)}
    <section class="col" aria-label={col.title}>
      <h4 class="head">{col.title} <span class="count" title={$tr('kanban.count', { n: col.cards.length })}>{col.cards.length}</span></h4>
      {#each col.cards as card, i (i)}
        <Card
          {card}
          {now}
          color={colorOf(card.who)}
          id={`${note.name}:${ci}:${card.text}`}
          dim={filter !== null && !card.who.includes(filter)}
        />
      {:else}
        <p class="none">{$tr('kanban.empty')}</p>
      {/each}
    </section>
  {/each}
</div>

<style>
  .desc {
    margin: 0 0 6px;
    color: var(--pencil);
    font: 13px/1.5 var(--font-body);
  }
  .cols {
    display: grid;
    grid-template-columns: repeat(var(--n), minmax(0, 1fr));
    gap: 14px;
  }
  .cols.narrow {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .head {
    margin: 0 0 8px;
    color: var(--pencil);
    font: 700 17px var(--font-hand);
  }
  .count {
    font: 12px var(--font-mono);
    font-variant-numeric: tabular-nums;
    margin-left: 4px;
  }
  .none {
    color: var(--pencil);
    font: 13px var(--font-hand);
    margin: 0;
    opacity: 0.7;
  }
</style>
