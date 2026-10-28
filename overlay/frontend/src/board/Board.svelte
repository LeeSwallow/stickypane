<script lang="ts">
  // The project window (03 §3.6): title bar, Glance line, tabs and the
  // grid of notes, or one note zoomed to fill the window. Data comes from
  // the shell's ProjectService; every change goes back through the
  // bridge, and the bridge's board.changed redraws the window.
  import './tokens.css'
  import { onMount, setContext } from 'svelte'
  import { setLanguage, t, tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import Paper from '../lib/doodle/Paper.svelte'
  import { watchSystemDark, type Mode } from '../lib/theme/mode'
  import { normalizePalette, type Palette } from '../lib/theme/palette'
  import { applyTheme } from '../lib/theme/tokens'
  import { chromeOf } from '../lib/ui/chrome'
  import { windowActions } from '../services/window'
  import EmptyState from '../lib/ui/EmptyState.svelte'
  import Toast from '../lib/ui/Toast.svelte'
  import CreateDialog from '../create/CreateDialog.svelte'
  import { act, arrange } from './actions'
  import { events, type Status } from '../services/board'
  import { projectService, type ProjectApi } from '../services/project'
  import { boardContext, type BoardCtx } from './context'
  import Glance from './Glance.svelte'
  import { glance, stalledCount } from './glance'
  import Grid from './Grid.svelte'
  import { connect } from './session'
  import { errorText } from '../services/errors'
  import { frameOf, keyAction } from './keys'
  import { columnsFor, shift, stepSize } from './layout'
  import { allNotes, findNote, resolveLink, type Board, type Note, type Tab } from './model'
  import NoteFrame from './NoteFrame.svelte'
  import Tabs from './Tabs.svelte'
  import { modeOf, paletteFor } from './theme'
  import TitleBar from './TitleBar.svelte'
  import {
    canGoBack, flip, followTab, goBack, goToPage, initView, isChanged, navigate, pageOf, seeAll, selectTab, zoomIn,
    type Seen, type View,
  } from './view'

  // zoom: a note to open zoomed on the first load; create: the create
  // dialog open on a shape (both for the dev screens page)
  let { api = projectService, zoom: startZoom, create: startCreate }: { api?: ProjectApi; zoom?: string; create?: string } = $props()

  let status = $state<Status | null>(null)
  let board = $state<Board | null>(null)
  let view = $state<View>(initView(''))
  let seen = $state<Seen>({})
  let filter = $state<string | null>(null)
  let now = $state(new Date())
  let width = $state(1000)
  // the grid's own width: what a note's fraction is a fraction of
  let gridWidth = $state(1000)
  let mode = $state<Mode>('system')
  let systemDark = $state(false)
  let hotkey = $state('')
  let failed = $state('')
  // the note a focused frame holds: what ⋯ "편집기로 열기" opens when none is zoomed
  let focused = $state<string | null>(null)
  // "+ 생성": the dialog, open from the title bar or with a
  let creating = $state(false)
  // a short word at the bottom (made, archived, deleted with 되돌리기)
  let notice = $state<{ message: string; action?: { label: string; run: () => void } } | null>(null)

  const chrome = chromeOf(location.search)
  const win = windowActions('project')

  const bridgePalette = $derived.by((): Palette | null => {
    try {
      return board ? normalizePalette(board.palette) : null
    } catch {
      return null
    }
  })
  const palette = $derived(bridgePalette && board ? paletteFor(bridgePalette, board.settings.theme, mode, systemDark) : null)

  $effect(() => {
    if (!palette) return
    applyTheme(palette, palette.dark)
    look.palette = palette
    look.dark = palette.dark
  })
  $effect(() => {
    if (!board) return
    look.style = board.settings.style === 'neat' ? 'neat' : 'doodle'
    const lang = board.settings.language?.resolved
    setLanguage(lang === 'ko' ? 'ko' : 'en')
  })

  const tab: Tab | null = $derived(board ? (board.tabs.find((t) => t.name === view.tab) ?? board.tabs[0] ?? null) : null)
  const open = $derived(tab ? tab.notes.filter((n) => n.open) : [])
  const closed = $derived(tab ? tab.notes.filter((n) => !n.open) : [])
  const zoomed: Note | null = $derived(board && view.zoom ? findNote(board.tabs, view.zoom) : null)
  const columns = $derived(columnsFor(width))
  const tags = $derived(board ? glance(board.tabs, now) : [])
  const stalled = $derived(board ? stalledCount(board.tabs) : 0)
  const files = $derived(board ? allNotes(board.tabs).map((n) => n.name) : [])
  const editTarget = $derived.by(() => {
    const n = board ? findNote(board.tabs, zoomed?.name ?? focused ?? '') : null
    return n && !n.pages && !n.error ? n.name : null
  })

  // --- loading -----------------------------------------------------------

  let loading: Promise<void> | null = null
  let again = false

  /** refresh reads the status and the board again; calls during one are folded into one more. */
  function refresh(): Promise<void> {
    if (loading) {
      again = true
      return loading
    }
    loading = (async () => {
      do {
        again = false
        try {
          status = await api.status()
          if (status.state !== 'ready') continue
          const b = await api.load()
          const before = board?.settings.tab ?? null
          board = b
          view = before === null ? (startZoom ? zoomIn(initView(b.settings.tab), startZoom) : initView(b.settings.tab)) : followTab(view, before, b.settings.tab)
          seen = seeAll(seen, allNotes(b.tabs))
          failed = ''
        } catch (e) {
          failed = String(e)
        }
      } while (again)
      loading = null
    })()
    return loading
  }

  const tabLabel = (x: Tab) => x.title || x.name.replace(/^\d+[-_. ]*/, '') || t('board.tabRoot')

  /** created opens the new note, shows its tab and puts the focus on it. */
  async function created(name: string) {
    await api.arrange({ note: name, open: true }).catch((e) => (failed = errorText(e)))
    await refresh()
    const at = locate(name)
    if (at) view = selectTab(view, at.tab)
    notice = { message: t('create.created', { name: findNote(board?.tabs ?? [], name)?.title || name }) }
    requestAnimationFrame(() => {
      const f = document.querySelector<HTMLElement>(`[data-frame="${CSS.escape(name)}"]`)
      f?.scrollIntoView({ block: 'nearest' })
      f?.focus()
    })
  }

  const update = (patch: (b: Board) => Board) => {
    if (board) board = patch(board)
  }

  // --- what notes ask of the window ---------------------------------------

  /** where a note is: its tab, and its book when it is a page */
  function locate(name: string): { tab: string; book: Note | null; page: number } | null {
    if (!board) return null
    for (const t of board.tabs)
      for (const n of t.notes) {
        if (n.name === name) return { tab: t.name, book: null, page: 0 }
        const i = (n.pages ?? []).findIndex((p) => p.name === name)
        if (i >= 0) return { tab: t.name, book: n, page: i }
      }
    return null
  }

  const ctx: BoardCtx = {
    find: (link) => (board ? findNote(board.tabs, resolveLink(link, files)) : null),
    navigate(link) {
      const name = resolveLink(link, files)
      const at = name ? locate(name) : null
      if (!at) return
      if (at.book) {
        view = goToPage(navigate(view, at.tab, at.book.name), at.book.name, at.page, at.book.pages?.length ?? 0)
      } else view = navigate(view, at.tab, name)
    },
    act: (note, action) =>
      act(api, note, action, (error) => (failed = t('note.actFailed', { title: note.title, error })), refresh, update),
    async arrange(name, patch) {
      try {
        await arrange(api, name, patch, update)
      } catch (e) {
        failed = String(e)
        await refresh()
      }
    },
    zoom: (name) => (view = zoomIn(view, name)),
    back: () => (view = goBack(view)),
    canBack: () => canGoBack(view),
    flip(book, delta, to) {
      const n = board ? findNote(board.tabs, book) : null
      const count = n?.pages?.length ?? 0
      view = to === undefined ? flip(view, book, delta, count) : goToPage(view, book, to, count)
    },
    pageOf: (book, count) => pageOf(view, book, count),
    changed: (n) => isChanged(seen, n),
    seen(name) {
      const n = board ? findNote(board.tabs, name) : null
      if (n && seen[name] !== n.modTime) seen = { ...seen, [name]: n.modTime }
    },
    filter: () => filter,
    now: () => now,
    edit: (name) => void api.open(name).catch((err) => (failed = String(err))),
  }
  setContext(boardContext, ctx)

  // --- keys --------------------------------------------------------------

  function onkey(e: KeyboardEvent) {
    if (!board || !tab) return
    const t = e.target as HTMLElement
    const editing = t.matches?.('input, textarea, select, [contenteditable]') ?? false
    // a dialog's keys are its own
    if (t.closest?.('[role="dialog"], [role="alertdialog"]')) return
    // Enter and Space on a button or a link are that control's own
    if ((e.key === 'Enter' || e.key === ' ') && t.closest?.('button, a')) return
    // the note keys act on: the zoomed one, or the frame holding the focus
    // (never the note a focused link names)
    const name = zoomed?.name ?? frameOf(document.activeElement)
    const note = name ? findNote(board.tabs, name) : null
    const book = !!note?.pages?.length
    const a = keyAction({ key: e.key, metaKey: e.metaKey, ctrlKey: e.ctrlKey, altKey: e.altKey, shiftKey: e.shiftKey, editing }, { zoomed: !!zoomed, canBack: canGoBack(view), book })
    if (!a) return
    switch (a.type) {
      case 'back':
        ctx.back()
        break
      case 'zoom':
        if (!note || zoomed) return
        ctx.zoom(note.name)
        break
      case 'size':
        if (!note || note.error) return
        ctx.arrange(note.name, { size: stepSize(note.size, gridWidth, a.dir) })
        break
      case 'page':
        if (!note?.pages?.length) return
        ctx.flip(note.name, a.delta)
        break
      case 'tab':
        if (!board.tabs[a.index]) return
        view = selectTab(view, board.tabs[a.index].name)
        break
      case 'focus': {
        if (zoomed) return
        const frames = [...document.querySelectorAll<HTMLElement>('.notes [data-frame]')]
        const i = frames.findIndex((f) => f.dataset.frame === name)
        frames[Math.max(0, Math.min(frames.length - 1, i < 0 ? 0 : i + a.delta))]?.focus()
        break
      }
      case 'order': {
        if (!note) return
        const order = shift(tab.notes.map((n) => n.name), note.name, a.delta)
        if (!order) return
        const tabName = tab.name
        update((b) => ({
          ...b,
          tabs: b.tabs.map((x) => (x.name === tabName ? { ...x, notes: order.map((o) => x.notes.find((n) => n.name === o)!) } : x)),
        }))
        api.order(tabName, order).catch((err) => (failed = String(err))).finally(refresh)
        requestAnimationFrame(() => document.querySelector<HTMLElement>(`[data-frame="${CSS.escape(note.name)}"]`)?.focus())
        break
      }
      case 'edit':
        if (!note || note.pages) return
        ctx.edit?.(note.name)
        break
      case 'create':
        creating = true
        break
      default:
        return
    }
    e.preventDefault()
  }

  const tabChanged = (t: Tab) => t.notes.some((n) => isChanged(seen, n) || (n.pages ?? []).some((p) => isChanged(seen, p)))

  async function pin() {
    if (!status) return
    await api.pin(!status.pinned).catch((err) => (failed = String(err)))
    status = await api.status()
  }

  onMount(() => {
    if (startCreate) creating = true
    const offs: (() => void)[] = [watchSystemDark((d) => (systemDark = d))]
    api
      .prefs()
      .then((p) => {
        mode = modeOf(p.mode)
        hotkey = p.hotkey ?? ''
      })
      .catch(() => {})
    refresh()
    // the bridge's events, from as soon as the window id is known, even
    // when the first load failed
    offs.push(
      connect({
        status: () => api.status(),
        bus: events,
        onChanged(data) {
          // A10's motion listens for this and picks a movement from the events.
          window.dispatchEvent(new CustomEvent('stickypane:board-changed', { detail: data }))
          refresh()
        },
        onRestarted: () => refresh(),
        onStatus: () => refresh(),
      }),
    )
    // elapsed times are words ("4m ago"): a new minute rewrites them
    const tick = setInterval(() => (now = new Date()), 60_000)
    offs.push(() => clearInterval(tick))
    return () => offs.forEach((off) => off())
  })

  const stateText: Record<string, string> = { loading: 'board.loading', missing: 'board.missing', too_old: 'board.tooOld' }
</script>

<svelte:window onkeydown={onkey} />

<div class="window" bind:clientWidth={width}>
  <TitleBar
    title={board?.title || (status?.path ?? '').split('/').filter(Boolean).pop() || ''}
    path={status?.path ?? ''}
    {hotkey}
    {chrome}
    window={win}
    pinned={status?.pinned ?? false}
    canEdit={!!editTarget}
    onpin={pin}
    onedit={() => editTarget && ctx.edit?.(editTarget)}
    oncreate={status?.state === 'ready' && board ? () => (creating = true) : undefined}
  />
  {#if status?.state === 'ready' && board}
    <Glance {tags} {stalled} {filter} {now} onfilter={(w) => (filter = w)} />
    {#if !zoomed}
      <Tabs
        tabs={board.tabs}
        current={tab?.name ?? ''}
        changed={tabChanged}
        {closed}
        onselect={(name) => (view = selectTab(view, name))}
        onopen={(name) => ctx.arrange(name, { open: true })}
      />
    {/if}
  {/if}
  <div class="scroll">
    <Paper>
      {#if !status || status.state !== 'ready' || !board}
        <div class="state" role="status">
          <EmptyState title={$tr(status ? (stateText[status.state] ?? 'board.cannotOpen') : 'board.loading')} body={status?.error} art={!!status && status.state !== 'loading'} />
        </div>
      {:else if zoomed}
        <!-- a click on the paper around a zoomed note goes back; Esc does too -->
        <main class="zoom" role="presentation" onclick={(e) => e.target === e.currentTarget && ctx.back()}>
          {#key zoomed.name}
            <NoteFrame note={zoomed} zoomed narrow={columns === 1} width={gridWidth} />
          {/key}
        </main>
      {:else if open.length}
        <main class="notes" onfocusin={(e) => (focused = frameOf(e.target as Element))}>
          <div bind:clientWidth={gridWidth}>
            <Grid notes={open} width={gridWidth} />
          </div>
        </main>
      {:else}
        <div class="state">
          <EmptyState title={$tr(tab && tab.notes.length ? 'board.noneOpen' : 'board.emptyTitle')} body={tab && tab.notes.length ? undefined : $tr('board.emptyHint')} />
        </div>
      {/if}
    </Paper>
  </div>
  {#if failed}
    <div class="toasts">
      <Toast message={failed} tone="bad" icon="warn" onclose={() => (failed = '')} />
    </div>
  {:else if notice}
    <div class="toasts">
      {#key notice}
        <Toast message={notice.message} action={notice.action} timeout={6000} onclose={() => (notice = null)} />
      {/key}
    </div>
  {/if}
  {#if board}
    <CreateDialog
      bind:open={creating}
      {api}
      tabs={board.tabs.map((x) => ({ name: x.name, label: tabLabel(x) }))}
      tab={tab?.name ?? ''}
      start={startCreate}
      oncreated={created}
    />
  {/if}
</div>

<style>
  /* the window scrolls its notes; the page itself never does (Mermaid
     measures a diagram in a hidden element at the end of the body) */
  :global(html),
  :global(body) {
    overflow: hidden;
  }
  .window {
    position: relative;
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--paper);
    color: var(--ink);
  }
  .scroll {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    overflow: auto;
  }
  .scroll > :global(.paper) {
    flex: 1 0 auto;
    display: flex;
    flex-direction: column;
  }
  .notes {
    padding: var(--space-6) var(--space-5) var(--space-10);
  }
  .zoom {
    flex: 1;
    display: flex;
    flex-direction: column;
    min-height: 0;
    max-height: 100%;
    padding: var(--space-4) var(--space-5) var(--space-5);
    box-sizing: border-box;
  }
  .zoom > :global(.frame) {
    flex: 1;
    min-height: 0;
  }
  .state {
    display: grid;
    place-items: center;
    flex: 1;
  }
  .toasts {
    position: absolute;
    left: 0;
    right: 0;
    bottom: var(--space-5);
    z-index: var(--z-toast);
    display: flex;
    justify-content: center;
    padding: 0 var(--space-4);
    pointer-events: none;
  }
  .toasts > :global(*) {
    pointer-events: auto;
  }
</style>
