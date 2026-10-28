<script lang="ts">
  // A hand-drawn arrow pointing right.
  import { look } from './look.svelte'
  import { linePaths, polylinePaths } from './rough'
  import Strokes from './Strokes.svelte'

  let {
    stroke = 'var(--pencil)',
    seed = 5,
    width = 60,
    height = 20,
  }: { stroke?: string; seed?: number; width?: number; height?: number } = $props()
  const y = $derived(height / 2)
  const segs = $derived([
    ...linePaths(2, y, width - 3, y, { stroke, seed, strokeWidth: 1.8, style: look.style }),
    ...polylinePaths(
      [
        [width - 12, y - 6],
        [width - 2, y],
        [width - 12, y + 6],
      ],
      { stroke, seed: seed + 1, strokeWidth: 1.8, style: look.style },
    ),
  ])
</script>

<span class="arrow" style:width="{width}px" style:height="{height}px"><Strokes {segs} viewBox="0 0 {width} {height}" /></span>

<style>
  .arrow {
    position: relative;
    display: inline-block;
    flex: none;
  }
</style>
