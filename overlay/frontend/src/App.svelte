<script lang="ts">
  import { appName } from './lib/app'
</script>

<main>
  <h1>{appName}</h1>
</main>

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
