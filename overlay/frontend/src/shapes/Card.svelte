<script lang="ts">
  // One card on a board: a small memo sheet in its owner's colour, with
  // @owner and how long since it moved. A stalled card gets a warning ring,
  // its time in words, and says so to a screen reader: never colour alone.
  import type { Card } from '../board/model'
  import { language, relativeTime, tr } from '../i18n'
  import Circle from '../lib/doodle/Circle.svelte'
  import Sheet from '../lib/doodle/Sheet.svelte'
  import { seedOf } from '../lib/theme/tokens'

  let { card, color, id, now, dim = false }: { card: Card; color: string; id: string; now: Date; dim?: boolean } = $props()
  const moved = $derived(card.moved ? new Date(card.moved) : null)
  const ago = $derived(moved && Number.isFinite(moved.getTime()) ? relativeTime(moved, now, $language) : '')
</script>

<div class="card" class:dim data-stalled={card.stalled} data-card={card.text}>
  <Sheet {color} {id}>
    <div class="text">{card.text}</div>
    {#if card.who.length || ago}
      <div class="meta">
        {#if card.who.length}<span class="who">@{card.who.join(' @')}</span>{/if}
        {#if ago}<span class="ago">{ago}</span>{/if}
      </div>
    {/if}
  </Sheet>
  {#if card.stalled}
    <span class="ring"><Circle stroke="var(--warn)" seed={seedOf(id)} /></span>
    <span class="stalled">{ago ? $tr('badge.stale', { time: ago }) : $tr('card.stalled')}</span>
  {/if}
</div>

<style>
  .card {
    position: relative;
    margin-bottom: var(--space-3);
  }
  .dim {
    opacity: 0.35;
  }
  .text {
    font-size: var(--text-body-lg);
    line-height: 1.25;
    overflow-wrap: anywhere;
  }
  .meta {
    display: flex;
    gap: var(--space-2);
    justify-content: space-between;
    margin-top: var(--space-1);
    font-size: var(--text-body-md);
    opacity: 0.85;
  }
  .ago {
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    font-variant-numeric: tabular-nums;
    align-self: center;
  }
  .ring {
    position: absolute;
    inset: 0;
    pointer-events: none;
  }
  .stalled {
    display: block;
    margin-top: var(--space-1);
    color: var(--warn);
    font: var(--weight-hand-label) var(--text-body-md) / var(--leading-body-md) var(--font-hand);
  }
</style>
