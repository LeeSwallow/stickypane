<script lang="ts" module>
  export interface Tab {
    id: string
    label: string
    count?: number
    attention?: boolean
  }
</script>

<script lang="ts">
  // The folder tabs. Crisp labels; the active tab is underlined — in the
  // doodle style by hand (a highlight mark, the one rough line in the
  // chrome), in neat by a straight bar. Arrows move and select.
  import Underline from '../doodle/Underline.svelte'
  import { look } from '../doodle/look.svelte'
  import { seedOf } from '../theme/tokens'
  import Badge from './Badge.svelte'
  import { nextIndex } from './keys'

  let {
    tabs,
    value = $bindable(),
    label,
    onchange,
  }: { tabs: Tab[]; value: string; label: string; onchange?: (id: string) => void } = $props()

  let list: HTMLElement | undefined = $state()

  function select(id: string) {
    value = id
    onchange?.(id)
  }

  function onkeydown(e: KeyboardEvent) {
    const i = tabs.findIndex((t) => t.id === value)
    const j = nextIndex(e.key, i, tabs.length)
    if (j === null) return
    e.preventDefault()
    select(tabs[j].id)
    list?.querySelectorAll<HTMLElement>('[role="tab"]')[j]?.focus()
  }
</script>

<div class="tabs" role="tablist" tabindex="-1" aria-label={label} bind:this={list} {onkeydown}>
  {#each tabs as tab (tab.id)}
    {@const on = tab.id === value}
    <button
      type="button"
      role="tab"
      class="tab"
      class:on
      aria-selected={on}
      tabindex={on ? 0 : -1}
      onclick={() => select(tab.id)}
    >
      <span class="text">
        {tab.label}
        {#if on}
          {#if look.style === 'doodle'}<Underline seed={seedOf(tab.id)} />{:else}<span class="bar"></span>{/if}
        {/if}
      </span>
      {#if tab.count !== undefined}<Badge count={tab.count} tone={tab.attention ? 'warn' : 'neutral'} />{/if}
    </button>
  {/each}
</div>

<style>
  .tabs {
    display: flex;
    align-items: stretch;
    gap: var(--space-1);
    --wails-draggable: no-drag;
  }
  .tab {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    height: var(--control-lg);
    padding: 0 var(--space-3);
    border: 0;
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--ink-2);
    font: 500 var(--text-body-md) / var(--leading-body-md) var(--font-body);
    cursor: default;
    transition:
      color var(--t-short) var(--ease),
      background-color var(--t-short) var(--ease);
  }
  .tab:hover {
    color: var(--ink);
    background: var(--hover);
  }
  .tab.on {
    color: var(--ink);
    font-weight: 600;
  }
  .tab.on:hover {
    background: transparent;
  }
  .tab:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: calc(-1 * var(--focus-width));
  }
  .text {
    position: relative;
    white-space: nowrap;
  }
  .bar {
    position: absolute;
    left: 0;
    right: 0;
    bottom: -6px;
    height: 2px;
    border-radius: 1px;
    background: var(--accent);
  }
</style>
