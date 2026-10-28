<script lang="ts">
  // The title bar (03 §3.6): the drag area of the frameless window, with the
  // logo, the project's name and path, the hotkey, "+ Create" (A9 fills it
  // in) and the window's pin.
  import Logo from '../lib/doodle/Logo.svelte'
  import { tr } from '../i18n'

  let {
    title,
    path,
    hotkey = '',
    pinned = false,
    onpin,
    oncreate,
  }: { title: string; path: string; hotkey?: string; pinned?: boolean; onpin: () => void; oncreate?: () => void } = $props()

  /** "alt+space" is "⌥Space", as macOS writes it. */
  const keys: Record<string, string> = { alt: '⌥', option: '⌥', cmd: '⌘', command: '⌘', ctrl: '⌃', control: '⌃', shift: '⇧' }
  const label = $derived(
    hotkey
      .split('+')
      .map((k) => keys[k.toLowerCase()] ?? k.charAt(0).toUpperCase() + k.slice(1))
      .join(''),
  )
  const home = $derived(path.replace(/^\/(Users|home)\/[^/]+/, '~'))
</script>

<header class="bar">
  <Logo size={28} />
  <h1>{title}</h1>
  <span class="path" title={path}>{home}</span>
  <span class="grow"></span>
  {#if label}<kbd title={$tr('window.hotkey')}>{label}</kbd>{/if}
  <button class="create" disabled={!oncreate} title={oncreate ? '' : $tr('create.soon')} onclick={() => oncreate?.()}>{$tr('create.button')}</button>
  <button
    class="pin"
    aria-pressed={pinned}
    title={$tr(pinned ? 'window.unpin' : 'window.pin')}
    aria-label={$tr(pinned ? 'window.unpin' : 'window.pin')}
    onclick={onpin}>📌</button
  >
</header>

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) var(--space-5) var(--space-1);
    --wails-draggable: drag;
  }
  .bar button,
  .bar kbd {
    --wails-draggable: no-drag;
  }
  h1 {
    margin: 0;
    font: 700 var(--hand-h1) var(--font-hand);
    white-space: nowrap;
  }
  .path {
    color: var(--pencil);
    font: var(--text-sm) var(--font-body);
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    min-width: 0;
  }
  .grow {
    flex: 1;
  }
  kbd {
    color: var(--pencil);
    font: var(--text-sm) var(--font-mono);
  }
  button {
    height: calc(var(--control) + var(--space-1));
    padding: 0 var(--space-3);
    border: var(--stroke) solid var(--grid-major);
    border-radius: var(--radius);
    background: none;
    color: var(--ink);
    font: 700 var(--hand-md) var(--font-hand);
    white-space: nowrap;
    cursor: pointer;
  }
  button:disabled {
    opacity: var(--dim);
    cursor: default;
  }
  .pin[aria-pressed='true'] {
    border-color: var(--accent);
  }
</style>
