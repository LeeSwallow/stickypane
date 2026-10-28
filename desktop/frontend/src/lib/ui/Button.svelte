<script lang="ts">
  // A text button. primary: the one thing to do here (one per view);
  // secondary: everything else; ghost: quiet, in toolbars and toasts;
  // danger: deletes or runs something. Loading keeps the label (so the width
  // holds) behind a still ring — no spinner, nothing repeats (04 M3).
  import type { Snippet } from 'svelte'
  import type { HTMLButtonAttributes } from 'svelte/elements'
  import { tr } from '../../i18n'
  import Icon from './icons/Icon.svelte'
  import type { IconName } from './icons/icons'

  let {
    variant = 'secondary',
    size = 'md',
    icon,
    iconEnd,
    loading = false,
    disabled = false,
    type = 'button',
    full = false,
    onclick,
    children,
    ...rest
  }: {
    variant?: 'primary' | 'secondary' | 'ghost' | 'danger'
    size?: 'sm' | 'md' | 'lg'
    icon?: IconName
    iconEnd?: IconName
    loading?: boolean
    disabled?: boolean
    type?: 'button' | 'submit'
    full?: boolean
    onclick?: (e: MouseEvent) => void
    children: Snippet
  } & Omit<HTMLButtonAttributes, 'onclick' | 'type'> = $props()
</script>

<button
  {type}
  class="btn {variant} {size}"
  class:full
  class:loading
  disabled={disabled || loading}
  aria-busy={loading || undefined}
  aria-disabled={disabled || loading || undefined}
  {onclick}
  {...rest}
>
  {#if loading}
    <svg class="ring" width="14" height="14" viewBox="0 0 14 14" aria-hidden="true">
      <circle cx="7" cy="7" r="5.25" fill="none" stroke="currentColor" stroke-width="1.75" stroke-dasharray="2.5 3" stroke-linecap="round" />
    </svg>
    <span class="sr">{$tr('ui.loading')}</span>
  {:else if icon}
    <Icon name={icon} size={size === 'sm' ? 14 : 16} />
  {/if}
  <span class="label">{@render children()}</span>
  {#if iconEnd}<Icon name={iconEnd} size={size === 'sm' ? 14 : 16} />{/if}
</button>

<style>
  .btn {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: var(--space-2);
    height: var(--control-md);
    padding: 0 var(--space-3);
    border: 1px solid transparent;
    border-radius: var(--radius-md);
    font: 600 var(--text-body-md) / 1 var(--font-body);
    white-space: nowrap;
    cursor: default;
    user-select: none;
    transition:
      background-color var(--t-short) var(--ease),
      border-color var(--t-short) var(--ease),
      color var(--t-short) var(--ease),
      filter var(--t-short) var(--ease);
    --wails-draggable: no-drag;
  }
  .sm {
    height: var(--control-sm);
    padding: 0 var(--space-2);
    gap: var(--space-1);
    font-size: var(--text-body-sm);
    border-radius: var(--radius-sm);
  }
  .lg {
    height: var(--control-lg);
    padding: 0 var(--space-4);
    font-size: var(--text-body-lg);
  }
  .full {
    width: 100%;
  }
  .label {
    overflow: hidden;
    text-overflow: ellipsis;
  }

  .primary {
    background: var(--accent-fill);
    color: var(--accent-ink);
    box-shadow: var(--elev-taped);
  }
  .secondary {
    background: var(--surface);
    border-color: var(--line-strong);
    color: var(--ink);
  }
  .ghost {
    background: transparent;
    color: var(--ink-2);
  }
  .danger {
    background: var(--danger-fill);
    color: var(--danger-ink);
    box-shadow: var(--elev-taped);
  }

  .primary:hover:not(:disabled),
  .danger:hover:not(:disabled),
  .primary[data-state='hover'],
  .danger[data-state='hover'] {
    filter: brightness(1.08) saturate(1.05);
  }
  .secondary:hover:not(:disabled),
  .secondary[data-state='hover'] {
    background: color-mix(in srgb, var(--surface), var(--ink) 5%);
    border-color: color-mix(in srgb, var(--line-strong), var(--ink) 12%);
  }
  .ghost:hover:not(:disabled),
  .ghost[data-state='hover'] {
    background: var(--hover);
    color: var(--ink);
  }

  .btn:active:not(:disabled),
  .btn[data-state='active'] {
    filter: brightness(0.94);
    translate: 0 0.5px;
  }
  .ghost:active:not(:disabled),
  .ghost[data-state='active'] {
    background: var(--press);
    filter: none;
  }

  .btn:focus-visible,
  .btn[data-state='focus'] {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }

  .btn:disabled:not(.loading) {
    opacity: 0.45;
    box-shadow: none;
  }
  .loading .label {
    opacity: 0.7;
  }
  .ring {
    flex: none;
  }
  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }

  @media (prefers-reduced-motion: reduce) {
    .btn:active:not(:disabled) {
      translate: none;
    }
  }
</style>
