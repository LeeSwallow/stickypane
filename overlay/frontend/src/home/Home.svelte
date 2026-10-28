<script lang="ts">
  // The home window: the boards on this machine, straight away (R-32).
  import { onMount } from 'svelte'
  import { chooseLanguage, resolveAuto, setLanguage, tr, type Choice } from '../i18n'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import Paper from '../lib/doodle/Paper.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Language from '../settings/Language.svelte'
  import { fallbackPalette } from '../lib/theme/fallback'
  import { resolveDark, watchSystemDark, type Mode } from '../lib/theme/mode'
  import { applyTheme } from '../lib/theme/tokens'
  import Button from '../lib/ui/Button.svelte'
  import { chromeOf } from '../lib/ui/chrome'
  import { windowActions } from '../services/window'
  import Confirm from '../lib/ui/Confirm.svelte'
  import TitleBar from '../lib/ui/TitleBar.svelte'
  import Toast from '../lib/ui/Toast.svelte'
  import { homeService, type Groups, type HomeApi, type Notice, type Prefs, type StickypaneState } from '../services/home'
  import Empty from './Empty.svelte'
  import Sessions from './Sessions.svelte'
  import Install from './Install.svelte'

  let { api = homeService, now = () => new Date() }: { api?: HomeApi; now?: () => Date } = $props()

  const KEY = 'stickypane.home.dismissed'
  let groups = $state<Groups | null>(null)
  let failure = $state('')
  let status = $state<StickypaneState | null>(null)
  let notices = $state<Notice[]>([])
  // lines of the window's own (a failed call), apart from the shell's
  let local = $state<Notice[]>([])
  let confirming = $state('')
  let prefs = $state<Prefs>({ hotkey: '', language: '', mode: '' })
  let systemDark = $state(false)
  let busy = $state(false)
  let at = $state(new Date())
  let dismissed = $state<number[]>(readDismissed())

  function readDismissed(): number[] {
    try {
      return JSON.parse(localStorage.getItem(KEY) ?? '[]') as number[]
    } catch {
      return []
    }
  }
  function rememberDismissed(ids: number[]) {
    dismissed = ids
    try {
      localStorage.setItem(KEY, JSON.stringify(ids))
    } catch {
      // private window or blocked storage: the note shows again next time
    }
  }

  const mode = $derived((prefs.mode === 'dark' || prefs.mode === 'light' ? prefs.mode : 'system') as Mode)
  const dark = $derived(resolveDark(mode, systemDark))
  $effect(() => {
    const p = fallbackPalette(dark)
    look.palette = p
    look.dark = p.dark
    applyTheme(p, p.dark)
  })

  const lang = $derived<Choice>(prefs.language === 'ko' || prefs.language === 'en' ? prefs.language : 'auto')
  $effect(() => setLanguage(lang === 'auto' ? resolveAuto() : lang))

  const chrome = chromeOf(location.search)
  const win = windowActions('home')
  // the shell's one-time word that a closed window left the app in the tray (03 §3.11.6)
  const TRAY = 'stayInTray'
  // the codes of the window's own failures, drawn as errors
  const LOCAL: Record<string, true> = { init_failed: true, restore_failed: true, status_failed: true, notices_failed: true }
  // notices put away for as long as the window lives
  let hidden = $state<string[]>([])
  const hide = (code: string) => (hidden = [...hidden, code])
  const shown = $derived([...notices, ...local].filter((n) => !hidden.includes(n.code)))
  const removed = $derived((groups?.Removed ?? []).filter((p) => !dismissed.includes(p.ID)))
  const count = $derived((groups?.Active ?? []).length + (groups?.Days ?? []).reduce((n, d) => n + (d.Rows ?? []).length, 0))
  const setup = $derived(status && !status.found ? 'install' : status && !status.bridge ? 'update' : null)
  const noticeText = (n: Notice) => {
    const s = $tr('notice.' + n.code, { detail: n.detail })
    return s === 'notice.' + n.code ? n.detail : s
  }

  async function loadSessions() {
    at = now()
    try {
      groups = await api.sessions()
      failure = ''
    } catch (e) {
      failure = String((e as Error)?.message ?? e)
    }
  }

  async function load() {
    try {
      await api.ensure()
    } catch (e) {
      failure = String((e as Error)?.message ?? e)
    }
    await Promise.all([
      loadSessions(),
      api.status().then((s) => (status = s), (e) => warn('status_failed', e)),
      api.notices().then((n) => (notices = n ?? []), (e) => warn('notices_failed', e)),
    ])
  }

  function warn(code: string, e: unknown) {
    local = [...local.filter((n) => n.code !== code), { code, detail: String((e as Error)?.message ?? e) }]
    hidden = hidden.filter((c) => c !== code) // a new failure is shown even if the last was put away
  }

  // pick asks for a folder; a folder with a board opens, any other asks first.
  async function pick() {
    busy = true
    try {
      const dir = await api.pickFolder()
      if (!dir) return
      const found = await api.inspect(dir)
      if (found.exists) {
        await api.openProject(found.path)
        await loadSessions()
      } else {
        confirming = found.path
      }
    } catch (e) {
      warn('init_failed', e)
    } finally {
      busy = false
    }
  }

  async function create() {
    const dir = confirming
    confirming = ''
    busy = true
    try {
      await api.initProject(dir)
      await loadSessions()
    } catch (e) {
      warn('init_failed', e)
    } finally {
      busy = false
    }
  }

  async function undo() {
    const ids = removed.map((p) => p.ID)
    const results = await Promise.allSettled(ids.map((id) => api.restore(id)))
    const bad = results.find((r) => r.status === 'rejected') as PromiseRejectedResult | undefined
    if (bad) warn('restore_failed', bad.reason)
    await loadSessions()
  }

  const host = {
    kind: 'home' as const,
    save: async (c: Choice) => {
      const next = { ...prefs, language: c === 'auto' ? '' : c }
      await api.setPrefs(next)
      prefs = next
    },
  }

  onMount(() => {
    const stopDark = watchSystemDark((d) => (systemDark = d))
    const stopNotices = api.onNotices((n) => (notices = n))
    const refresh = () => void loadSessions()
    window.addEventListener('focus', refresh)
    // The language and mode are read before the first paint of the list; the list does not wait for them.
    void api.prefs().then(
      (p) => (prefs = p),
      () => {},
    )
    void load()
    return () => {
      stopDark()
      stopNotices()
      window.removeEventListener('focus', refresh)
    }
  })
