<script lang="ts">
  // A hand-drawn underline beneath the text it sits in (the parent is position:relative).
  import { look } from './look.svelte'
  import { linePaths } from './rough'
  import Strokes from './Strokes.svelte'

  let { stroke = 'var(--accent)', seed = 3 }: { stroke?: string; seed?: number } = $props()
  let w = $state(0)
  const segs = $derived(w ? linePaths(0, 5, w, 4, { stroke, seed, strokeWidth: 2.2, style: look.style }) : [])
</script>

<span class="u" bind:clientWidth={w}><Strokes {segs} viewBox="0 0 {w} 10" /></span>

<style>
  .u {
    position: absolute;
    left: -2px;
    right: -2px;
    bottom: -8px;
    height: 10px;
    pointer-events: none;
  }
</style>
