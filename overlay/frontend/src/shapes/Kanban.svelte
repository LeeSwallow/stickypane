<script lang="ts">
  // A board note (03 §3.8): columns with a hand-written title and a count,
  // cards as memo sheets. In a narrow window the columns fold to two.
  // A card is dragged to another place or column, or moved with h l j k
  // when it has the focus, as on the terminal: one note.act `move`, drawn
  // at once (board/interact.ts) and put back if the bridge refuses.
  import { noteHex, useBoard } from '../board/context'
  import type { KanbanData, Note } from '../board/model'
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor } from '../lib/theme/tokens'
  import { dropAt, keyMove } from '../board/interact'
  import AddLine from './AddLine.svelte'
  import Card from './Card.svelte'

  let { note, narrow = false }: { note: Note; narrow?: boolean } = $props()
  const board = useBoard()
  const data = $derived((note.data as KanbanData | null) ?? { desc: [], columns: [] })
  const now = $derived(board.now())
  const filter = $derived(board.filter())
  const colorOf = (who: string[]) =>
    who.length && look.palette ? agentColor(who[0], look.palette.notes) : noteHex(note.color, look.palette)

  // the card being dragged, and where it would land
  let held = $state<{ col: number; i: number } | null>(null)
  let over = $state<{ col: number; before: number | null } | null>(null)

  function drop(to: number, before: number | null) {
    const h = held
    held = over = null
    if (!h) return
    const a = dropAt(h.col, h.i, to, before, data.columns[to]?.cards.length ?? 0)
    if (a) void board.act(note, a)
  }

  function key(e: KeyboardEvent, col: number, i: number) {
    if (e.metaKey || e.ctrlKey || e.altKey) return
    const a = keyMove(data, col, i, e.key)
    if (!a) return
    e.preventDefault()
    e.stopPropagation()
    const to = Number(a.args?.to)
    const at = a.args?.at !== undefined ? Number(a.args.at) : (data.columns[to]?.cards.length ?? 1) - (to === col ? 1 : 0)
    void board.act(note, a).then(() =>
      requestAnimationFrame(() => document.querySelector<HTMLElement>(`[data-move="${CSS.escape(`${note.name}:${to}:${at}`)}"]`)?.focus()),
    )
  }
</script>

{#each data.desc as line, i (i)}
  <p class="desc">{line}</p>
{/each}
<div class="cols" class:narrow style:--n={data.columns.length}>
  {#each data.columns as col, ci (ci)}
    <section
      class="col"
      class:over={over?.col === ci}
      aria-label={col.title}
      ondragover={(e) => {
        if (!held) return
        e.preventDefault()
        if (over?.col !== ci) over = { col: ci, before: null }
      }}
      ondragleave={(e) => {
        if (!(e.currentTarget as HTMLElement).contains(e.relatedTarget as Node)) over = null
      }}
      ondrop={(e) => {
        e.preventDefault()
        drop(ci, over?.col === ci ? over.before : null)
      }}
    >
      <h4 class="head">{col.title} <span class="count" title={$tr('kanban.count', { n: col.cards.length })}>{col.cards.length}</span></h4>
      {#each col.cards as card, i (i)}
        <!-- svelte-ignore a11y_no_noninteractive_tabindex -->
        <!-- svelte-ignore a11y_no_noninteractive_element_interactions -->
        <div
          class="move"
          class:held={held?.col === ci && held.i === i}
          class:before={over?.col === ci && over.before === i}
          draggable="true"
          tabindex="0"
          role="listitem"
          aria-roledescription={$tr('kanban.card')}
          aria-keyshortcuts="h l j k"
          title={$tr('kanban.moveHint')}
          data-move={`${note.name}:${ci}:${i}`}
          ondragstart={(e) => {
            held = { col: ci, i }
            e.dataTransfer?.setData('text/plain', card.text)
            if (e.dataTransfer) e.dataTransfer.effectAllowed = 'move'
          }}
          ondragend={() => (held = over = null)}
          ondragover={(e) => {
            if (!held) return
            e.preventDefault()
            e.stopPropagation()
            const r = (e.currentTarget as HTMLElement).getBoundingClientRect()
            const before = e.clientY < r.top + r.height / 2 ? i : i + 1 < col.cards.length ? i + 1 : null
            over = { col: ci, before }
          }}
          onkeydown={(e) => key(e, ci, i)}
        >
          <Card
            {card}
            {now}
            color={colorOf(card.who)}
            id={`${note.name}:${ci}:${card.text}`}
            dim={filter !== null && !card.who.includes(filter)}
          />
        </div>
      {:else}
        <p class="none">{$tr('kanban.empty')}</p>
      {/each}
      <AddLine label={$tr('kanban.add')} onadd={(text) => board.act(note, { name: 'add', args: { col: String(ci), text } })} />
    </section>
  {/each}
</div>

<style>
  .desc {
    margin: 0 0 var(--space-2);
    color: var(--pencil);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .cols {
    display: grid;
    grid-template-columns: repeat(var(--n), minmax(0, 1fr));
    gap: var(--space-4);
  }
  .cols.narrow {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .head {
    margin: 0 0 var(--space-2);
    color: var(--pencil);
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
  }
  .count {
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    font-variant-numeric: tabular-nums;
    margin-left: var(--space-1);
  }
  .col {
    border-radius: var(--radius-sm);
    transition: background var(--t-short) var(--ease);
  }
  .col.over {
    background: var(--grid-major);
  }
  .move {
    border-radius: var(--radius-sm);
    cursor: grab;
  }
  .move:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .move.held {
    opacity: 0.4;
  }
  .move.before {
    box-shadow: 0 calc(-1 * var(--space-1)) 0 0 var(--focus);
  }
  .none {
    color: var(--pencil);
    font: var(--weight-hand-label) var(--text-body-md) / var(--leading-body-md) var(--font-hand);
    margin: 0;
    opacity: 0.7;
  }
</style>
