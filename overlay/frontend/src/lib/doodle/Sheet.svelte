<script lang="ts">
  // A crayon-filled card (a memo sheet): wash, hatching, outline, and a tilt of at most MAX_TILT (1.5°).
  import type { Snippet } from 'svelte'
  import { tiltOf } from '../theme/scale'
  import { seedOf } from '../theme/tokens'
  import Crayon from './Crayon.svelte'
  import { look } from './look.svelte'

  let { color, id = 'sheet', children }: { color: string; id?: string; children?: Snippet } = $props()
  const seed = $derived(seedOf(id))
  const tilt = $derived(tiltOf(seed, look.style))
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
