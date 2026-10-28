<script lang="ts" module>
  import type { IconName } from './icons/icons'
  export interface Segment {
    value: string
    label: string
    icon?: IconName
  }
</script>

<script lang="ts">
  // One of a few, side by side (grid ↔ canvas; card / half / page). A radio
  // group: one tab stop, arrows choose. Icon-only segments still have their
  // label as name and tooltip.
  import Icon from './icons/Icon.svelte'
  import { nextIndex } from './keys'
  import Tooltip from './Tooltip.svelte'

  let {
    options,
    value = $bindable(),
    label,
    iconOnly = false,
    size = 'md',
    onchange,
  }: {
    options: Segment[]
    value: string
    label: string
    iconOnly?: boolean
    size?: 'sm' | 'md'
    onchange?: (v: string) => void
  } = $props()

  let group: HTMLElement | undefined = $state()

  function pick(v: string) {
    if (v === value) return
    value = v
    onchange?.(v)
  }

  function onkeydown(e: KeyboardEvent) {
    const i = options.findIndex((o) => o.value === value)
    const j = nextIndex(e.key, i, options.length)
    if (j === null) return
    e.preventDefault()
    pick(options[j].value)
    group?.querySelectorAll<HTMLElement>('[role="radio"]')[j]?.focus()
  }
</script>

{#snippet seg(o: Segment)}
  {@const on = o.value === value}
  <button
    type="button"
    role="radio"
    class="seg"
    class:on
    aria-checked={on}
    aria-label={iconOnly ? o.label : undefined}
    tabindex={on ? 0 : -1}
    onclick={() => pick(o.value)}
  >
    {#if o.icon}<Icon name={o.icon} size={size === 'sm' ? 16 : 18} />{/if}
    {#if !iconOnly}<span>{o.label}</span>{/if}
  </button>
{/snippet}

<div class="group {size}" class:icons={iconOnly} role="radiogroup" tabindex="-1" aria-label={label} bind:this={group} {onkeydown}>
  {#each options as o (o.value)}
    {#if iconOnly}
      <Tooltip text={o.label} describe={false}>{@render seg(o)}</Tooltip>
    {:else}
      {@render seg(o)}
    {/if}
  {/each}
</div>

<style>
  .group {
    display: inline-flex;
    align-items: stretch;
    gap: 2px;
    height: var(--control-md);
    padding: 2px;
    box-sizing: border-box;
    border-radius: var(--radius-md);
    background: var(--surface-sunk);
    box-shadow: inset 0 0 0 1px var(--line);
    --wails-draggable: no-drag;
  }
  .group.sm {
    height: var(--control-sm);
    border-radius: var(--radius-sm);
  }
  .seg {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 6px;
    height: 100%;
    padding: 0 var(--space-3);
    border: 0;
    border-radius: calc(var(--radius-md) - 2px);
    background: transparent;
    color: var(--ink-2);
    font: 500 var(--text-body-sm) / 1 var(--font-body);
    white-space: nowrap;
    cursor: default;
    transition:
      background-color var(--t-short) var(--ease),
      color var(--t-short) var(--ease);
  }
  .icons .seg {
    aspect-ratio: 1;
    padding: 0;
  }
  .sm .seg {
    border-radius: 2px;
  }
  .seg:hover {
    color: var(--ink);
  }
  .seg.on {
    background: var(--surface);
    color: var(--ink);
    box-shadow:
      0 0 0 1px var(--line),
      var(--elev-taped);
  }
  .seg:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: 0;
  }
</style>
