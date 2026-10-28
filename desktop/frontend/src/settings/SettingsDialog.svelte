<script lang="ts">
  // The settings (⌘, or ⋯ → 설정…, 10 §10.5). What belongs to the board —
  // theme, style, language — is the board's sticky.json, through the
  // bridge, and the terminal sees it too. What belongs to this computer's
  // app — its language, light or dark, the hotkey — is overlay.json,
  // through the shell. They are four tabs (this board, this computer,
  // terminal and agents, about), each its own panel.
  // Every change shows at once.
  import { tr, type Choice, type LanguageHost } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { motion } from '../lib/motion/level.svelte'
  import type { Mode } from '../lib/theme/mode'
  import Button from '../lib/ui/Button.svelte'
  import Dialog from '../lib/ui/Dialog.svelte'
  import Input from '../lib/ui/Input.svelte'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import type { Settings } from '../board/model'
  import { errorText } from '../services/errors'
  import { setupService, type SetupApi } from '../services/install'
  import { setAppPref, type BoardSettingsPatch, type SettingsApi } from '../services/settings'
  import Tabs from '../lib/ui/Tabs.svelte'
  import Language from './Language.svelte'
  import Terminal from './Terminal.svelte'

  let {
    open = $bindable(false),
    api,
    setup = setupService,
    settings,
    mode,
    hotkey = '',
    appLanguage = '',
    version = '',
    onmode,
    onchanged,
    onapplanguage,
  }: {
    open?: boolean
    api: SettingsApi
    /** the terminal command and the agents (any window's) */
    setup?: SetupApi
    /** the board's, as board.load gave them */
    settings: Settings
    mode: Mode
    hotkey?: string
    /** the app's language, overlay.json's: "", "ko" or "en" */
    appLanguage?: string
    /** the bridge's stickypane */
    version?: string
    onmode: (m: Mode) => void
    /** a board setting was saved: read the board again */
    onchanged: () => void
    /** the app's language was saved */
    onapplanguage?: (l: string) => void
  } = $props()

  // the settings in four tabs, so none is far down a long page; arrows
  // move between the tabs (Tabs), Tab goes into the one shown
  type Section = 'board' | 'app' | 'cli' | 'about'
  let section = $state<Section>('board')
  const id = $props.id()
  const sections = $derived([
    { id: 'board', label: $tr('settings.board') },
    { id: 'app', label: $tr('settings.app') },
    { id: 'cli', label: $tr('settings.terminal') },
    { id: 'about', label: $tr('settings.about') },
  ])
  const panel = (s: Section) => ({
    role: 'tabpanel',
    id: `${id}-panel-${s}`,
    'aria-labelledby': `${id}-tab-${s}`,
    tabindex: 0,
    class: 'panel',
  })

  let failed = $state('')
  let keys = $state('')
  let keysError = $state('')
  let keysSaved = $state(false)
  $effect(() => {
    if (open) {
      keys = hotkey
      keysError = ''
      keysSaved = false
      failed = ''
    }
  })

  // the bridge's themes.list; until it answers (or from an older bridge),
  // the one chosen is the only one offered besides auto
  let names = $state<string[]>([])
  $effect(() => {
    if (!open) return
    api.themes().then(
      (list) => (names = list.map((t) => t.name)),
      () => (names = []),
    )
  })
  const themes = $derived.by(() => {
    const out = ['auto', ...names]
    const now = settings.theme || 'auto'
    return out.includes(now) ? out : [...out, now]
  })

  async function board(p: BoardSettingsPatch) {
    failed = ''
    try {
      await api.setSettings(p)
      onchanged()
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
    }
  }

  async function setMode(m: Mode) {
    failed = ''
    onmode(m)
    try {
      await setAppPref(api, { mode: m === 'system' ? '' : m })
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
    }
  }

  // how much the windows move (04 §4.6): shown at once, kept in overlay.json
  async function setMotion(m: string) {
    failed = ''
    motion.setting = m === 'system' ? '' : m
    try {
      await setAppPref(api, { motion: motion.setting })
    } catch (e) {
      failed = $tr('settings.failed', { error: errorText(e) })
    }
  }

  async function saveKeys() {
    keysError = ''
    keysSaved = false
    try {
      await setAppPref(api, { hotkey: keys.trim() })
      keysSaved = true
    } catch (e) {
      keysError = $tr('settings.hotkey.failed', { error: errorText(e) })
    }
  }

  const host = $derived<LanguageHost>({
    kind: 'project',
    settingsSet: (_, v) => api.setSettings({ language: v }),
    settingsGet: () => api.settings() as ReturnType<Extract<LanguageHost, { kind: 'project' }>['settingsGet']>,
    app: appLanguage,
  })

  // the app's language is this computer's (overlay.json), as the home
  // window's is; this window keeps drawing in the board's when it has one
  const appChoice = $derived<Choice>(appLanguage === 'ko' || appLanguage === 'en' ? appLanguage : 'auto')
  async function setAppLanguage(c: Choice) {
    const value = c === 'auto' ? '' : c
    await setAppPref(api, { language: value })
    onapplanguage?.(value)
  }
</script>

