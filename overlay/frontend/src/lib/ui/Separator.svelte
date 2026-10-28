<script lang="ts">
  // A hairline between groups: vertical in a toolbar, horizontal in a stack.
  let { vertical = true }: { vertical?: boolean } = $props()
</script>

<span class="sep" class:vertical role="separator" aria-orientation={vertical ? 'vertical' : 'horizontal'}></span>

<style>
  .sep {
    display: block;
    flex: none;
    background: var(--line);
    height: 1px;
    width: 100%;
  }
  .vertical {
    width: 1px;
    height: calc(var(--control-md) - var(--space-3));
    margin: 0 var(--space-1);
  }
</style>
