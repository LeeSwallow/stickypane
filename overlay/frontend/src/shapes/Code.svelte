<script lang="ts">
  // Source as a code block, coloured by Shiki when its language is known
  // (the same colours as a note's code blocks), plain until then. A
  // script and a request show what will run or be sent with it.
  import { highlightCode } from './markdown'

  let { source, lang, label }: { source: string; lang: string; label?: string } = $props()
  let html = $state.raw<{ for: string; html: string } | null>(null)

  $effect(() => {
    const src = source
    let live = true
    highlightCode(src, lang)
      .then((h) => live && h && (html = { for: src, html: h }))
      .catch(() => {})
    return () => (live = false)
  })
</script>

<div class="code" role="region" aria-label={label}>
  {#if html && html.for === source}
    {@html html.html}
  {:else}
    <pre>{source.replace(/\n$/, '')}</pre>
  {/if}
</div>

<style>
  .code :global(pre) {
    margin: 0;
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-md);
    background: var(--select);
    overflow: auto;
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    color: var(--ink);
  }
  .code :global(pre.shiki) {
    background: var(--select) !important;
  }
  .code :global(pre.shiki span) {
    color: var(--shiki-light);
  }
  :global([data-mode='dark']) .code :global(pre.shiki span) {
    color: var(--shiki-dark);
  }
</style>
