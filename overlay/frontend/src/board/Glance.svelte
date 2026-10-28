<script lang="ts">
  // The Glance line (03 §3.6): one tag per @name, its card now and how long
  // it has been there, a warning border when one of its cards stalled. A
  // click filters the board's cards by that name; again clears it.
  import { language, relativeTime, tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor } from '../lib/theme/tokens'
  import type { GlanceTag } from './glance'

  let {
    tags,
    stalled,
    filter,
    now,
    onfilter,
  }: { tags: GlanceTag[]; stalled: number; filter: string | null; now: Date; onfilter: (who: string | null) => void } = $props()
  const color = (who: string) => (look.palette ? agentColor(who, look.palette.notes) : 'var(--ink)')
</script>

{#if tags.length}
  <nav class="glance" aria-label={$tr('glance.label')}>
    {#each tags as g (g.who)}
      <button
        class="tag"
        class:stalled={g.stalled}
        aria-pressed={filter === g.who}
        title={$tr(filter === g.who ? 'glance.clear' : 'glance.filter', { who: g.who })}
        onclick={() => onfilter(filter === g.who ? null : g.who)}
      >
        <span class="who" style:color={color(g.who)}>@{g.who}</span>
        <span class="card">{g.card ? g.card.text : $tr('glance.resting')}</span>
        {#if g.since !== null}<span class="since">{relativeTime(new Date(now.getTime() - g.since), now, $language)}</span>{/if}
      </button>
    {/each}
    {#if stalled}<span class="count">{$tr('glance.stalled', { n: stalled })}</span>{/if}
  </nav>
{/if}

<style>
  .glance {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-1) var(--space-5);
  }
  .tag {
    display: inline-flex;
    align-items: baseline;
    gap: var(--space-2);
    max-width: 22em;
    padding: var(--space-1) var(--space-3);
    border: var(--stroke) solid var(--grid-major);
    border-radius: 999px;
    background: var(--paper);
    color: var(--ink);
    cursor: pointer;
  }
  .tag.stalled {
    border-color: var(--warn);
  }
  .tag[aria-pressed='true'] {
    border-color: var(--accent);
  }
  .who {
    font: 700 var(--hand-md) var(--font-hand);
  }
  .card {
    font: var(--text-sm) var(--font-body);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  .since,
  .count {
    color: var(--pencil);
    font: var(--text-xs) var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .count {
    color: var(--warn);
    margin-left: auto;
  }
</style>
