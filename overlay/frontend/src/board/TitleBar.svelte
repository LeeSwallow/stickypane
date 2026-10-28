<script lang="ts">
  // The board window's title bar (03 §3.6, §3.11.6): the design system's
  // TitleBar — traffic lights on macOS, the project's name on one line with
  // its path only in a tooltip, the hotkey — with the board's actions: grid
  // or canvas (the canvas comes with A7b), search (later), "+ 만들기" (A9),
  // the window's pin, and ⋯ with "편집기로 열기" before the window's own
  // "창 닫기" and "stickypane 종료".
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import type { Chrome, WindowActions } from '../lib/ui/chrome'
  import IconButton from '../lib/ui/IconButton.svelte'
  import { hotkeyLabel } from '../lib/ui/keys'
  import type { MenuItem } from '../lib/ui/menu'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import UiTitleBar from '../lib/ui/TitleBar.svelte'

  let {
    title,
    path,
    hotkey = '',
    pinned = false,
    chrome,
    window: win,
    canEdit = false,
    onpin,
    onedit,
    oncreate,
  }: {
    title: string
    path: string
    hotkey?: string
    pinned?: boolean
    chrome: Chrome
    window: WindowActions
    /** a note is zoomed or focused: "편집기로 열기" opens it */
    canEdit?: boolean
    onpin: () => void
    onedit: () => void
    oncreate?: () => void
  } = $props()

  let view = $state('grid')
  const views = $derived([
    { value: 'grid', label: $tr('ui.view.grid'), icon: 'grid' as const },
    { value: 'canvas', label: $tr('ui.view.canvasSoon'), icon: 'canvas' as const, disabled: true },
  ])
  const menu = $derived<MenuItem[]>([{ id: 'edit', label: $tr('note.openInEditor'), icon: 'edit-external', shortcut: 'E', disabled: !canEdit }])
</script>

<UiTitleBar name={title} {path} hotkey={hotkeyLabel(hotkey)} {chrome} window={win} {menu} onmenu={(id) => id === 'edit' && onedit()}>
  {#snippet actions()}
    <SegmentedControl label={$tr('ui.view')} bind:value={view} iconOnly size="sm" options={views} />
    <IconButton icon="search" label={$tr('ui.searchSoon')} disabled />
    <Button variant="primary" size="sm" icon="plus" disabled={!oncreate} title={oncreate ? undefined : $tr('create.soon')} onclick={() => oncreate?.()}>
      {$tr('create.make')}
    </Button>
    <IconButton
      icon="pin"
      label={pinned ? $tr('window.unpin') : $tr('window.pin')}
      pressed={pinned}
      crayon={pinned}
      align="end"
      onclick={onpin}
    />
  {/snippet}
</UiTitleBar>
