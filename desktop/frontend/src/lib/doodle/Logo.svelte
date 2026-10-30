<script lang="ts">
  // The icon, drawn: three memo sheets with crayon hatching and a thick outline.
  // Drawn (not the PNG) so it follows the theme and the style.
  import { look } from './look.svelte'
  import { logoPaths } from './logo'
  import Strokes from './Strokes.svelte'

  let { size = 28 }: { size?: number } = $props()
  const drawn = $derived(logoPaths(look.palette?.notes ?? [], look.style, look.dark, look.palette?.text ?? '#000000'))
  // in dark mode a light rim under the outline, as the app icon has, so the
  // dark outline holds on dark paper (03 §3.11.5)
  const segs = $derived(
    look.dark
      ? [...drawn.filter((g) => g.role === 'outline').map((g) => ({ ...g, stroke: 'rgb(255 255 255 / .26)', strokeWidth: g.strokeWidth + 1.6 })), ...drawn]
      : drawn,
  )
</script>

<span class="logo" style:width="{size}px" style:height="{size}px"><Strokes {segs} viewBox="0 0 32 28" /></span>

<style>
  .logo {
    position: relative;
    display: inline-block;
    flex: none;
  }
</style>
