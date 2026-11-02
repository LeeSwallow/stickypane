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
    ontidy,
    onreset,
    host = '',
    ontearall,
    ontearboard,
    onputbackall,
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
    /** ⋯ → 정렬: the canvas's notes laid out again (03 §3.6.2) */
    ontidy?: () => void
    /** ⋯ → 배치 초기화 */
    onreset?: () => void
    /** the host of a board on another computer: said by the name, and no editor here */
    host?: string
    /** ⋯ → every note of the tab in its own window */
    ontearall?: () => void
    /** ⋯ → every note of every tab in its own window */
    ontearboard?: () => void
    /** ⋯ → every torn-off note back on the board */
    onputbackall?: () => void
  } = $props()

  const views = $derived([
    { value: 'canvas', label: $tr('ui.view.canvas'), icon: 'canvas' as const, disabled: !onmode },
    { value: 'grid', label: $tr('ui.view.grid'), icon: 'grid' as const, disabled: !onmode },
  ])
  const menu = $derived<MenuItem[]>([
    ...(mode === 'canvas' && ontidy ? [{ id: 'tidy', label: $tr('board.tidy'), icon: 'grid' as const, shortcut: '⌥⌘L' }] : []),
    ...(onreset ? [{ id: 'reset', label: $tr('board.reset'), icon: 'restore' as const }] : []),
    ...(mode === 'canvas' && onsnap ? [{ id: 'snap', label: $tr(snap ? 'canvas.snapOn' : 'canvas.snapOff'), icon: snap ? ('check' as const) : undefined }] : []),
    ...(ontidy || onreset ? [{ kind: 'separator' as const, id: 'sep-layout' }] : []),
    ...(ontearall ? [{ id: 'tearAll', label: $tr('board.tearAll'), icon: 'tear-off' as const, shortcut: '⌥⌘T' }] : []),
    ...(ontearboard ? [{ id: 'tearBoard', label: $tr('board.tearBoard'), icon: 'tear-off' as const, shortcut: '⇧⌥⌘T' }] : []),
    ...(onputbackall ? [{ id: 'putBackAll', label: $tr('board.putBackAll'), icon: 'restore' as const }] : []),
    ...(ontearall || ontearboard || onputbackall ? [{ kind: 'separator' as const, id: 'sep-torn' }] : []),
    ...(host ? [] : [{ id: 'edit', label: $tr('note.openInEditor'), icon: 'edit-external' as const, shortcut: 'E', disabled: !canEdit }]),
    { id: 'settings', label: $tr('settings.open'), icon: 'settings', shortcut: '⌘,', disabled: !onsettings },
  ])
  // the hotkey is said in the name's tooltip, not on the bar (#63)
  const tip = $derived(hotkey ? `${path} — ${$tr('window.hotkeyTip', { key: hotkeyLabel(hotkey) })}` : path)
  function choose(id: string) {
    if (id === 'edit') onedit()
    else if (id === 'settings') onsettings?.()
    else if (id === 'snap') onsnap?.(!snap)
    else if (id === 'tidy') ontidy?.()
    else if (id === 'reset') onreset?.()
    else if (id === 'tearAll') ontearall?.()
    else if (id === 'tearBoard') ontearboard?.()
    else if (id === 'putBackAll') onputbackall?.()
  }
</script>

<UiTitleBar name={title} path={tip} {chrome} window={win} {menu} onmenu={choose} {host} hostTip={host ? $tr('board.remoteTip', { host }) : ''}>
  {#snippet actions()}
    <SegmentedControl label={$tr('ui.view')} value={mode} iconOnly size="sm" options={views} onchange={(v) => onmode?.(v as LayoutMode)} />
    <Button variant="primary" size="sm" icon="plus" disabled={!oncreate} title={oncreate ? undefined : $tr('create.soon')} onclick={() => oncreate?.()}>
      {$tr('create.new')}
    </Button>
    {#if ontearboard || onputbackall}
      <IconButton
        icon={ontearboard ? 'tear-off' : 'restore'}
        label={ontearboard ? $tr('board.tearEverything') : $tr('board.putBackAll')}
        shortcut={ontearboard ? '⇧⌥⌘T' : undefined}
        align="end"
        onclick={() => (ontearboard ? ontearboard() : onputbackall?.())}
      />
    {/if}
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
