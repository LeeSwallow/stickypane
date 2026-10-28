<script lang="ts">
  // A menu of actions: under its trigger (anchored in a position:relative
  // parent), or at a point (a context menu: `at`). Arrows move, Home/End
  // jump, Enter/Space choose, Esc or a click outside closes and gives focus
  // back. Crisp: menus are chrome.
  import { tick } from 'svelte'
  import Icon from './icons/Icon.svelte'
  import Kbd from './Kbd.svelte'
  import { nextIndex } from './keys'
  import { isAction, type MenuItem } from './menu'

  let {
    items,
    open = $bindable(false),
    label,
    at,
    align = 'end',
    inline = false,
    onselect,
  }: {
    items: MenuItem[]
    open?: boolean
    label: string
    /** a context menu: the point it opens at, in the viewport */
    at?: { x: number; y: number }
    align?: 'start' | 'end'
    /** drawn in the page flow, always open (the reference page) */
    inline?: boolean
    onselect?: (id: string) => void
  } = $props()

  let el: HTMLElement | undefined = $state()
  let back: HTMLElement | null = null

  const buttons = () => [...(el?.querySelectorAll<HTMLElement>('[role="menuitem"]') ?? [])]

  $effect(() => {
    if (!open || inline) return
    back = document.activeElement as HTMLElement | null
    tick().then(() => buttons().find((b) => b.getAttribute('aria-disabled') !== 'true')?.focus())
    const outside = (e: PointerEvent) => {
      if (el && !el.contains(e.target as Node)) close(false)
    }
    window.addEventListener('pointerdown', outside, true)
    return () => window.removeEventListener('pointerdown', outside, true)
  })

  function close(refocus = true) {
    open = false
    if (refocus) back?.focus()
  }

  function choose(id: string, disabled?: boolean) {
    if (disabled) return
    close()
    onselect?.(id)
  }

  function onkeydown(e: KeyboardEvent) {
    const bs = buttons()
    const i = bs.indexOf(document.activeElement as HTMLElement)
    if (e.key === 'Escape') {
      e.preventDefault()
      e.stopPropagation()
      close()
      return
    }
    if (e.key === 'Tab') {
      close(false)
      return
    }
    const j = nextIndex(e.key, i, bs.length, 'vertical', (k) => bs[k].getAttribute('aria-disabled') === 'true')
    if (j !== null) {
      e.preventDefault()
      bs[j].focus()
    }
  }
</script>

{#if open || inline}
  <div
    bind:this={el}
    class="menu {align}"
    class:ctx={!!at}
    class:inline
    role="menu"
    tabindex="-1"
    aria-label={label}
    style:left={at ? `${at.x}px` : undefined}
    style:top={at ? `${at.y}px` : undefined}
    {onkeydown}
  >
    {#each items as m (m.id)}
      {#if isAction(m)}
        <button
          type="button"
          role="menuitem"
          tabindex="-1"
          class="item"
          class:danger={m.danger}
          aria-disabled={m.disabled || undefined}
          onclick={() => choose(m.id, m.disabled)}
        >
          <span class="ic">{#if m.icon}<Icon name={m.icon} size={16} />{/if}</span>
          <span class="label">{m.label}</span>
          {#if m.shortcut}<Kbd keys={m.shortcut} />{/if}
        </button>
      {:else}
        <div class="sep" role="separator"></div>
      {/if}
    {/each}
  </div>
{/if}

<style>
  .menu {
    position: absolute;
    top: calc(100% + var(--space-1));
    z-index: var(--z-menu);
    min-width: 200px;
    max-width: 320px;
    padding: var(--space-1);
    border: 1px solid var(--line);
    border-radius: var(--radius-lg);
    background: var(--surface);
    box-shadow: var(--elev-lifted);
    outline: none;
    --wails-draggable: no-drag;
  }
  .menu.end {
    right: 0;
  }
  .menu.start {
    left: 0;
  }
  .menu.ctx {
    position: fixed;
    right: auto;
  }
  .menu.inline {
    position: relative;
    top: auto;
    right: auto;
    left: auto;
  }
  .item {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: 100%;
    height: var(--row-h);
    padding: 0 var(--space-2);
    border: 0;
    border-radius: var(--radius-sm);
    background: transparent;
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    text-align: left;
    cursor: default;
  }
  .ic {
    display: grid;
    place-items: center;
    width: 16px;
    color: var(--ink-2);
  }
  .label {
    flex: 1;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .item:hover,
  .item:focus-visible,
  .item:focus {
    background: var(--hover);
    outline: none;
  }
  .item:focus-visible {
    box-shadow: inset 0 0 0 var(--focus-width) var(--focus);
  }
  .item.danger,
  .item.danger .ic {
    color: var(--danger-text);
  }
  .item.danger:hover,
  .item.danger:focus {
    background: var(--danger-wash);
  }
  .item[aria-disabled='true'] {
    color: var(--ink-3);
    opacity: 0.6;
  }
  .item[aria-disabled='true']:hover {
    background: transparent;
  }
  .sep {
    height: 1px;
    margin: var(--space-1) calc(-1 * var(--space-1));
    background: var(--line);
  }
</style>
