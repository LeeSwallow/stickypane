<script lang="ts">
  // A keyboard shortcut, in the mono face: "⌥Space", "⌘K".
  let { keys, tone = 'chrome' }: { keys: string; tone?: 'chrome' | 'tip' } = $props()
</script>

<kbd class={tone}>{keys}</kbd>

<style>
  kbd {
    display: inline-block;
    padding: 0 var(--space-1);
    border-radius: var(--radius-sm);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    font-variant-numeric: tabular-nums;
    white-space: nowrap;
  }
  .chrome {
    color: var(--ink-2);
    background: var(--surface-sunk);
    box-shadow: inset 0 0 0 1px var(--line);
  }
  .tip {
    color: inherit;
    opacity: 0.75;
    padding: 0;
  }
</style>
