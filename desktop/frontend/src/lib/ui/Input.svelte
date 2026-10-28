<script lang="ts">
  // A one-line field with a label, an optional hint and an error. The error
  // is said in words (aria-invalid + aria-describedby), not only in red.
  import type { HTMLInputAttributes } from 'svelte/elements'
  import Icon from './icons/Icon.svelte'
  import type { IconName } from './icons/icons'

  let {
    label,
    value = $bindable(''),
    hint,
    error,
    icon,
    hideLabel = false,
    disabled = false,
    ...rest
  }: {
    label: string
    value?: string
    hint?: string
    error?: string
    icon?: IconName
    hideLabel?: boolean
    disabled?: boolean
  } & Omit<HTMLInputAttributes, 'value'> = $props()

  const id = $props.id()
  const described = $derived([error && `${id}-err`, hint && `${id}-hint`].filter(Boolean).join(' ') || undefined)
</script>

<div class="field" class:disabled>
  <label for={id} class:sr={hideLabel}>{label}</label>
  <div class="box" class:invalid={!!error}>
    {#if icon}<span class="ic"><Icon name={icon} size={16} /></span>{/if}
    <input {id} bind:value {disabled} aria-invalid={error ? true : undefined} aria-describedby={described} {...rest} />
  </div>
  {#if error}<p class="msg err" id="{id}-err">{error}</p>{/if}
  {#if hint}<p class="msg" id="{id}-hint">{hint}</p>{/if}
</div>

<style>
  .field {
    display: grid;
    gap: 6px;
    min-width: 0;
    --wails-draggable: no-drag;
  }
  label {
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .box {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    height: var(--control-lg);
    padding: 0 var(--space-3);
    box-sizing: border-box;
    border: 1px solid var(--line-strong);
    border-radius: var(--radius-md);
    background: var(--surface-sunk);
    transition:
      border-color var(--t-short) var(--ease),
      box-shadow var(--t-short) var(--ease);
  }
  .box:hover {
    border-color: color-mix(in srgb, var(--line-strong), var(--ink) 15%);
  }
  .box:focus-within,
  .box:has(input[data-state='focus']) {
    border-color: var(--focus);
    box-shadow: 0 0 0 3px var(--accent-wash);
  }
  .box.invalid {
    border-color: var(--danger-text);
  }
  .box.invalid:focus-within {
    box-shadow: 0 0 0 3px var(--danger-wash);
  }
  .ic {
    display: inline-flex;
    color: var(--ink-3);
  }
  input {
    flex: 1;
    min-width: 0;
    height: 100%;
    padding: 0;
    border: 0;
    outline: none;
    background: transparent;
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  input::placeholder {
    color: var(--ink-3);
  }
  .msg {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .err {
    color: var(--danger-text);
  }
  .disabled {
    opacity: 0.5;
  }
  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip-path: inset(50%);
  }
</style>
