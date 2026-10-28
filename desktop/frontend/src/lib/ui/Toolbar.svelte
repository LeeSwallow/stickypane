<script lang="ts">
  // A row of controls with one name; arrows move between them. Never tilted,
  // never on a rough line: toolbars are chrome.
  import type { Snippet } from 'svelte'
  import { roving } from './keys'

  let { label, gap = 1, children }: { label: string; gap?: 1 | 2; children: Snippet } = $props()
</script>

<div class="toolbar" role="toolbar" aria-label={label} style:gap="var(--space-{gap})" use:roving>
  {@render children()}
</div>

<style>
  .toolbar {
    display: flex;
    align-items: center;
    flex: none;
    --wails-draggable: no-drag;
  }
</style>
