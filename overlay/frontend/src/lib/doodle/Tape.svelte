<script lang="ts">
  // A strip of tape holding a note to the paper. Doodle only.
  import { look } from './look.svelte'
  let { left = '50%' }: { left?: string } = $props()
</script>

{#if look.style === 'doodle'}
  <div class="tape" style:left></div>
{/if}

<style>
  .tape {
    position: absolute;
    top: -9px;
    width: 62px;
    height: 18px;
    margin-left: -31px;
    background: var(--tape);
    transform: rotate(-3deg);
    z-index: 2;
    border-left: 1px dashed rgba(255, 255, 255, 0.12);
    border-right: 1px dashed rgba(255, 255, 255, 0.12);
  }
</style>
