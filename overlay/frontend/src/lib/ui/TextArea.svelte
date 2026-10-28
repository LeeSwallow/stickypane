<script lang="ts">
  // A many-line field: the same label, hint and error as Input.
  import type { HTMLTextareaAttributes } from 'svelte/elements'

  let {
    label,
    value = $bindable(''),
    hint,
    error,
    rows = 4,
    disabled = false,
    ...rest
  }: {
    label: string
    value?: string
    hint?: string
    error?: string
    rows?: number
    disabled?: boolean
  } & Omit<HTMLTextareaAttributes, 'value'> = $props()

  const id = $props.id()
  const described = $derived([error && `${id}-err`, hint && `${id}-hint`].filter(Boolean).join(' ') || undefined)
</script>

<div class="field" class:disabled>
  <label for={id}>{label}</label>
  <textarea {id} {rows} bind:value {disabled} class:invalid={!!error} aria-invalid={error ? true : undefined} aria-describedby={described} {...rest}></textarea>
  {#if error}<p class="msg err" id="{id}-err">{error}</p>{/if}
  {#if hint}<p class="msg" id="{id}-hint">{hint}</p>{/if}
</div>

<style>
  .field {
    display: grid;
    gap: 6px;
    --wails-draggable: no-drag;
  }
  label {
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  textarea {
    resize: vertical;
    padding: var(--space-2) var(--space-3);
    border: 1px solid var(--line-strong);
    border-radius: var(--radius-md);
    background: var(--surface-sunk);
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    outline: none;
    transition:
      border-color var(--t-short) var(--ease),
      box-shadow var(--t-short) var(--ease);
  }
  textarea::placeholder {
    color: var(--ink-3);
  }
  textarea:focus {
    border-color: var(--focus);
    box-shadow: 0 0 0 3px var(--accent-wash);
  }
  textarea.invalid {
    border-color: var(--danger-text);
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
</style>
