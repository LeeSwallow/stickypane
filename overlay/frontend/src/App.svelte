<script lang="ts">
  import { appName } from './lib/app'
  // TEMPORARY: the look demo, at ?demo=look, in dev builds only. The dynamic
  // import behind import.meta.env.DEV is dropped from production builds.
  const demo = import.meta.env.DEV ? new URLSearchParams(location.search).get('demo') : null
  const LookDemo = demo === 'look' ? import('./demo/LookDemo.svelte') : null
</script>

{#if LookDemo}
  {#await LookDemo then m}
    <m.default />
  {/await}
{:else}
  <main>
    <h1>{appName}</h1>
  </main>
{/if}

<style>
  main {
    height: 100%;
    display: grid;
    place-items: center;
  }
  h1 {
    font-family: var(--font-hand);
    font-weight: 700;
    font-size: 28px;
  }
</style>
