<script lang="ts">
  // A hand-drawn rectangle outline around its parent (which is position:relative).
  import { look } from './look.svelte'
  import { boxPaths } from './rough'
  import Strokes from './Strokes.svelte'

  let {
    stroke = 'var(--pencil)',
    fill,
    seed = 1,
    strokeWidth,
    roughness,
  }: { stroke?: string; fill?: string; seed?: number; strokeWidth?: number; roughness?: number } = $props()
  let w = $state(0)
  let h = $state(0)
  const segs = $derived(w ? boxPaths(w, h, { stroke, fill, seed, strokeWidth, roughness, style: look.style }) : [])
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
