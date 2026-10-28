<script lang="ts">
  // Fills its parent the way the icon is drawn: a light wash, crayon hatching, a thick outline.
  import { cardColorsFor } from '../theme/tokens'
  import { look } from './look.svelte'
  import { crayonPaths } from './rough'
  import Strokes from './Strokes.svelte'

  let { color, seed = 1 }: { color: string; seed?: number } = $props()
  let w = $state(0)
  let h = $state(0)
  const segs = $derived.by(() => {
    if (!w || !look.palette) return []
    const c = cardColorsFor(color, look.palette, look.dark)
    return crayonPaths(w, h, { ...c, style: look.style, seed })
  })
</script>

<div class="fill" bind:clientWidth={w} bind:clientHeight={h}>
  <Strokes {segs} viewBox="0 0 {w} {h}" />
</div>

<style>
  .fill {
    position: absolute;
    inset: 0;
    pointer-events: none;
  }
</style>