<Dialog bind:open title={$tr('settings.title')} width={560}>
  <div class="settings" class:doodle={look.style === 'doodle'}>
    <Tabs label={$tr('settings.sections')} tabs={sections} bind:value={section} controls={id} />
    {#if section === 'board'}
    <section {...panel('board')}>
      <p class="scope">{$tr('settings.boardScope')}</p>
      <label class="row">
        <span class="label">{$tr('settings.theme')}</span>
        <select class="select" value={settings.theme || 'auto'} onchange={(e) => board({ theme: e.currentTarget.value })}>
          {#each themes as name (name)}
            <option value={name}>{name === 'auto' ? $tr('settings.theme.auto') : name}</option>
          {/each}
        </select>
      </label>
      <div class="row">
        <span class="label">{$tr('settings.style')}</span>
        <SegmentedControl
          label={$tr('settings.style')}
          size="sm"
          value={settings.style === 'neat' ? 'neat' : 'doodle'}
          options={[
            { value: 'doodle', label: $tr('settings.style.doodle') },
            { value: 'neat', label: $tr('settings.style.neat') },
          ]}
          onchange={(v) => {
            look.style = v === 'neat' ? 'neat' : 'doodle'
            board({ style: v as 'doodle' | 'neat' })
          }}
        />
      </div>
      <div class="row">
        <Language
          {host}
          current={(settings.language?.choice as Choice) ?? 'auto'}
          label={$tr('settings.boardLanguage')}
          autoLabel={$tr('settings.boardLanguage.auto')}
          hint={$tr('settings.boardLanguage.hint')}
        />
      </div>
    </section>
    {:else if section === 'app'}
    <section {...panel('app')}>
      <p class="scope">{$tr('settings.appScope')}</p>
      <div class="row">
        <Language
          current={appChoice}
          label={$tr('settings.appLanguage')}
          hint={$tr('settings.appLanguage.hint')}
          pick={setAppLanguage}
        />
      </div>
      <div class="row">
        <span class="label">{$tr('settings.mode')}</span>
        <SegmentedControl
          label={$tr('settings.mode')}
          size="sm"
          value={mode}
          options={[
            { value: 'system', label: $tr('settings.mode.system'), icon: 'auto' },
            { value: 'dark', label: $tr('settings.mode.dark'), icon: 'moon' },
            { value: 'light', label: $tr('settings.mode.light'), icon: 'sun' },
          ]}
          onchange={(v) => setMode(v as Mode)}
        />
      </div>
      <div class="row">
        <span class="label">{$tr('settings.motion')}</span>
        <SegmentedControl
          label={$tr('settings.motion')}
          size="sm"
          value={motion.setting === 'reduce' || motion.setting === 'off' ? motion.setting : 'system'}
          options={[
            { value: 'system', label: $tr('settings.motion.system') },
            { value: 'reduce', label: $tr('settings.motion.reduce') },
            { value: 'off', label: $tr('settings.motion.off') },
          ]}
          onchange={setMotion}
        />
      </div>
      <form class="keys" onsubmit={(e) => (e.preventDefault(), saveKeys())}>
        <Input label={$tr('settings.hotkey')} hint={keysSaved ? $tr('settings.hotkey.saved') : $tr('settings.hotkey.hint')} error={keysError} bind:value={keys} />
        <Button size="sm" variant="secondary" type="submit" disabled={!keys.trim() || keys.trim() === hotkey}>{$tr('settings.save')}</Button>
      </form>
    </section>
    {:else if section === 'cli'}
    <section {...panel('cli')}>
      <Terminal {setup} {open} project />
    </section>
    {:else}
    <section {...panel('about')}>
      <p class="version">{version ? $tr('settings.version', { version }) : 'stickypane'}</p>
    </section>
    {/if}

    {#if failed}<p class="err" role="alert">{failed}</p>{/if}
  </div>
  {#snippet actions()}
    <Button variant="primary" onclick={() => (open = false)}>{$tr('ui.close')}</Button>
  {/snippet}
</Dialog>

<style>
  .settings {
    --ground: var(--surface);
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
  }
  .settings.doodle {
    --ground: var(--paper);
  }
  .settings > :global(.tabs) {
    flex-wrap: wrap;
    border-bottom: var(--stroke-hairline) solid var(--line);
    padding-bottom: var(--space-1);
  }
  /* a tab that is still too long for the window scrolls, and its edge
     fades so there is plainly more below */
  section {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
    min-height: 220px;
    max-height: min(56vh, 460px);
    overflow: auto;
    padding: var(--space-1) var(--space-1) var(--space-2) 0;
    background:
      linear-gradient(transparent, var(--ground) 60%) center bottom / 100% 40px no-repeat local,
      linear-gradient(transparent, color-mix(in srgb, var(--ink) 14%, transparent)) center bottom / 100% 12px no-repeat scroll;
  }
  section:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
    border-radius: var(--radius-sm);
  }
  .version {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-body-sm) var(--font-mono);
  }
  .scope {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .row {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }
  .label {
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .select {
    min-width: 12rem;
    height: var(--control-md);
    padding: 0 var(--space-2);
    border: 1px solid var(--line-strong);
    border-radius: var(--radius-md);
    background: var(--surface-sunk);
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .select:focus-visible {
    outline: none;
    border-color: var(--focus);
    box-shadow: 0 0 0 3px var(--accent-wash);
  }
  .keys {
    display: flex;
    align-items: flex-start;
    gap: var(--space-2);
  }
  .keys > :global(.field) {
    flex: 1;
  }
  .keys > :global(.btn) {
    margin-top: 22px;
  }
  .err {
    margin: 0;
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
