<script lang="ts">
  // A square button with one icon. Its label is its accessible name and its
  // tooltip; a toggle (pressed given) says so with aria-pressed. When pressed
  // in the doodle style it fills with crayon hatching, like the pin in 04.
  import type { HTMLButtonAttributes } from 'svelte/elements'
  import Icon from './icons/Icon.svelte'
  import type { IconName } from './icons/icons'
  import Tooltip from './Tooltip.svelte'
  import CrayonIcon from './icons/CrayonIcon.svelte'
  import { isCrayon } from './icons/crayon'
  import { look } from '../doodle/look.svelte'

  let {
    icon,
    label,
    shortcut,
    size = 'md',
    tone = 'default',
    pressed,
    crayon = false,
    disabled = false,
    tooltip = true,
    placement = 'bottom',
    align = 'center',
    onclick,
    ...rest
  }: {
    icon: IconName
    /** the accessible name and the tooltip; always required */
    label: string
    shortcut?: string
    size?: 'sm' | 'md'
    tone?: 'default' | 'danger'
    /** a toggle: true/false; leave undefined for a plain button */
    pressed?: boolean
    /** draw the icon in the crayon tier (when it has one): for icons that stand alone, or a pressed pin */
    crayon?: boolean
    disabled?: boolean
    tooltip?: boolean
    placement?: 'top' | 'bottom'
    align?: 'start' | 'center' | 'end'
    onclick?: (e: MouseEvent) => void
  } & Omit<HTMLButtonAttributes, 'onclick' | 'aria-label' | 'aria-pressed'> = $props()
</script>

{#snippet button()}
  <button
    type="button"
    class="ib {size} {tone}"
    class:on={pressed}
    class:crayon
    class:doodle={look.style === 'doodle'}
    aria-label={label}
    aria-pressed={pressed}
    {disabled}
    {onclick}
    {...rest}
  >
    {#if crayon && isCrayon(icon)}
      <CrayonIcon name={icon} size={size === 'sm' ? 16 : 20} />
    {:else}
      <Icon name={icon} size={size === 'sm' ? 16 : 20} />
    {/if}
  </button>
{/snippet}

{#if tooltip && !disabled}
  <Tooltip text={label} {shortcut} {placement} {align} describe={false}>{@render button()}</Tooltip>
{:else}
  {@render button()}
{/if}

<style>
  .ib {
    display: inline-grid;
    place-items: center;
    flex: none;
    width: var(--control-md);
    height: var(--control-md);
    padding: 0;
    border: 0;
    border-radius: var(--radius-md);
    background: transparent;
    color: var(--ink-2);
    cursor: default;
    transition:
      background-color var(--t-short) var(--ease),
      color var(--t-short) var(--ease);
    --wails-draggable: no-drag;
  }
  .sm {
    width: var(--control-sm);
    height: var(--control-sm);
    border-radius: var(--radius-sm);
  }
  .ib:hover:not(:disabled),
  .ib[data-state='hover'] {
    background: var(--hover);
    color: var(--ink);
  }
  .ib:active:not(:disabled),
  .ib[data-state='active'] {
    background: var(--press);
    color: var(--ink);
  }
  .ib:focus-visible,
  .ib[data-state='focus'] {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .ib.on {
    background: var(--accent-wash);
    color: var(--accent-text);
  }
  .ib.on.doodle:not(.crayon) {
    background-image: repeating-linear-gradient(
      -41deg,
      color-mix(in srgb, var(--accent) 30%, transparent) 0 1.25px,
      transparent 1.25px 4px
    );
  }
  .danger:hover:not(:disabled),
  .danger[data-state='hover'] {
    background: var(--danger-wash);
    color: var(--danger-text);
  }
  .ib:disabled {
    color: var(--ink-3);
    opacity: 0.45;
  }
</style>
