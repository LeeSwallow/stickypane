<script lang="ts">
  // A dialog drawn as a note taped over the window: paper, a calm hand-drawn
  // border and a strip of tape in the doodle style (a plain card in neat).
  // It holds controls, so it never leans. Modal: focus stays inside, Esc and
  // the scrim close it, focus goes back where it was.
  import type { Snippet } from 'svelte'
  import { tick } from 'svelte'
  import { look } from '../doodle/look.svelte'
  import RoughBox from '../doodle/RoughBox.svelte'
  import Tape from '../doodle/Tape.svelte'
  import { ROUGH, STROKE } from '../theme/scale'
  import { focusables } from './keys'

  let {
    open = $bindable(false),
    title,
    inline = false,
    width = 400,
    role = 'dialog',
    onclose,
    children,
    actions,
  }: {
    open?: boolean
    title: string
    /** drawn in the page flow with no scrim (the reference page) */
    inline?: boolean
    width?: number
    role?: 'dialog' | 'alertdialog'
    onclose?: () => void
    children?: Snippet
    actions?: Snippet
  } = $props()

  const id = $props.id()
  let box: HTMLElement | undefined = $state()

  function close() {
    open = false
    onclose?.()
  }

  $effect(() => {
    if (!open || inline || !box) return
    const back = document.activeElement as HTMLElement | null
    tick().then(() => {
      const first = box?.querySelector<HTMLElement>('[data-autofocus]') ?? (box ? focusables(box)[0] : undefined)
      first?.focus()
    })
    return () => back?.focus()
  })

  function onkeydown(e: KeyboardEvent) {
    if (inline || !box) return
    if (e.key === 'Escape') {
      e.preventDefault()
      close()
    } else if (e.key === 'Tab') {
      const f = focusables(box)
      if (!f.length) return
      const [first, last] = [f[0], f[f.length - 1]]
      if (e.shiftKey && document.activeElement === first) {
        e.preventDefault()
        last.focus()
      } else if (!e.shiftKey && document.activeElement === last) {
        e.preventDefault()
        first.focus()
      }
    }
  }
</script>

{#if open || inline}
  <div class="layer" class:inline>
    {#if !inline}<div class="scrim" aria-hidden="true" onclick={close}></div>{/if}
    <div
      bind:this={box}
      class="dialog"
      class:doodle={look.style === 'doodle'}
      {role}
      aria-modal={inline ? undefined : 'true'}
      aria-labelledby="{id}-t"
      style:width="{width}px"
      tabindex="-1"
      {onkeydown}
    >
      {#if look.style === 'doodle'}
        <Tape />
        <RoughBox stroke="var(--line-strong)" seed={7} strokeWidth={STROKE.note} roughness={ROUGH.note} />
      {/if}
      <h2 id="{id}-t" class="title">{title}</h2>
      {#if children}<div class="body">{@render children()}</div>{/if}
      {#if actions}<div class="actions">{@render actions()}</div>{/if}
    </div>
  </div>
{/if}

<style>
  .layer {
    position: fixed;
    inset: 0;
    z-index: var(--z-dialog);
    display: grid;
    place-items: center;
    padding: var(--space-6);
  }
  .layer.inline {
    position: relative;
    inset: auto;
    z-index: auto;
    padding: var(--space-3) 0 0;
    place-items: start;
  }
  .scrim {
    position: absolute;
    inset: 0;
    background: var(--scrim);
    animation: fade var(--t-base) var(--ease) both;
  }
  .dialog {
    position: relative;
    max-width: 100%;
    box-sizing: border-box;
    padding: var(--space-5) var(--space-6) var(--space-5);
    border: 1px solid var(--line);
    border-radius: var(--radius-xl);
    background: var(--surface);
    color: var(--ink);
    box-shadow: var(--elev-floating);
    outline: none;
    animation: drop var(--t-base) var(--slap) both;
  }
  .dialog.doodle {
    border-color: transparent;
    border-radius: var(--radius-sm);
    background: var(--paper);
  }
  .inline .dialog {
    animation: none;
  }
  .title {
    position: relative;
    margin: 0;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  .dialog:not(.doodle) .title {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .body {
    position: relative;
    margin-top: var(--space-2);
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .actions {
    position: relative;
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
    margin-top: var(--space-5);
  }
  @keyframes drop {
    from {
      opacity: 0;
      translate: 0 -10px;
      scale: 0.98;
    }
  }
  @keyframes fade {
    from {
      opacity: 0;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .dialog {
      animation: fade var(--t-reduced) linear both;
    }
  }
</style>
