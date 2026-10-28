<script lang="ts">
  // The home window: the boards on this machine, straight away (R-32).
  import { onMount } from 'svelte'
  import { chooseLanguage, resolveAuto, setLanguage, tr, type Choice } from '../i18n'
  import Logo from '../lib/doodle/Logo.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import Paper from '../lib/doodle/Paper.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Language from '../settings/Language.svelte'
  import { fallbackPalette } from '../lib/theme/fallback'
  import { resolveDark, watchSystemDark, type Mode } from '../lib/theme/mode'
  import { applyTheme } from '../lib/theme/tokens'
  import { realApi, type Groups, type HomeApi, type Notice, type Prefs, type StickypaneState } from './api'
  import Empty from './Empty.svelte'
  import Sessions from './Sessions.svelte'
  import Install from './Install.svelte'

  let { api = realApi, now = () => new Date() }: { api?: HomeApi; now?: () => Date } = $props()

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

  const shown = $derived([...notices, ...local])
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

<Paper>
  <main>
    <header>
      <Logo size={34} />
      <span class="brand">{$tr('home.title')}</span>
      {#if count > 0}
        <button type="button" class="add" onclick={pick} disabled={busy || status?.found !== true}>{$tr('home.addProject')}</button>
      {/if}
    </header>

    {#if shown.length || failure || confirming}
      <div class="notices">
        {#each shown as n (n.code)}
          <div role="status" class="notice">
            <NoteFrame color={look.palette?.warn ?? 'currentColor'} id={'notice-' + n.code}>{noticeText(n)}</NoteFrame>
          </div>
        {/each}
        {#if confirming}
          <div role="dialog" aria-label={$tr('home.confirmInit')} class="notice">
            <NoteFrame color={look.palette?.accent ?? 'currentColor'} id="confirm-init">
              <p class="ask">{$tr('home.confirmInit')}</p>
              <p class="where">{confirming}</p>
              <button type="button" onclick={create}>{$tr('home.create')}</button>
              <button type="button" onclick={() => (confirming = '')}>{$tr('home.cancel')}</button>
            </NoteFrame>
          </div>
        {/if}
        {#if failure}
          <div role="alert" class="notice">
            <NoteFrame color={look.palette?.bad ?? 'currentColor'} id="notice-index">{$tr('notice.index', { detail: failure })}</NoteFrame>
          </div>
        {/if}
      </div>
    {/if}

    {#if groups && count > 0}
      {#if setup}
        <div class="notice setup"><NoteFrame color={look.palette?.warn ?? 'currentColor'} id="setup"><Install kind={setup} copy={api.copy} /></NoteFrame></div>
      {/if}
      <Sessions {groups} now={at} open={(p) => void api.openProject(p)} />
    {:else if groups && !failure}
      <Empty {pick} copy={api.copy} {setup} {busy} canPick={status?.found === true} />
    {/if}

    {#if removed.length}
      <p class="removed">
        <span>{$tr('home.removed', { n: removed.length })}</span>
        <span aria-hidden="true">·</span>
        <button type="button" onclick={undo}>{$tr('home.undo')}</button>
        <button type="button" class="x" aria-label={$tr('home.dismiss')} onclick={() => rememberDismissed([...dismissed, ...removed.map((p) => p.ID)])}>×</button>
      </p>
    {/if}

    <footer><Language {host} current={lang} /></footer>
  </main>
</Paper>

<style>
  main {
    box-sizing: border-box;
    min-height: 100vh;
    padding: 18px 24px 24px;
    max-width: 760px;
    margin: 0 auto;
  }
  header {
    display: flex;
    align-items: center;
    gap: 10px;
    margin-bottom: 18px;
  }
  .brand {
    font-family: var(--font-hand);
    font-size: 28px;
    font-weight: 700;
  }
  .add {
    margin-left: auto;
    font-family: var(--font-hand);
    font-size: 18px;
    padding: 2px 14px;
    color: var(--ink);
    background: transparent;
    border: 2px solid var(--accent);
    border-radius: 10px;
    cursor: pointer;
  }
  .add:disabled {
    opacity: 0.5;
    cursor: default;
  }
  .notices {
    display: flex;
    flex-direction: column;
    gap: 14px;
    margin-bottom: 18px;
  }
  .notice {
    font-family: var(--font-hand);
    font-size: 18px;
  }
  .ask {
    margin: 0;
  }
  .where {
    margin: 2px 0 8px;
    font-family: var(--font-mono);
    font-size: 12px;
    overflow-wrap: anywhere;
  }
  .notice button {
    font: inherit;
    color: var(--ink);
    background: transparent;
    border: 1.5px solid currentColor;
    border-radius: 8px;
    padding: 0 12px;
    margin-right: 8px;
    cursor: pointer;
  }
  .setup {
    margin-bottom: 18px;
  }
  .removed {
    display: flex;
    gap: 8px;
    align-items: baseline;
    margin: 8px 0 0;
    font-family: var(--font-hand);
    font-size: 17px;
    color: var(--pencil);
  }
  .removed button {
    font: inherit;
    color: var(--ink);
    background: none;
    border: 0;
    padding: 0;
    text-decoration: underline;
    cursor: pointer;
  }
  .removed .x {
    text-decoration: none;
    margin-left: 4px;
  }
  footer {
    margin-top: 28px;
  }
</style>
