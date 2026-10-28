<script lang="ts">
  // Nothing here yet: the three crayon sheets of the icon (empty-state art is
  // one of the places hand drawing belongs), a title, a line, an action.
  import type { Snippet } from 'svelte'
  import Logo from '../doodle/Logo.svelte'
  import { look } from '../doodle/look.svelte'

  let {
    title,
    body,
    art = true,
    children,
  }: { title: string; body?: string; art?: boolean; children?: Snippet } = $props()
</script>

<div class="empty">
  {#if art}<div class="art" aria-hidden="true"><Logo size={72} /></div>{/if}
  <h3 class="title" class:neat={look.style === 'neat'}>{title}</h3>
  {#if body}<p class="body">{body}</p>{/if}
  {#if children}<div class="actions">{@render children()}</div>{/if}
</div>

<style>
  .empty {
    display: grid;
    justify-items: center;
    gap: var(--space-2);
    padding: var(--space-8) var(--space-6);
    text-align: center;
  }
  .art {
    margin-bottom: var(--space-2);
    opacity: 0.9;
  }
  .title {
    margin: 0;
    color: var(--ink);
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  .title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .body {
    max-width: 36ch;
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .actions {
    display: flex;
    gap: var(--space-2);
    margin-top: var(--space-3);
  }
</style>
