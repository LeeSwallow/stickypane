<script lang="ts">
  // The window's title bar, and its drag region. Logo, the project name on
  // one line (cut with … when narrow; the full path is its tooltip, never a
  // wrapped line), the hotkey, then the caller's actions, the ⋯ menu (which
  // ends with Close window and Quit stickypane), and — where the OS gives no
  // buttons (chrome "own") — minimise and close. On macOS ("mac") it leaves
  // room at the start for the traffic lights.
  import type { Snippet } from 'svelte'
  import { tr } from '../../i18n'
  import Logo from '../doodle/Logo.svelte'
  import { look } from '../doodle/look.svelte'
  import { barLead, runWindowMenu, windowMenu, type Chrome, type WindowActions } from './chrome'
  import IconButton from './IconButton.svelte'
  import Kbd from './Kbd.svelte'
  import Menu from './Menu.svelte'
  import type { MenuItem } from './menu'
  import Separator from './Separator.svelte'
  import Toolbar from './Toolbar.svelte'
  import Tooltip from './Tooltip.svelte'

  let {
    name,
    path,
    hotkey,
    chrome = 'none',
    window: win,
    menu = [],
    menuOpen = $bindable(false),
    onmenu,
    actions,
    pinPath = false,
  }: {
    name: string
    /** the full folder, shown only in the name's tooltip */
    path?: string
    hotkey?: string
    chrome?: Chrome
    window: WindowActions
    /** the window's own items; Close window and Quit are added after them */
    menu?: MenuItem[]
    menuOpen?: boolean
    onmenu?: (id: string) => void
    actions?: Snippet
    /** show the path tooltip without hover (the reference page) */
    pinPath?: boolean
  } = $props()

  const items = $derived([...menu, ...windowMenu(chrome, $tr)].filter((m, i) => !(i === 0 && m.kind === 'separator')))

  function choose(id: string) {
    if (!runWindowMenu(id, win)) onmenu?.(id)
  }
</script>

<header class="bar {chrome}" data-titlebar style:padding-left="{barLead(chrome)}px">
  <div class="id">
    <Logo size={20} />
    {#if path}
      <Tooltip text={path} mono align="start" pinned={pinPath}>
        <!-- svelte-ignore a11y_no_noninteractive_tabindex: the name is focusable so the path tooltip is reachable by keyboard -->
        <span class="name" class:neat={look.style === 'neat'} tabindex="0">{name}</span>
      </Tooltip>
    {:else}
      <span class="name" class:neat={look.style === 'neat'}>{name}</span>
    {/if}
    {#if hotkey}<span class="hotkey" title={$tr('window.hotkey', { key: hotkey })}><Kbd keys={hotkey} /></span>{/if}
  </div>
  <span class="grow"></span>
  <Toolbar label={$tr('ui.windowActions')}>
    {@render actions?.()}
    <span class="anchor">
      <IconButton icon="more" label={$tr('ui.more')} aria-haspopup="menu" aria-expanded={menuOpen} align="end" onclick={() => (menuOpen = !menuOpen)} />
      <Menu items={items} bind:open={menuOpen} label={$tr('ui.windowActions')} onselect={choose} />
    </span>
    {#if chrome === 'own'}
      <Separator />
      <IconButton icon="minimise" label={$tr('ui.minimise')} align="end" onclick={win.minimise} />
      <IconButton icon="close" label={$tr('ui.closeWindow')} align="end" tone="danger" onclick={win.close} />
    {/if}
  </Toolbar>
</header>

<style>
  .bar {
    position: relative;
    z-index: var(--z-chrome);
    display: flex;
    align-items: center;
    gap: var(--space-3);
    height: var(--titlebar-h);
    padding-right: var(--space-2);
    box-sizing: border-box;
    border-bottom: 1px solid var(--line);
    background: var(--chrome);
    color: var(--ink);
    user-select: none;
    -webkit-user-select: none;
    --wails-draggable: drag;
  }
  /* macOS hidden-inset with its toolbar: the traffic lights sit centred in
     a 52px bar at the start; barLead() leaves them their room */
  .bar.mac {
    height: max(var(--titlebar-h), 52px);
  }
  .id {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-width: 0;
  }
  .id :global(.wrap) {
    min-width: 0;
  }
  .name {
    display: block;
    min-width: 0;
    max-width: 100%;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    border-radius: var(--radius-sm);
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
    --wails-draggable: no-drag;
  }
  .name.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .name:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .grow {
    flex: 1;
    min-width: var(--space-4);
  }
  .anchor {
    position: relative;
    display: inline-flex;
  }
</style>
