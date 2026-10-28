<script lang="ts">
  // An on/off switch (role="switch") with its label beside it.
  let {
    checked = $bindable(false),
    label,
    disabled = false,
    onchange,
  }: { checked?: boolean; label: string; disabled?: boolean; onchange?: (on: boolean) => void } = $props()
</script>

<button
  type="button"
  role="switch"
  class="toggle"
  aria-checked={checked}
  {disabled}
  onclick={() => {
    checked = !checked
    onchange?.(checked)
  }}
>
  <span class="track" aria-hidden="true"><span class="knob"></span></span>
  <span class="label">{label}</span>
</button>

<style>
  .toggle {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    padding: 2px;
    border: 0;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    cursor: default;
    --wails-draggable: no-drag;
  }
  .track {
    position: relative;
    width: 28px;
    height: 16px;
    flex: none;
    border-radius: var(--radius-pill);
    background: var(--line-strong);
    transition: background-color var(--t-short) var(--ease);
  }
  .knob {
    position: absolute;
    top: 2px;
    left: 2px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: #fff;
    box-shadow: 0 1px 2px rgb(0 0 0 / 0.3);
    transition: translate var(--t-base) var(--ease);
  }
  [aria-checked='true'] .track {
    background: var(--accent-fill);
  }
  [aria-checked='true'] .knob {
    translate: 12px 0;
  }
  .toggle:focus-visible {
    outline: none;
  }
  .toggle:focus-visible .track {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .toggle:disabled {
    opacity: 0.45;
  }
  @media (prefers-reduced-motion: reduce) {
    .knob {
      transition: none;
    }
  }
</style>
