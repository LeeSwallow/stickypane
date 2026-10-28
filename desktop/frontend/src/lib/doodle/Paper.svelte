<script lang="ts">
  // Graph paper: 18px fine lines and a heavier line every 90px (03 §3.4).
  import type { Snippet } from 'svelte'
  let { children }: { children?: Snippet } = $props()
</script>

<div class="paper">{@render children?.()}</div>

<style>
  .paper {
    min-height: 100%;
    background-color: var(--paper);
    background-image:
      linear-gradient(var(--grid-major) 1px, transparent 1px),
      linear-gradient(90deg, var(--grid-major) 1px, transparent 1px),
      linear-gradient(var(--grid) 1px, transparent 1px),
      linear-gradient(90deg, var(--grid) 1px, transparent 1px);
    background-size: 90px 90px, 90px 90px, 18px 18px, 18px 18px;
    color: var(--ink);
  }
</style>
