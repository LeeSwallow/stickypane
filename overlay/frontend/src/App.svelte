<script lang="ts">
  // Which window this page is decides what it draws: the home list or one
  // project's board. The demo pages (look, system, screens, shapes, motion)
  // are reachable in dev builds only. The launch intro plays over either
  // window when it first shows (lib/motion/Intro.svelte).
  import { windowKind } from './lib/shell'
  import Home from './home/Home.svelte'
  import Board from './board/Board.svelte'
  import Intro from './lib/motion/Intro.svelte'
  import { settingsService } from './services/settings'

  const kind = windowKind(location.search)
  const demo = import.meta.env.DEV ? new URLSearchParams(location.search).get('demo') : null
  const LookDemo = demo === 'look' ? import('./demo/LookDemo.svelte') : null
  const SystemDemo = demo === 'system' ? import('./demo/SystemDemo.svelte') : null
  const ScreensDemo = demo === 'screens' ? import('./demo/ScreensDemo.svelte') : null
  const ShapesDemo = demo === 'shapes' ? import('./demo/ShapesDemo.svelte') : null
  const MotionDemo = demo === 'motion' ? import('./demo/MotionDemo.svelte') : null
</script>

{#if LookDemo}
  {#await LookDemo then m}
    <m.default />
  {/await}
{:else if SystemDemo}
  {#await SystemDemo then m}
    <m.default />
  {/await}
{:else if ScreensDemo}
  {#await ScreensDemo then m}
    <m.default />
  {/await}
{:else if ShapesDemo}
  {#await ShapesDemo then m}
    <m.default />
  {/await}
{:else if MotionDemo}
  {#await MotionDemo then m}
    <m.default />
  {/await}
{:else}
  {#if kind === 'project'}
    <Board />
  {:else}
    <Home />
  {/if}
  <Intro prefs={settingsService.appPrefs} />
{/if}
