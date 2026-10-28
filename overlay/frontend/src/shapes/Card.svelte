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
    margin-bottom: 12px;
  }
  .dim {
    opacity: 0.35;
  }
  .text {
    font-size: 16px;
    line-height: 1.25;
    overflow-wrap: anywhere;
  }
  .meta {
    display: flex;
    gap: 6px;
    justify-content: space-between;
    margin-top: 4px;
    font-size: 13px;
    opacity: 0.85;
  }
  .ago {
    font: 11px var(--font-mono);
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
    margin-top: 4px;
    color: var(--warn);
    font: 700 13px var(--font-hand);
  }
</style>
