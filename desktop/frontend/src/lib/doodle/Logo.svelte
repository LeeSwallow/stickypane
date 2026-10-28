<script lang="ts">
  // The icon, drawn: three memo sheets with crayon hatching and a thick outline.
  // Drawn (not the PNG) so it follows the theme and the style.
  import { look } from './look.svelte'
  import { logoPaths } from './logo'
  import Strokes from './Strokes.svelte'

  let { size = 28 }: { size?: number } = $props()
  const segs = $derived(logoPaths(look.palette?.notes ?? [], look.style, look.dark, look.palette?.text ?? '#000000'))
</script>

<span class="logo" style:width="{size}px" style:height="{size}px"><Strokes {segs} viewBox="0 0 32 28" /></span>

<style>
  .logo {
    position: relative;
    display: inline-block;
    flex: none;
  }
</style>
