<script lang="ts">
  // The list: what is moving now, then today, yesterday and the earlier days.
  import { dayLabel, language, tr } from '../i18n'
  import type { Groups } from './api'
  import SessionRow from './SessionRow.svelte'

  let { groups, now, open }: { groups: Groups; now: Date; open: (path: string) => void } = $props()
  const active = $derived(groups.Active ?? [])
  const days = $derived((groups.Days ?? []).filter((d) => (d.Rows ?? []).length > 0))
</script>

<div class="sessions">
  {#if active.length}
    <section>
      <h2>{$tr('sessions.active')}</h2>
      {#each active as row (row.ID)}
        <SessionRow {row} {now} active {open} />
      {/each}
    </section>
  {/if}
  {#each days as day (day.Date)}
    <section>
      <h2>{dayLabel(day.Date, now, $language)}</h2>
      {#each day.Rows ?? [] as row (row.ID)}
        <SessionRow {row} {now} {open} />
      {/each}
    </section>
  {/each}
</div>

<style>
  section {
    display: flex;
    flex-direction: column;
    gap: 12px;
    margin-bottom: 22px;
  }
  h2 {
    margin: 0;
    font-family: var(--font-hand);
    font-size: 20px;
    font-weight: 700;
    color: var(--pencil);
  }
</style>
