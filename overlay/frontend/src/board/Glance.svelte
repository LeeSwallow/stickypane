<script lang="ts">
  // The Glance line (03 §3.6): one Chip per @name — its card now and how
  // long it has been there, a warning when one of its cards stalled. A
  // click filters the board's cards by that name; again clears it. A Badge
  // at the end counts the stalled cards.
  import { language, relativeTime, tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor } from '../lib/theme/tokens'
  import Badge from '../lib/ui/Badge.svelte'
  import Chip from '../lib/ui/Chip.svelte'
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
    <span class="tags">
      {#each tags as g (g.who)}
        <span class="tag" title={$tr(filter === g.who ? 'glance.clear' : 'glance.filter', { who: g.who })}>
          <Chip
            name={g.who}
            color={color(g.who)}
            detail={g.card ? g.card.text : $tr('glance.resting')}
            time={g.since !== null ? relativeTime(new Date(now.getTime() - g.since), now, $language) : undefined}
            stalled={g.stalled}
            selected={filter === g.who}
            onclick={() => onfilter(filter === g.who ? null : g.who)}
          />
        </span>
      {/each}
    </span>
    {#if stalled}<Badge tone="warn">{$tr('glance.stalled', { n: stalled })}</Badge>{/if}
  </nav>
{/if}

<style>
  .glance {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    background: var(--chrome);
  }
  .tags {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    flex: 1;
    min-width: 0;
  }
  .tag {
    display: inline-flex;
    min-width: 0;
    max-width: 100%;
  }
</style>
