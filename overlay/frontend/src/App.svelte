<script lang="ts">
  import { appName } from './lib/app'
  // TEMPORARY: the look demo, at ?demo=look. Removed when the real screens exist.
  import LookDemo from './demo/LookDemo.svelte'
  const demo = new URLSearchParams(location.search).get('demo')
</script>

{#if demo === 'look'}
  <LookDemo />
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
