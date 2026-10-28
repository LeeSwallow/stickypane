<script lang="ts">
  // An agent's name tag (the Glance line, 03 §3.6): a colour dot (its note
  // colour), @name, what it is on and for how long. Stalled adds the warn
  // border, an icon and the time in warn — never colour alone. With onclick
  // it is a filter toggle (aria-pressed).
  import Icon from './icons/Icon.svelte'

  let {
    name,
    color,
    detail,
    time,
    stalled = false,
    selected,
    onclick,
    ...rest
  }: {
    name: string
    color: string
    detail?: string
    time?: string
    stalled?: boolean
    selected?: boolean
    onclick?: () => void
    [k: `data-${string}`]: string
  } = $props()
</script>

<svelte:element
  this={onclick ? 'button' : 'span'}
  type={onclick ? 'button' : undefined}
  class="chip"
  class:stalled
  class:on={selected}
  class:button={!!onclick}
  aria-pressed={onclick ? !!selected : undefined}
  role={onclick ? undefined : 'group'}
  {onclick}
  {...rest}
>
  <i class="dot" style:background={color} aria-hidden="true"></i>
  <span class="name">@{name}</span>
  {#if detail}<span class="detail">{detail}</span>{/if}
  {#if stalled}<span class="warn"><Icon name="warn" size={14} /></span>{/if}
  {#if time}<span class="time">{time}</span>{/if}
</svelte:element>

<style>
  .chip {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    max-width: 280px;
    height: var(--control-md);
    padding: 0 var(--space-3) 0 var(--space-2);
    box-sizing: border-box;
    border: 1px solid var(--line);
    border-radius: var(--radius-pill);
    background: var(--surface);
    color: var(--ink);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    white-space: nowrap;
    --wails-draggable: no-drag;
  }
  .button {
    cursor: default;
    transition:
      background-color var(--t-short) var(--ease),
      border-color var(--t-short) var(--ease);
  }
  .button:hover,
  .chip[data-state='hover'] {
    background: color-mix(in srgb, var(--surface), var(--ink) 6%);
    border-color: var(--line-strong);
  }
  .button:focus-visible,
  .chip[data-state='focus'] {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .on {
    border-color: var(--accent);
    background: var(--accent-wash);
  }
  .stalled {
    border-color: var(--warn);
  }
  .dot {
    width: 8px;
    height: 8px;
    flex: none;
    border-radius: 50%;
    box-shadow: 0 0 0 1px rgb(0 0 0 / 0.15);
  }
  .name {
    font-weight: 600;
  }
  .detail {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    color: var(--ink-2);
  }
  .time {
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
    font-variant-numeric: tabular-nums;
  }
  .warn,
  .stalled .time {
    color: var(--warn-text);
  }
  .warn {
    display: inline-flex;
  }
</style>
