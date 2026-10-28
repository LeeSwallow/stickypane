<script lang="ts">
  // A crayon-filled card (a memo sheet): wash, hatching, outline, and a tilt of up to 2 degrees.
  import type { Snippet } from 'svelte'
  import { seedOf } from '../theme/tokens'
  import Crayon from './Crayon.svelte'
  import { look } from './look.svelte'

  let { color, id = 'sheet', children }: { color: string; id?: string; children?: Snippet } = $props()
  const seed = $derived(seedOf(id))
  const tilt = $derived(look.style === 'neat' ? 0 : ((seed % 9) - 4) * 0.45)
  const neat = $derived(look.style === 'neat')
</script>

<div class="sheet" class:neat style:transform="rotate({tilt}deg)" style:color="var(--card-ink)">
  <Crayon {color} {seed} />
  <div class="body">{@render children?.()}</div>
</div>

<style>
  .sheet {
    position: relative;
    padding: 12px 16px;
    font-family: var(--font-hand);
    font-weight: 700;
    font-size: 18px;
  }
  .sheet.neat {
    font-family: var(--font-body);
    font-weight: 600;
    font-size: 14px;
  }
  .body {
    position: relative;
  }
</style>
