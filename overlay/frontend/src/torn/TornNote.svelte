<script lang="ts">
  // A note torn off its board (03 §3.6.1, ADR-0017): one note in a small
  // window of its own, anywhere on the screen, over the same bridge as its
  // board's window. It draws and acts as on the board (the same frame and
  // shapes, through the same context); its bar is where the window is
  // carried, kept on top, and put back. The shell closes it when the note
  // leaves the board (archived, deleted, moved away).
  import '../board/tokens.css'
  import { onMount, setContext } from 'svelte'
  import { setLanguage, t, tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { noteOf } from '../lib/shell'
  import { watchSystemDark, type Mode } from '../lib/theme/mode'
  import { normalizePalette, type Palette } from '../lib/theme/palette'
  import { applyTheme } from '../lib/theme/tokens'
  import EmptyState from '../lib/ui/EmptyState.svelte'
  import IconButton from '../lib/ui/IconButton.svelte'
  import Toast from '../lib/ui/Toast.svelte'
  import { events, type Bus, type Status } from '../services/board'
  import { errorText } from '../services/errors'
  import { projectService, type ProjectApi } from '../services/project'
  import { act, arrange } from '../board/actions'
  import { boardContext, noteHex, type BoardCtx } from '../board/context'
  import { allNotes, findNote, resolveLink, type Board } from '../board/model'
  import NoteFrame from '../board/NoteFrame.svelte'
  import { connect } from '../board/session'
  import { modeOf, paletteFor } from '../board/theme'

  // bus: the shell's events (a test passes its own)
  let {
    api = projectService,
    bus = events,
    note: start = noteOf(location.search),
  }: { api?: ProjectApi; bus?: Bus; note?: string } = $props()

  let status = $state<Status | null>(null)
  let board = $state<Board | null>(null)
  let failed = $state('')
  let mode = $state<Mode>('system')
  let systemDark = $state(false)
  let now = $state(new Date())
  let width = $state(360)
  // the page a book is open at, in this window
  let pages = $state<Record<string, number>>({})

  // the shell names the note again after a move
  const name = $derived(status?.note || start)
  const note = $derived(board ? findNote(board.tabs, name) : null)
  const files = $derived(board ? allNotes(board.tabs).map((n) => n.name) : [])

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
    setLanguage(board.settings.language?.resolved === 'ko' ? 'ko' : 'en')
  })
  $effect(() => {
    document.title = note?.title || name
  })

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
          if (status.state === 'ready') board = await api.load()
          failed = ''
        } catch (e) {
          failed = errorText(e)
        }
      } while (again)
      loading = null
    })()
    return loading
  }

  const update = (patch: (b: Board) => Board) => {
    if (board) board = patch(board)
  }

  async function putBack() {
    await api.putBack().catch((e) => (failed = errorText(e)))
  }

  async function pin() {
    if (!status) return
    await api.pin(!status.pinned).catch((e) => (failed = errorText(e)))
    status = await api.status()
  }

  const ctx: BoardCtx = {
    find: (link) => (board ? findNote(board.tabs, resolveLink(link, files)) : null),
    // a link opens the note it names torn off too, beside this one
    navigate(link) {
      const n = resolveLink(link, files)
      if (n) api.tearOff(n).catch((e) => (failed = errorText(e)))
    },
    act: (n, a) => act(api, n, a, (error) => (failed = t('note.actFailed', { title: n.title, error })), refresh, update),
    async arrange(n, patch) {
      try {
        await arrange(api, n, patch, update)
      } catch (e) {
        failed = errorText(e)
        await refresh()
      }
    },
    zoom: () => {},
    back: () => {},
    canBack: () => false,
    flip(book, delta, to) {
      const count = (board ? findNote(board.tabs, book)?.pages?.length : 0) ?? 0
      const at = to ?? (pages[book] ?? 0) + delta
      pages = { ...pages, [book]: Math.max(0, Math.min(count - 1, at)) }
    },
    pageOf: (book, count) => Math.min(pages[book] ?? 0, Math.max(0, count - 1)),
    changed: () => false,
    seen: () => {},
    filter: () => null,
    now: () => now,
    edit: (n) => void api.open(n).catch((e) => (failed = errorText(e))),
    putBack: () => void putBack(),
  }
  setContext(boardContext, ctx)

  onMount(() => {
    const offs: (() => void)[] = [watchSystemDark((d) => (systemDark = d))]
    api
      .prefs()
      .then((p) => (mode = modeOf(p.mode)))
      .catch(() => {})
    refresh()
    offs.push(connect({ status: () => api.status(), bus, onChanged: () => refresh(), onRestarted: () => refresh(), onStatus: () => refresh() }))
    const tick = setInterval(() => (now = new Date()), 60_000)
    offs.push(() => clearInterval(tick))
    return () => offs.forEach((off) => off())
  })

  const stateText: Record<string, string> = { loading: 'board.loading', missing: 'board.missing', too_old: 'board.tooOld' }
</script>

<div class="torn" bind:clientWidth={width} style:--tape={note ? noteHex(note.color, look.palette) : 'var(--line)'} data-torn={name}>
  <header class="bar" data-titlebar>
    <span class="tape" aria-hidden="true"></span>
    <span class="grow"></span>
    <IconButton
      icon="pin"
      size="sm"
      label={status?.pinned ? $tr('window.unpin') : $tr('window.pin')}
      pressed={status?.pinned ?? false}
      crayon={status?.pinned ?? false}
      align="end"
      onclick={pin}
    />
    <IconButton icon="restore" size="sm" label={$tr('note.putBack')} align="end" onclick={putBack} />
  </header>
  <main class="body">
    {#if note}
      {#key note.name}
        <NoteFrame {note} free narrow small={width < 300} {width} />
      {/key}
    {:else if !status || status.state !== 'ready' || !board}
      <div class="state" role="status">
        <EmptyState title={$tr(status ? (stateText[status.state] ?? 'board.cannotOpen') : 'board.loading')} body={status?.error} art={false} />
      </div>
    {:else}
      <div class="state" role="status">
        <EmptyState title={$tr('torn.gone')} art={false} />
      </div>
    {/if}
  </main>
  {#if failed}
    <div class="toasts">
      <Toast message={failed} tone="bad" icon="warn" onclose={() => (failed = '')} />
    </div>
  {/if}
</div>

<style>
  :global(html),
  :global(body) {
    overflow: hidden;
  }
  .torn {
    position: relative;
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--paper);
    color: var(--ink);
  }
  /* the window is carried by its bar */
  .bar {
    --wails-draggable: drag;
    position: relative;
    display: flex;
    align-items: center;
    gap: var(--space-1);
    flex: none;
    height: var(--control-md);
    padding: 0 var(--space-2);
  }
  .tape {
    position: absolute;
    left: 50%;
    top: var(--space-1);
    width: 64px;
    height: var(--space-4);
    transform: translateX(-50%) rotate(-2deg);
    border-radius: 2px;
    background: color-mix(in srgb, var(--tape) 55%, transparent);
  }
  .grow {
    flex: 1;
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
    flex-direction: column;
    padding: 0 var(--space-3) var(--space-3);
  }
  .body > :global(.frame) {
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
    bottom: var(--space-3);
    display: flex;
    justify-content: center;
    padding: 0 var(--space-3);
  }
</style>
