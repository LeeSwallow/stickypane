<script lang="ts">
  // The board window's title bar (03 §3.6, §3.11.6): the design system's
  // TitleBar — traffic lights on macOS, the project's name on one line with
  // its path only in a tooltip, the hotkey — with the board's actions: grid
  // or canvas (03 §3.6.1), search (later), "+ 생성",
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
  import type { LayoutMode } from '../services/canvas'

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
    onsettings,
    mode = 'grid',
    onmode,
    snap = true,
    onsnap,
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
    /** ⋯ → 설정… (and ⌘,) */
    onsettings?: () => void
    /** the board's layout: notes anywhere, or the terminal's grid */
    mode?: LayoutMode
    onmode?: (m: LayoutMode) => void
    /** the canvas's notes land on its grid (⋯ → snap, on the canvas) */
    snap?: boolean
    onsnap?: (on: boolean) => void
  } = $props()

  const views = $derived([
    { value: 'canvas', label: $tr('ui.view.canvas'), icon: 'canvas' as const, disabled: !onmode },
    { value: 'grid', label: $tr('ui.view.grid'), icon: 'grid' as const, disabled: !onmode },
  ])
  const menu = $derived<MenuItem[]>([
    { id: 'edit', label: $tr('note.openInEditor'), icon: 'edit-external', shortcut: 'E', disabled: !canEdit },
    { id: 'settings', label: $tr('settings.open'), icon: 'settings', shortcut: '⌘,', disabled: !onsettings },
    ...(mode === 'canvas' && onsnap ? [{ id: 'snap', label: $tr(snap ? 'canvas.snapOn' : 'canvas.snapOff'), icon: snap ? ('check' as const) : undefined }] : []),
  ])
</script>

<UiTitleBar name={title} {path} hotkey={hotkeyLabel(hotkey)} {chrome} window={win} {menu} onmenu={(id) => (id === 'edit' ? onedit() : id === 'settings' ? onsettings?.() : id === 'snap' && onsnap?.(!snap))}>
  {#snippet actions()}
    <SegmentedControl label={$tr('ui.view')} value={mode} iconOnly size="sm" options={views} onchange={(v) => onmode?.(v as LayoutMode)} />
    <IconButton icon="search" label={$tr('ui.searchSoon')} disabled />
    <Button variant="primary" size="sm" icon="plus" disabled={!oncreate} title={oncreate ? undefined : $tr('create.soon')} onclick={() => oncreate?.()}>
      {$tr('create.new')}
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
