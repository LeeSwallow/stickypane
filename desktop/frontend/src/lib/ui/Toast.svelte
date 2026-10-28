<script lang="ts">
  // A short message at the bottom of the window, with at most one action
  // (되돌리기 / Undo, 종료 / Quit). It comes in once (04 M3) and goes by
  // itself after `timeout` ms unless the pointer or focus is on it.
  import { onMount } from 'svelte'
  import { tr } from '../../i18n'
  import Button from './Button.svelte'
  import IconButton from './IconButton.svelte'
  import Icon from './icons/Icon.svelte'
  import type { IconName } from './icons/icons'

  let {
    message,
    tone = 'neutral',
    icon,
    action,
    timeout = 0,
    onclose,
  }: {
    message: string
    tone?: 'neutral' | 'good' | 'warn' | 'bad'
    icon?: IconName
    action?: { label: string; run: () => void }
    /** ms before it goes by itself; 0 stays */
    timeout?: number
    onclose?: () => void
  } = $props()

  let held = false
  onMount(() => {
    if (!timeout) return
    const id = setInterval(() => !held && (clearInterval(id), onclose?.()), timeout)
    return () => clearInterval(id)
  })
</script>

<div
  class="toast {tone}"
  role={tone === 'bad' ? 'alert' : 'status'}
  aria-live={tone === 'bad' ? 'assertive' : 'polite'}
  onpointerenter={() => (held = true)}
  onpointerleave={() => (held = false)}
  onfocusin={() => (held = true)}
  onfocusout={() => (held = false)}
>
  {#if icon}<span class="ic"><Icon name={icon} size={16} /></span>{/if}
  <span class="msg">{message}</span>
  {#if action}
    <Button variant="ghost" size="sm" onclick={action.run}><span class="act">{action.label}</span></Button>
  {/if}
  {#if onclose}
    <IconButton icon="close" size="sm" label={$tr('ui.dismiss')} onclick={onclose} placement="top" />
  {/if}
</div>

<style>
  .toast {
    display: inline-flex;
    align-items: center;
    gap: var(--space-2);
    max-width: 440px;
    min-height: var(--control-lg);
    padding: var(--space-1) var(--space-1) var(--space-1) var(--space-3);
    box-sizing: border-box;
    border: 1px solid var(--line);
    border-radius: var(--radius-lg);
    background: var(--surface);
    color: var(--ink);
    box-shadow: var(--elev-lifted);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    animation: in var(--t-base) var(--slap) both;
  }
  .msg {
    flex: 1;
    padding: var(--space-1) 0;
  }
  .ic {
    display: inline-flex;
    color: var(--ink-2);
  }
  .good .ic {
    color: var(--good-text);
  }
  .warn .ic {
    color: var(--warn-text);
  }
  .bad .ic {
    color: var(--danger-text);
  }
  .act {
    color: var(--accent-text);
  }
  @keyframes in {
    from {
      opacity: 0;
      translate: 0 8px;
    }
  }
  @media (prefers-reduced-motion: reduce) {
    .toast {
      animation: fade var(--t-reduced) linear both;
    }
    @keyframes fade {
      from {
        opacity: 0;
      }
    }
  }
</style>
