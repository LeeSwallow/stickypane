<script lang="ts">
  // One choice drawn as a picture (03 §3.12): a theme, a style, light or
  // dark. A radio in its group: the picture, a name, a line under it, and
  // when chosen an accent ring and a check. The group moves the focus
  // between the cards with the arrows (CardGroup).
  import type { Snippet } from 'svelte'
  import Icon from '../lib/ui/icons/Icon.svelte'
  import { look } from '../lib/doodle/look.svelte'

  let {
    label,
    sub = '',
    checked = false,
    onpick,
    picture,
  }: { label: string; sub?: string; checked?: boolean; onpick: () => void; picture: Snippet } = $props()
</script>

<button
  type="button"
  role="radio"
  aria-checked={checked}
  tabindex={checked ? 0 : -1}
  class="card"
  class:checked
  class:neat={look.style === 'neat'}
  onclick={onpick}
>
  <span class="picture">{@render picture()}</span>
  <span class="text">
    <span class="label">{label}</span>
    {#if sub}<span class="sub">{sub}</span>{/if}
  </span>
  {#if checked}<span class="tick" aria-hidden="true"><Icon name="check" size={14} /></span>{/if}
</button>

<style>
  .card {
    position: relative;
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-width: 0;
    padding: var(--space-2);
    border: 1px solid var(--line);
    border-radius: var(--radius-lg);
    background: var(--surface);
    color: var(--ink);
    text-align: left;
    cursor: pointer;
    transition:
      border-color var(--t-short) var(--ease),
      box-shadow var(--t-short) var(--ease);
  }
  .card:hover {
    border-color: var(--line-strong);
  }
  .card.checked {
    border-color: var(--accent);
    box-shadow: 0 0 0 1px var(--accent);
  }
  .card:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .picture {
    display: block;
    overflow: hidden;
    border-radius: var(--radius-md);
    border: 1px solid var(--line);
  }
  .text {
    display: flex;
    flex-direction: column;
    min-width: 0;
    padding: 0 var(--space-1);
  }
  .label {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
  }
  .neat .label {
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .sub {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .tick {
    position: absolute;
    top: var(--space-1);
    right: var(--space-1);
    display: grid;
    place-items: center;
    width: 22px;
    height: 22px;
    border-radius: 50%;
    background: var(--accent-fill);
    color: var(--accent-ink);
  }
</style>