</script>

<div class="window">
  <TitleBar name={$tr('home.title')} {chrome} window={win}>
    {#snippet actions()}
      {#if count > 0}
        <Button variant="primary" size="sm" icon="plus" onclick={pick} disabled={busy || status?.found !== true}>{$tr('home.addProject')}</Button>
      {/if}
    {/snippet}
  </TitleBar>

  {#if shown.length || failure}
    <div class="notices">
      {#each shown as n (n.code)}
        {#if n.code === TRAY}
          <Toast message={noticeText(n)} icon="tab" action={{ label: $tr('app.quit'), run: win.quit }} onclose={() => hide(n.code)} />
        {:else}
          <Toast message={noticeText(n)} tone={n.code in LOCAL ? 'bad' : 'warn'} icon="warn" onclose={() => hide(n.code)} />
        {/if}
      {/each}
      {#if failure}
        <Toast message={$tr('notice.index', { detail: failure })} tone="bad" icon="warn" />
      {/if}
    </div>
  {/if}

  <div class="scroll">
    <Paper>
      <main>
        {#if groups && count > 0}
          {#if setup}
            <div class="setup"><NoteFrame color={look.palette?.warn ?? 'currentColor'} id="setup"><Install kind={setup} copy={api.copy} /></NoteFrame></div>
          {/if}
          <Sessions {groups} now={at} open={(p) => void api.openProject(p)} />
        {:else if groups && !failure}
          <Empty {pick} copy={api.copy} {setup} {busy} canPick={status?.found === true} />
        {/if}
        <footer><Language {host} current={lang} /></footer>
      </main>
    </Paper>
  </div>

  {#if removed.length}
    <div class="toasts">
      <Toast
        message={$tr('home.removed', { n: removed.length })}
        icon="restore"
        action={{ label: $tr('home.undo'), run: undo }}
        onclose={() => rememberDismissed([...dismissed, ...removed.map((p) => p.ID)])}
      />
    </div>
  {/if}

  <Confirm
    open={confirming !== ''}
    title={$tr('home.confirmInit')}
    message={confirming}
    confirmLabel={$tr('home.create')}
    cancelLabel={$tr('home.cancel')}
    onconfirm={create}
    oncancel={() => (confirming = '')}
  />
</div>

<style>
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
    overflow: auto;
  }
  main {
    box-sizing: border-box;
    max-width: 760px;
    margin: 0 auto;
    padding: var(--space-6) var(--space-6) var(--space-10);
  }
  .notices {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-3) var(--space-4) 0;
  }
  .notices > :global(*) {
    width: min(100%, 440px);
  }
  .setup {
    margin-bottom: var(--space-6);
  }
  footer {
    display: flex;
    justify-content: center;
    margin-top: var(--space-8);
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
