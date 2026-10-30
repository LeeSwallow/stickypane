<script lang="ts" module>
  import type { Board as BoardData } from '../board/model'
  import type { Status as BoardStatus } from '../services/board'
  import type { Layout as CanvasLayout, LayoutMode as Mode2 } from '../services/canvas'
  import type { Palette as P } from '../lib/theme/palette'
  import type { BoardSettingsPatch as Patch } from '../services/settings'
  /** what the window asks of the board it was opened from */
  export interface BoardSide {
    status(): Promise<BoardStatus>
    load(): Promise<BoardData>
    themes(): Promise<P[]>
    setSettings(p: Patch): Promise<unknown>
    layout(): Promise<CanvasLayout>
    setMode(m: Mode2): Promise<void>
  }
</script>

<script lang="ts">
  // The settings window (03 §3.12): its own window, a sidebar of sections
  // on the left, the section on paper on the right. 보드 is the board the
  // window was opened from (its sticky.json, through the bridge, shared
  // with the terminal; its layout on this computer); 이 컴퓨터 is
  // overlay.json; 터미널과 에이전트 and 정보 are the app's. Every change is
  // kept at once and the board's window redraws with it.
  import { onMount } from 'svelte'
  import { resolveAuto, setLanguage, tr, type Choice } from '../i18n'
  import Logo from '../lib/doodle/Logo.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import Paper from '../lib/doodle/Paper.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import { motion } from '../lib/motion/level.svelte'
  import { STICKYPANE_DARK, STICKYPANE_LIGHT, fallbackPalette } from '../lib/theme/fallback'
  import { resolveDark, watchSystemDark, type Mode } from '../lib/theme/mode'
  import { normalizePalette, type Palette } from '../lib/theme/palette'
  import { applyTheme } from '../lib/theme/tokens'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import type { Chrome } from '../lib/ui/chrome'
  import CrayonIcon from '../lib/ui/icons/CrayonIcon.svelte'
  import type { CrayonName } from '../lib/ui/icons/crayon'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import TitleBar from '../lib/ui/TitleBar.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import type { Board } from '../board/model'
  import { modeOf, paletteFor } from '../board/theme'
  import type { Status } from '../services/board'
  import type { LayoutMode, Layout } from '../services/canvas'
  import { errorText } from '../services/errors'
  import type { SetupApi } from '../services/install'
  import {
    SECTIONS, setAppPref,
    type About, type AppApi, type AppPrefs, type Autostart, type BoardSettingsPatch, type Section,
  } from '../services/settings'
  import { windowActions } from '../services/window'
  import CardGroup from './CardGroup.svelte'
  import ChoiceCard from './ChoiceCard.svelte'
  import HotkeyRecorder from './HotkeyRecorder.svelte'
  import Row from './Row.svelte'
  import Swatch from './Swatch.svelte'
  import Terminal from './Terminal.svelte'
  import { chosenFamily, families, nameFor } from './themes'

  let {
    board: side,
    app,
    setup,
    section: start = 'board',
    chrome = 'none',
  }: { board: BoardSide; app: AppApi; setup: SetupApi; section?: Section; chrome?: Chrome } = $props()

  const GITHUB = 'https://github.com/LeeSwallow/stickypane'
  const DOCS = 'https://github.com/LeeSwallow/stickypane/tree/main/docs'

  let section = $state<Section>('board')
  $effect.pre(() => {
    section = start
  })
  let status = $state<Status | null>(null)
  let board = $state<Board | null>(null)
  let themes = $state<Palette[]>([])
  let layoutMode = $state<LayoutMode>('canvas')
  let prefs = $state<AppPrefs>({ hotkey: '', language: '', mode: '' })
  let systemDark = $state(false)
  let autostart = $state<Autostart | null>(null)
  let about = $state<About | null>(null)
  let failed = $state('')
  let loaded = $state(false)

  const win = windowActions('home')
  const mode = $derived<Mode>(modeOf(prefs.mode))
  const dark = $derived(resolveDark(mode, systemDark))
  const ready = $derived(status?.state === 'ready' && !!board)

  // the window draws as its board does (its theme for the screen's light
  // or dark, its style), and speaks the app's language
  const bridgePalette = $derived.by((): Palette | null => {
    try {
      return board ? normalizePalette(board.palette) : null
    } catch {
      return null
    }
  })
  const palette = $derived(bridgePalette && board ? paletteFor(bridgePalette, board.settings.theme, mode, systemDark, themes) : fallbackPalette(dark))
  $effect(() => {
    applyTheme(palette, palette.dark)
    look.palette = palette
    look.dark = palette.dark
    look.style = board?.settings.style === 'neat' ? 'neat' : 'doodle'
  })
  $effect(() => setLanguage(prefs.language === 'ko' || prefs.language === 'en' ? prefs.language : resolveAuto()))

  async function readBoard() {
    try {
      status = await side.status()
      if (status.state !== 'ready') {
        board = null
        return
      }
      board = await side.load()
      layoutMode = (await side.layout()).mode
      if (!themes.length) themes = await side.themes().catch(() => [])
    } catch (e) {
      board = null
      failed = errorText(e)
    } finally {
      loaded = true
    }
  }

  onMount(() => {
    const offs = [
      watchSystemDark((d) => (systemDark = d)),
      app.onPrefs((p) => {
        prefs = p
        motion.setting = p.motion ?? ''
      }),
      app.onBoard((s) => {
        if (s) section = s
        void readBoard()
      }),
    ]
    app.prefs().then((p) => {
      prefs = p
      motion.setting = p.motion ?? ''
    }, () => {})
    app.autostart().then((a) => (autostart = a), () => (autostart = null))
    app.about().then((a) => (about = a), () => {})
    void readBoard()
    return () => offs.forEach((off) => off())
  })

  // --- 보드 ---------------------------------------------------------------

  const fams = $derived(families(themes))
  const theme = $derived(board?.settings.theme || 'auto')
  const chosen = $derived(chosenFamily(fams, theme))
  const boardChoice = $derived<Choice>((board?.settings.language?.choice as Choice) ?? 'auto')

  async function setBoard(p: BoardSettingsPatch) {
    failed = ''
    if (board) board = { ...board, settings: { ...board.settings, ...(p as object), language: p.language ? { ...board.settings.language, choice: p.language } : board.settings.language } }
    try {
      await side.setSettings(p)
      board = await side.load()
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
      void readBoard()
    }
  }

  async function setLayout(m: LayoutMode) {
    layoutMode = m
    try {
      await side.setMode(m)
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
    }
  }

  // --- 이 컴퓨터 -------------------------------------------------------------

  const prefApi = { appPrefs: () => app.prefs(), setAppPrefs: (p: AppPrefs) => app.setPrefs(p) }

  async function setPref(patch: Partial<AppPrefs>) {
    failed = ''
    prefs = { ...prefs, ...patch }
    if (patch.motion !== undefined) motion.setting = patch.motion
    try {
      prefs = await setAppPref(prefApi, patch)
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
    }
  }

  async function setLogin(on: boolean) {
    try {
      autostart = await app.setAutostart(on)
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
      autostart = await app.autostart().catch(() => autostart)
    }
  }

  // --- the sidebar ---------------------------------------------------------

  const NAV: Record<Section, { icon: CrayonName; color: number }> = {
    board: { icon: 'board', color: 0 },
    app: { icon: 'computer', color: 2 },
    cli: { icon: 'terminal', color: 3 },
    about: { icon: 'info', color: 4 },
  }
  const id = $props.id()
  let nav: HTMLElement | undefined = $state()

  function navKey(e: KeyboardEvent) {
    const i = SECTIONS.indexOf(section)
    const j = { ArrowDown: i + 1, ArrowUp: i - 1, Home: 0, End: SECTIONS.length - 1 }[e.key]
    if (j === undefined) return
    e.preventDefault()
    section = SECTIONS[(j + SECTIONS.length) % SECTIONS.length]
    nav?.querySelectorAll<HTMLElement>('[role="tab"]')[SECTIONS.indexOf(section)]?.focus()
  }

  const sheetColor = $derived(palette.notes[NAV[section].color] ?? 'currentColor')
  const neat = $derived(look.style === 'neat')
