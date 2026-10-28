<script lang="ts">
  // A hand-drawn checkbox. Done is told by colour, a tick and (in the row) a strike-through.
  import { look } from './look.svelte'
  import { boxPaths, polylinePaths } from './rough'
  import Strokes from './Strokes.svelte'

  let { checked = false, seed = 1, size = 20 }: { checked?: boolean; seed?: number; size?: number } = $props()
  const box = $derived(boxPaths(size, size, { stroke: checked ? 'var(--good)' : 'var(--pencil)', seed, strokeWidth: 1.5, style: look.style }))
  // the tick apart, so that it can be drawn as with a pen (lib/motion)
  const tick = $derived(
    checked
      ? polylinePaths(
          [
            [size * 0.25, size * 0.52],
            [size * 0.43, size * 0.72],
            [size * 0.78, size * 0.25],
          ],
          { stroke: 'var(--good)', strokeWidth: 2.4, seed: seed + 1, style: look.style },
        )
      : [],
  )
</script>

<span class="check" style:width="{size}px" style:height="{size}px"
  ><Strokes segs={box} viewBox="0 0 {size} {size}" />{#if tick.length}<span class="tick" data-pen><Strokes segs={tick} viewBox="0 0 {size} {size}" /></span>{/if}</span
>

<style>
  .tick {
    position: absolute;
    inset: 0;
  }
  .check {
    position: relative;
    display: inline-block;
    flex: none;
  }
</style>
