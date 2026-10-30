<script lang="ts">
  // One setting: its name and a line on what it does on the left, the
  // control on the right — or, for a control drawn as pictures (stack),
  // below. Rows are parted by a hairline (03 §3.12).
  import type { Snippet } from 'svelte'

  let {
    label,
    hint = '',
    stack = false,
    id,
    children,
    status,
  }: { label: string; hint?: string; stack?: boolean; id?: string; children?: Snippet; status?: Snippet } = $props()
  const uid = $props.id()
  const lid = $derived(id ?? `${uid}-label`)
</script>

<div class="row" class:stack role="group" aria-labelledby={lid}>
  <div class="what">
    <div class="name">
      <span class="label" id={lid}>{label}</span>
      {@render status?.()}
    </div>
    {#if hint}<p class="hint">{hint}</p>{/if}
  </div>
  {#if children}<div class="control">{@render children()}</div>{/if}
</div>

<style>
  .row {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-4) 0;
    border-top: var(--stroke-hairline) solid var(--line);
  }
  .row:first-of-type {
    border-top: 0;
  }
  .row.stack {
    flex-direction: column;
    align-items: stretch;
    gap: var(--space-3);
  }
  .what {
    flex: 1;
    min-width: 0;
  }
  .name {
    display: flex;
    align-items: center;
    flex-wrap: wrap;
    gap: var(--space-2);
  }
  .label {
    color: var(--ink);
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .hint {
    margin: 2px 0 0;
    max-width: 52ch;
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .control {
    display: flex;
    flex: none;
    align-items: center;
    justify-content: flex-end;
    gap: var(--space-2);
    min-width: 0;
  }
  .stack .control {
    display: block;
  }
</style>
