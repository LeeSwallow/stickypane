<script lang="ts">
  // A note torn off its board (03 §3.6.1, ADR-0017): one note in a small
  // window of its own, anywhere on the screen, over the same bridge as its
  // board's window. It draws and acts as on the board (the same frame and
  // shapes, through the same context); its bar is where the window is
  // carried, kept on top, and put back. Its ⋯ runs a script, sends a
  // request, moves, archives and deletes as the board's does, after the
  // same questions. The shell closes it when the note
  // leaves the board (archived, deleted, moved away).
  import '../board/tokens.css'
  import { onMount, setContext, tick as drawn } from 'svelte'
  import { boardLanguage, setLanguage, t, tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { noteOf } from '../lib/shell'
  import { watchSystemDark, type Mode } from '../lib/theme/mode'
  import { normalizePalette, type Palette } from '../lib/theme/palette'
  import { agentColor, applyTheme } from '../lib/theme/tokens'
  import { director } from '../lib/motion/director'
  import { level, motion, watchReducedMotion } from '../lib/motion/level.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Confirm from '../lib/ui/Confirm.svelte'
  import Dialog from '../lib/ui/Dialog.svelte'
  import EmptyState from '../lib/ui/EmptyState.svelte'
  import Input from '../lib/ui/Input.svelte'
  import IconButton from '../lib/ui/IconButton.svelte'
  import Toast from '../lib/ui/Toast.svelte'
  import { events, type Bus, type Status } from '../services/board'
  import { errorText } from '../services/errors'
  import { projectService, type ProjectApi } from '../services/project'
  import { act, arrange } from '../board/actions'
  import { boardContext, noteHex, type BoardCtx } from '../board/context'
  import { allNotes, findNote, resolveLink, type Board } from '../board/model'
  import NoteFrame from '../board/NoteFrame.svelte'
  import * as manage from '../board/manage'
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
  // a short word at the bottom (run, sent, archived, deleted with 되돌리기)
  let notice = $state<manage.Said | null>(null)
  // a question before a delete, a run or a send; the note being moved
  let asking = $state<manage.Asking | null>(null)
  let moving = $state(false)
  let moveTo = $state('')
  let mode = $state<Mode>('system')
  // the app's language: what a board with none of its own speaks
  let appLanguage = $state('')
  let systemDark = $state(false)
  let now = $state(new Date())
  let width = $state(360)
  // the page a book is open at, in this window
  let pages = $state<Record<string, number>>({})
  // where the note is drawn: what moves when it changes (lib/motion)
  let body: HTMLElement | undefined = $state()

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
  // every theme, from the bridge's themes.list: the other half of a pair
  let themes = $state<Palette[]>([])
  let themesAsked = false
  $effect(() => {
    if (status?.state !== 'ready' || themesAsked) return
    themesAsked = true
    api.themes().then(
      (t) => (themes = t),
      () => {},
    )
  })
  const palette = $derived(bridgePalette && board ? paletteFor(bridgePalette, board.settings.theme, mode, systemDark, themes) : null)
  $effect(() => {
    if (!palette) return
    applyTheme(palette, palette.dark)
    look.palette = palette
    look.dark = palette.dark
  })
  $effect(() => {
    if (!board) return
    look.style = board.settings.style === 'neat' ? 'neat' : 'doodle'
    setLanguage(boardLanguage(board.settings.language, appLanguage))
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

  const deps: manage.ManageDeps = {
    get api() {
      return api
    },
    refresh,
    say: (s) => (notice = s),
    fail: (m) => (failed = m),
  }
  const question = $derived(asking ? manage.question(asking, board?.title ?? '') : null)

  async function answer() {
    const a = asking
    asking = null
    if (a) await manage.answer(deps, a)
  }

  async function moveNote() {
    moving = false
    if (note && moveTo.trim()) await manage.move(deps, note, moveTo)
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
    manage(n, what, request) {
      if (what === 'archive') void manage.archive(deps, n)
      else if (what === 'move') {
        moving = true
        moveTo = ''
      } else asking = { what, note: n, request }
    },
  }
  setContext(boardContext, ctx)

  // a board.changed moves what changed in this note, as on the board (04 §4.3)
  const stage = director({
    root: () => body ?? null,
    board: () => board,
    refresh,
    settle: drawn,
    visible: () => new Set(note ? [note.name] : []),
    look: () => ({ style: look.style, level: level() }),
    colorOf: (who) => (look.palette ? agentColor(who, look.palette.notes) : undefined),
    hidden: () => document.visibilityState === 'hidden',
  })

  onMount(() => {
    const offs: (() => void)[] = [watchSystemDark((d) => (systemDark = d)), watchReducedMotion(), () => stage.stop()]
    api
      .prefs()
      .then((p) => {
        mode = modeOf(p.mode)
        motion.setting = p.motion ?? ''
        appLanguage = p.language ?? ''
      })
      .catch(() => {})
    refresh()
    offs.push(
      connect({ status: () => api.status(), bus, onChanged: (data) => void stage.changed(data), onRestarted: () => refresh(), onStatus: () => refresh() }),
    )
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
  <main class="body" bind:this={body}>
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
  {:else if notice}
    <div class="toasts">
      {#key notice}
        <Toast message={notice.message} action={notice.action} timeout={6000} onclose={() => (notice = null)} />
      {/key}
    </div>
  {/if}
  {#if asking && question}
    <Confirm open title={question.title} message={question.body} confirmLabel={question.yes} danger onconfirm={answer} oncancel={() => (asking = null)} />
  {/if}
  {#if moving && note}
    {@const targets = manage.moveTargets(board?.tabs ?? [], note.tab)}
    <Dialog open title={t('note.moveTitle', { title: note.title })} width={320} onclose={() => (moving = false)}>
      <form class="move" onsubmit={(e) => (e.preventDefault(), moveNote())}>
        <Input label={$tr('note.moveTo')} hint={$tr('note.moveHint')} bind:value={moveTo} data-autofocus />
        {#if targets.length}
          <div class="targets">
            {#each targets as to (to)}
              <Button size="sm" variant="secondary" icon="tab" onclick={() => (moveTo = to)}>{to === '.' ? $tr('note.moveTop') : to}</Button>
            {/each}
          </div>
        {/if}
      </form>
      {#snippet actions()}
        <Button variant="ghost" onclick={() => (moving = false)}>{$tr('ui.cancel')}</Button>
        <Button variant="primary" disabled={!moveTo.trim()} onclick={moveNote}>{$tr('note.move').replace(/…$/, '')}</Button>
      {/snippet}
    </Dialog>
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
  .move {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  .targets {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
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
