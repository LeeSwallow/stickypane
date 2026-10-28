<script lang="ts" module>
  // When one tooltip has just been shown, the next one shows at once (moving along a toolbar).
  let lastShown = 0
</script>

<script lang="ts">
  // A short, crisp label for the control it wraps. Shown after a pause on
  // hover, at once on keyboard focus; Esc or a click hides it. It describes
  // the control (aria-describedby) unless the control already says the same
  // with its own name (describe={false}, as IconButton does).
  import type { Snippet } from 'svelte'
  import { MOTION } from '../theme/scale'
  import Kbd from './Kbd.svelte'

  let {
    text,
    shortcut,
    placement = 'bottom',
    align = 'center',
    describe = true,
    mono = false,
    pinned = false,
    children,
  }: {
    text: string
    shortcut?: string
    placement?: 'top' | 'bottom'
    align?: 'start' | 'center' | 'end'
    describe?: boolean
    mono?: boolean
    /** pinned shows it without hover (the reference page) */
    pinned?: boolean
    children: Snippet
  } = $props()

  const id = $props.id()
  let open = $state(false)
  let wrap: HTMLElement | undefined = $state()
  let timer: ReturnType<typeof setTimeout> | undefined

  const show = (now: boolean) => {
    clearTimeout(timer)
    const quick = now || Date.now() - lastShown < 600
    timer = setTimeout(() => {
      open = true
      lastShown = Date.now()
    }, quick ? 0 : 450)
  }
  const hide = () => {
    clearTimeout(timer)
    if (open) lastShown = Date.now()
    open = false
  }

  $effect(() => {
    const target = wrap?.querySelector<HTMLElement>('button, [href], input, textarea, [tabindex]')
    if (!target || !describe) return
    target.setAttribute('aria-describedby', id)
    return () => target.removeAttribute('aria-describedby')
  })
  $effect(() => () => clearTimeout(timer))
</script>

<span
  class="wrap"
  role="presentation"
  bind:this={wrap}
  onpointerenter={() => show(false)}
  onpointerleave={hide}
  onpointerdown={hide}
  onfocusin={(e) => (e.target as HTMLElement).matches?.(':focus-visible') && show(true)}
  onfocusout={hide}
  onkeydown={(e) => e.key === 'Escape' && open && (hide(), e.stopPropagation())}
>
  {@render children()}
  <span
    {id}
    role="tooltip"
    class="tip {placement} {align}"
    class:open={open || pinned}
    class:mono
    style:--tip-t="{MOTION.short}ms"
  >
    <span class="text">{text}</span>{#if shortcut}<Kbd keys={shortcut} tone="tip" />{/if}
  </span>
</span>

<style>
  .wrap {
    position: relative;
    display: inline-flex;
  }
  .tip {
    position: absolute;
    z-index: var(--z-tooltip);
    display: flex;
    align-items: center;
    gap: var(--space-2);
    width: max-content;
    max-width: 360px;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--tip);
    color: var(--tip-ink);
    box-shadow: var(--elev-lifted);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    letter-spacing: 0;
    white-space: normal;
    overflow-wrap: anywhere;
    pointer-events: none;
    opacity: 0;
    visibility: hidden;
    transition:
      opacity var(--tip-t) var(--ease),
      visibility 0s linear var(--tip-t);
  }
  .tip.mono .text {
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .tip.open {
    opacity: 1;
    visibility: visible;
    transition: opacity var(--tip-t) var(--ease);
  }
  .bottom {
    top: calc(100% + var(--space-2));
  }
  .top {
    bottom: calc(100% + var(--space-2));
  }
  .center {
    left: 50%;
    translate: -50% 0;
  }
  .start {
    left: 0;
  }
  .end {
    right: 0;
  }
</style>