</script>

<div class="window">
  <TitleBar name={$tr('settings.title')} {chrome} window={win} />
  <div class="body">
    <nav class="side" aria-label={$tr('settings.sections')}>
      <div role="tablist" aria-orientation="vertical" tabindex="-1" bind:this={nav} onkeydown={navKey}>
        {#each SECTIONS as s (s)}
          <button
            type="button"
            role="tab"
            id="{id}-tab-{s}"
            aria-selected={section === s}
            aria-controls="{id}-panel"
            tabindex={section === s ? 0 : -1}
            class="tab"
            class:on={section === s}
            onclick={() => (section = s)}
          >
            <CrayonIcon name={NAV[s].icon} size={22} />
            <span>{$tr('settings.nav.' + s)}</span>
          </button>
        {/each}
      </div>
    </nav>

    <div class="scroll">
      <Paper>
        <div class="page" role="tabpanel" id="{id}-panel" aria-labelledby="{id}-tab-{section}">
          {#key section}
            <NoteFrame color={sheetColor} id="settings-{section}" tape="18%">
              <header class="head">
                <h1 class:neat>
                  {$tr('settings.nav.' + section)}{#if section === 'board' && board}<span class="of"> · {board.title}</span>{/if}
                </h1>
                {#if section === 'board' && ready}<p class="where">{$tr('settings.board.where')}</p>{/if}
                {#if section === 'app'}<p class="where">{$tr('settings.app.where')}</p>{/if}
                {#if section === 'cli'}<p class="where">{$tr('settings.cli.where')}</p>{/if}
              </header>

              {#if section === 'board'}
                {#if !loaded}
                  <p class="quiet">{$tr('settings.board.loading')}</p>
                {:else if !ready || !board}
                  <div class="none">
                    <CrayonIcon name="board" size={40} />
                    <p class="none-title" class:neat>{$tr('settings.board.none')}</p>
                    <p class="quiet">{$tr('settings.board.noneHint')}</p>
                  </div>
                {:else}
                  <Row label={$tr('settings.theme')} hint={$tr('settings.theme.hint')} stack>
                    <CardGroup label={$tr('settings.theme')} min={150}>
                      <ChoiceCard label={$tr('settings.theme.auto')} sub="auto" checked={theme === 'auto'} onpick={() => setBoard({ theme: 'auto' })}>
                        {#snippet picture()}
                          <span class="pair"><Swatch palette={STICKYPANE_DARK} style={look.style} width={80} height={64} seed="auto-d" /><Swatch palette={STICKYPANE_LIGHT} style={look.style} width={80} height={64} seed="auto-l" /></span>
                        {/snippet}
                      </ChoiceCard>
                      {#each fams as f (f.id)}
                        <ChoiceCard
                          label={f.label}
                          sub={f.halves.length ? f.halves.join(' · ') : $tr('settings.theme.darkOnly')}
                          checked={chosen?.id === f.id}
                          onpick={() => setBoard({ theme: nameFor(f, dark) })}
                        >
                          {#snippet picture()}
                            <span class="pair">
                              {#if f.dark}<Swatch palette={f.dark} style={look.style} width={f.light ? 80 : 160} height={64} seed={f.id + '-d'} />{/if}
                              {#if f.light}<Swatch palette={f.light} style={look.style} width={f.dark ? 80 : 160} height={64} seed={f.id + '-l'} />{/if}
                            </span>
                          {/snippet}
                        </ChoiceCard>
                      {/each}
                    </CardGroup>
                  </Row>
                  <Row label={$tr('settings.style')} hint={$tr('settings.style.hint')} stack>
                    <CardGroup label={$tr('settings.style')} min={180}>
                      {#each ['doodle', 'neat'] as const as st (st)}
                        <ChoiceCard
                          label={$tr('settings.style.' + st)}
                          checked={(board.settings.style === 'neat' ? 'neat' : 'doodle') === st}
                          onpick={() => {
                            look.style = st
                            void setBoard({ style: st })
                          }}
                        >
                          {#snippet picture()}
                            <span class="style-pic" class:hand={st === 'doodle'}>
                              <Swatch {palette} style={st} width={200} height={84} seed="style-{st}" />
                              <span class="style-word">{st === 'doodle' ? 'Aa 가나' : 'Aa 가나'}</span>
                            </span>
                          {/snippet}
                        </ChoiceCard>
                      {/each}
                    </CardGroup>
                  </Row>
                  <Row label={$tr('settings.boardLanguage')} hint={$tr('settings.boardLanguage.hint')}>
                    <SegmentedControl
                      label={$tr('settings.boardLanguage')}
                      size="sm"
                      value={boardChoice}
                      options={[
                        { value: 'ko', label: $tr('settings.language.ko') },
                        { value: 'en', label: $tr('settings.language.en') },
                        { value: 'auto', label: $tr('settings.boardLanguage.auto') },
                      ]}
                      onchange={(v) => setBoard({ language: v as Choice })}
                    />
                  </Row>
                  <Row label={$tr('settings.view')} hint={$tr('settings.view.hint')}>
                    <SegmentedControl
                      label={$tr('settings.view')}
                      size="sm"
                      value={layoutMode}
                      options={[
                        { value: 'canvas', label: $tr('ui.view.canvas'), icon: 'canvas' },
                        { value: 'grid', label: $tr('ui.view.grid'), icon: 'grid' },
                      ]}
                      onchange={(v) => setLayout(v as LayoutMode)}
                    />
                  </Row>
                  <Row label={$tr('settings.opens')} hint={$tr('settings.opens.hint')}>
                    <SegmentedControl
                      label={$tr('settings.opens')}
                      size="sm"
                      value={board.settings.view || 'terminal'}
                      options={(['terminal', 'overlay', 'both'] as const).map((v) => ({ value: v, label: $tr('settings.opens.' + v) }))}
                      onchange={(v) => setBoard({ view: v as 'terminal' | 'overlay' | 'both' })}
                    />
                  </Row>
                {/if}
              {:else if section === 'app'}
                <Row label={$tr('settings.appearance')} hint={$tr('settings.appearance.hint')} stack>
                  <CardGroup label={$tr('settings.appearance')} min={150}>
                    {#each ['system', 'light', 'dark'] as const as m (m)}
                      <ChoiceCard label={$tr('settings.mode.' + m)} checked={mode === m} onpick={() => setPref({ mode: m === 'system' ? '' : m })}>
                        {#snippet picture()}
                          <span class="pair">
                            {#if m !== 'dark'}<Swatch palette={STICKYPANE_LIGHT} style={look.style} width={m === 'system' ? 80 : 160} height={64} seed="mode-{m}-l" />{/if}
                            {#if m !== 'light'}<Swatch palette={STICKYPANE_DARK} style={look.style} width={m === 'system' ? 80 : 160} height={64} seed="mode-{m}-d" />{/if}
                          </span>
                        {/snippet}
                      </ChoiceCard>
                    {/each}
                  </CardGroup>
                </Row>
                <Row label={$tr('settings.appLanguage')} hint={$tr('settings.appLanguage.hint')}>
                  <SegmentedControl
                    label={$tr('settings.appLanguage')}
                    size="sm"
                    value={prefs.language === 'ko' || prefs.language === 'en' ? prefs.language : 'auto'}
                    options={[
                      { value: 'ko', label: $tr('settings.language.ko') },
                      { value: 'en', label: $tr('settings.language.en') },
                      { value: 'auto', label: $tr('settings.language.auto') },
                    ]}
                    onchange={(v) => setPref({ language: v === 'auto' ? '' : v })}
                  />
                </Row>
                <Row label={$tr('settings.motion')} hint={$tr('settings.motion.hint')}>
                  <SegmentedControl
                    label={$tr('settings.motion')}
                    size="sm"
                    value={prefs.motion === 'reduce' || prefs.motion === 'off' ? prefs.motion : 'system'}
                    options={(['system', 'reduce', 'off'] as const).map((v) => ({ value: v, label: $tr('settings.motion.' + v) }))}
                    onchange={(v) => setPref({ motion: v === 'system' ? '' : v })}
                  />
                </Row>
                <Row label={$tr('settings.hotkey')} hint={$tr('settings.hotkey.what')}>
                  <HotkeyRecorder
                    label={$tr('settings.hotkey')}
                    value={prefs.hotkey}
                    check={(a) => app.checkHotkey(a)}
                    save={async (a) => void (prefs = await setAppPref(prefApi, { hotkey: a }))}
                  />
                </Row>
                <Row label={$tr('settings.pinNew')} hint={$tr('settings.pinNew.hint')}>
                  <Toggle label={$tr('settings.pinNew')} checked={!!prefs.pinNew} onchange={(on) => setPref({ pinNew: on })} />
                </Row>
                {#if autostart?.supported}
                  <Row label={$tr('settings.login')} hint={$tr('settings.login.hint')}>
                    <Toggle label={$tr('settings.login')} checked={autostart.enabled} onchange={setLogin} />
                  </Row>
                {/if}
              {:else if section === 'cli'}
                <Terminal {setup} project={ready} />
              {:else}
                <div class="about">
                  <Logo size={64} />
                  <div>
                    <p class="app-name" class:neat>stickypane</p>
                    <p class="quiet">{$tr('about.tagline')}</p>
                  </div>
                </div>
                <Row label={$tr('about.app')}>
                  <code>{about?.app || '—'}</code>
                </Row>
                <Row label={$tr('about.cli')} hint={about?.stickypane.path ?? ''}>
                  {#if about && !about.stickypane.found}
                    <Badge tone="warn">{$tr('about.notFound')}</Badge>
                  {:else}
                    <code>{about?.stickypane.version || status?.version || '—'}</code>
                  {/if}
                </Row>
                <Row label={$tr('about.licenses')} hint={$tr('about.licensesBody')} />
                <div class="links">
                  <Button size="sm" variant="secondary" icon="link" onclick={() => app.openURL(GITHUB)}>{$tr('about.github')}</Button>
                  <Button size="sm" variant="secondary" icon="link" onclick={() => app.openURL(DOCS)}>{$tr('about.docs')}</Button>
                </div>
              {/if}
              {#if failed}<p class="err" role="alert">{failed}</p>{/if}
            </NoteFrame>
          {/key}
        </div>
      </Paper>
    </div>
  </div>
</div>

<style>
  :global(html),
  :global(body) {
    overflow: hidden;
  }
  .window {
    display: flex;
    flex-direction: column;
    height: 100vh;
    background: var(--paper);
    color: var(--ink);
  }
  .body {
    flex: 1;
    min-height: 0;
    display: flex;
  }
  .side {
    flex: none;
    width: 200px;
    padding: var(--space-4) var(--space-3);
    box-sizing: border-box;
    border-right: 1px solid var(--line);
    background: var(--chrome);
  }
  [role='tablist'] {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .tab {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    width: 100%;
    height: 40px;
    padding: 0 var(--space-3);
    border: 0;
    border-radius: var(--radius-md);
    background: none;
    color: var(--ink-2);
    text-align: left;
    font: 500 var(--text-body-md) / var(--leading-body-md) var(--font-body);
    cursor: pointer;
  }
  .tab:hover {
    background: var(--hover);
    color: var(--ink);
  }
  .tab.on {
    background: var(--accent-wash);
    color: var(--ink);
    font-weight: 600;
  }
  .tab:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: calc(-1 * var(--focus-width));
  }
  .scroll {
    flex: 1;
    min-width: 0;
    overflow: auto;
  }
  .page {
    box-sizing: border-box;
    max-width: 720px;
    padding: var(--space-8) var(--space-6) var(--space-8);
  }
  .head {
    margin-bottom: var(--space-2);
  }
  h1 {
    margin: 0;
    font: var(--weight-hand-display) var(--text-hand-display) / var(--leading-hand-display) var(--font-hand);
  }
  h1.neat {
    font: 600 22px / 28px var(--font-body);
  }
  .of {
    color: var(--ink-2);
  }
  .where {
    margin: var(--space-1) 0 0;
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .quiet {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .none {
    display: grid;
    justify-items: center;
    gap: var(--space-2);
    padding: var(--space-8) var(--space-4);
    text-align: center;
  }
  .none-title {
    margin: 0;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  .none-title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .pair {
    display: flex;
  }
  .pair > :global(*) {
    flex: 1;
    min-width: 0;
  }
  .style-pic {
    position: relative;
    display: block;
  }
  .style-word {
    position: absolute;
    right: var(--space-2);
    bottom: var(--space-1);
    color: var(--ink);
    font: 600 18px / 1 var(--font-body);
  }
  .style-pic.hand .style-word {
    font: 700 22px / 1 var(--font-hand);
  }
  .about {
    display: flex;
    align-items: center;
    gap: var(--space-4);
    padding: var(--space-2) 0 var(--space-4);
  }
  .app-name {
    margin: 0;
    font: var(--weight-hand-display) var(--text-hand-display) / var(--leading-hand-display) var(--font-hand);
  }
  .app-name.neat {
    font: 600 22px / 28px var(--font-body);
  }
  code {
    padding: 2px var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
  }
  .links {
    display: flex;
    gap: var(--space-2);
    padding-top: var(--space-2);
  }
  .err {
    margin: var(--space-3) 0 0;
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
