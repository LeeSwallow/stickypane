<script lang="ts">
  // A ring drawn round its parent, to call attention (a stalled card).
  import { look } from './look.svelte'
  import { ellipsePaths } from './rough'
  import Strokes from './Strokes.svelte'

  let { stroke = 'var(--warn)', seed = 2 }: { stroke?: string; seed?: number } = $props()
  let w = $state(0)
  let h = $state(0)
  const segs = $derived(w ? ellipsePaths(w / 2, h / 2, w, h, { stroke, seed, strokeWidth: 2, roughness: 1.8, style: look.style }) : [])
</script>

<span class="c" bind:clientWidth={w} bind:clientHeight={h}><Strokes {segs} viewBox="0 0 {w} {h}" /></span>

<style>
  .c {
    position: absolute;
    inset: -10px -9px;
    pointer-events: none;
    z-index: 3;
  }
</style>
