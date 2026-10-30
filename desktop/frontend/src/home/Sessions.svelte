<script lang="ts">
  // The boards (08 §8.3): what is moving now, then today, then earlier, as
  // cards; a search narrows them by name and folder.
  import { dayLabel, language, tr } from '../i18n'
  import type { Groups, Peek, Row } from '../services/home'
  import SessionRow from './SessionRow.svelte'
  import { columnsFor, spans } from './grid'

  let {
    groups,
    now,
    open,
    peeks = {},
    query = '',
    home = '',
  }: {
    groups: Groups
    now: Date
    open: (path: string) => void
    peeks?: Record<string, Peek>
    query?: string
    home?: string
  } = $props()

  const pad = (n: number) => String(n).padStart(2, '0')
  const today = $derived(`${now.getFullYear()}-${pad(now.getMonth() + 1)}-${pad(now.getDate())}`)
  const words = $derived(query.toLowerCase().split(/\s+/).filter(Boolean))
  const match = (r: Row) => words.every((w) => `${r.Title} ${r.Path}`.toLowerCase().includes(w))

  const active = $derived((groups.Active ?? []).filter(match))
  const days = $derived((groups.Days ?? []).map((d) => ({ date: d.Date, rows: (d.Rows ?? []).filter(match) })))
  const todays = $derived(days.filter((d) => d.date === today).flatMap((d) => d.rows))
  const earlier = $derived(days.filter((d) => d.date !== today).flatMap((d) => d.rows.map((row) => ({ row, date: d.date }))))
  const none = $derived(!active.length && !todays.length && !earlier.length)
  // two or three to a row as the window allows; a short last row shares the width
  let width = $state(900)
  const cols = $derived(columnsFor(width))
  const span = (n: number, i: number) => spans(n, cols)[i]
</script>

<div class="sessions" bind:clientWidth={width}>
  {#if active.length}
    <section>
      <h2>{$tr('sessions.active')}</h2>
      <div class="cards">
        {#each active as row, i (row.ID)}
          <div class="cell" style:grid-column="span {span(active.length, i)}"><SessionRow {row} {now} active {open} peek={peeks[row.Path]} {home} /></div>
        {/each}
      </div>
    </section>
  {/if}
  {#if todays.length}
    <section>
      <h2>{$tr('home.today')}</h2>
      <div class="cards">
        {#each todays as row, i (row.ID)}
          <div class="cell" style:grid-column="span {span(todays.length, i)}"><SessionRow {row} {now} {open} peek={peeks[row.Path]} {home} /></div>
        {/each}
      </div>
    </section>
  {/if}
  {#if earlier.length}
    <section>
      <h2>{$tr('home.earlier')}</h2>
      <div class="cards">
        {#each earlier as e, i (e.row.ID)}
          <div class="cell" style:grid-column="span {span(earlier.length, i)}"><SessionRow row={e.row} {now} {open} peek={peeks[e.row.Path]} day={dayLabel(e.date, now, $language)} {home} /></div>
        {/each}
      </div>
    </section>
  {/if}
  {#if none && words.length}
    <p class="none" role="status">{$tr('home.noMatch', { q: query })}</p>
  {/if}
</div>

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
  .cell > :global(.card) {
    flex: 1;
  }
  .none {
    margin: var(--space-8) 0;
    text-align: center;
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
</style>
