<script lang="ts">
  // A count (mono, in a small pill) or a status (a dot and a word). Colour
  // is never the only signal: a status always has its word.
  import type { Snippet } from 'svelte'

  let {
    count,
    tone = 'neutral',
    children,
  }: {
    count?: number
    tone?: 'neutral' | 'accent' | 'good' | 'warn' | 'bad'
    children?: Snippet
  } = $props()
</script>

{#if children}
  <span class="status {tone}"><i aria-hidden="true"></i>{@render children()}</span>
{:else}
  <span class="count {tone}">{count}</span>
{/if}

<style>
  .count {
    display: inline-grid;
    place-items: center;
    min-width: 18px;
    height: 18px;
    padding: 0 5px;
    box-sizing: border-box;
    border-radius: var(--radius-pill);
    background: var(--surface-sunk);
    color: var(--ink-2);
    box-shadow: inset 0 0 0 1px var(--line);
    font: 500 var(--text-mono-sm) / 1 var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .status {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    height: 20px;
    padding: 0 var(--space-2);
    border-radius: var(--radius-pill);
    font: 600 var(--text-body-xs) / 1 var(--font-body);
    white-space: nowrap;
    background: var(--surface-sunk);
    color: var(--ink-2);
  }
  .status i {
    width: 6px;
    height: 6px;
    border-radius: 50%;
    background: currentColor;
  }
  .count.accent,
  .status.accent {
    background: var(--accent-wash);
    color: var(--accent-text);
    box-shadow: none;
  }
  .count.good,
  .status.good {
    background: var(--good-wash);
    color: var(--good-text);
    box-shadow: none;
  }
  .count.warn,
  .status.warn {
    background: var(--warn-wash);
    color: var(--warn-text);
    box-shadow: none;
  }
  .count.bad,
  .status.bad {
    background: var(--danger-wash);
    color: var(--danger-text);
    box-shadow: none;
  }
</style>
