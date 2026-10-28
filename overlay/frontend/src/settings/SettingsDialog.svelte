<script lang="ts">
  // The settings (⌘, or ⋯ → 설정…, 10 §10.5). What belongs to the board —
  // theme, style, language — is the board's sticky.json, through the
  // bridge, and the terminal sees it too. What belongs to this computer's
  // app — light or dark, the hotkey — is overlay.json, through the shell.
  // Every change shows at once.
  import { tr, type Choice, type LanguageHost } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import type { Mode } from '../lib/theme/mode'
  import Button from '../lib/ui/Button.svelte'
  import Dialog from '../lib/ui/Dialog.svelte'
  import Input from '../lib/ui/Input.svelte'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import type { Settings } from '../board/model'
  import { errorText } from '../services/errors'
  import { setupService, type SetupApi } from '../services/install'
  import { setAppPref, type BoardSettingsPatch, type SettingsApi } from '../services/settings'
  import Language from './Language.svelte'
  import Terminal from './Terminal.svelte'

  let {
    open = $bindable(false),
    api,
    setup = setupService,
    settings,
    mode,
    hotkey = '',
    version = '',
    onmode,
    onchanged,
  }: {
    open?: boolean
    api: SettingsApi
    /** the terminal command and the agents (any window's) */
    setup?: SetupApi
    /** the board's, as board.load gave them */
    settings: Settings
    mode: Mode
    hotkey?: string
    /** the bridge's stickypane */
    version?: string
    onmode: (m: Mode) => void
    /** a board setting was saved: read the board again */
    onchanged: () => void
  } = $props()

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
  })
</script>

<Dialog bind:open title={$tr('settings.title')} width={520}>
  <div class="settings">
    <section aria-labelledby="s-board">
      <h3 id="s-board" class:neat={look.style === 'neat'}>{$tr('settings.board')}</h3>
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
        <Language {host} current={(settings.language?.choice as Choice) ?? 'auto'} />
      </div>
    </section>

    <section aria-labelledby="s-app">
      <h3 id="s-app" class:neat={look.style === 'neat'}>{$tr('settings.app')}</h3>
      <p class="scope">{$tr('settings.appScope')}</p>
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
      <form class="keys" onsubmit={(e) => (e.preventDefault(), saveKeys())}>
        <Input label={$tr('settings.hotkey')} hint={keysSaved ? $tr('settings.hotkey.saved') : $tr('settings.hotkey.hint')} error={keysError} bind:value={keys} />
        <Button size="sm" variant="secondary" type="submit" disabled={!keys.trim() || keys.trim() === hotkey}>{$tr('settings.save')}</Button>
      </form>
    </section>

    <section aria-labelledby="s-cli">
      <h3 id="s-cli" class:neat={look.style === 'neat'}>{$tr('settings.terminal')}</h3>
      <Terminal {setup} {open} project />
    </section>

    <section aria-labelledby="s-about">
      <h3 id="s-about" class:neat={look.style === 'neat'}>{$tr('settings.about')}</h3>
      <p class="scope">{version ? $tr('settings.version', { version }) : 'stickypane'}</p>
    </section>

    {#if failed}<p class="err" role="alert">{failed}</p>{/if}
  </div>
  {#snippet actions()}
    <Button variant="primary" onclick={() => (open = false)}>{$tr('ui.close')}</Button>
  {/snippet}
</Dialog>

<style>
  .settings {
    display: flex;
    flex-direction: column;
    gap: var(--space-5);
    max-height: min(64vh, 560px);
    overflow: auto;
    padding-right: var(--space-1);
  }
  section {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  h3 {
    margin: 0;
    color: var(--ink);
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
  }
  h3.neat {
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .scope {
    margin: calc(-1 * var(--space-2)) 0 0;
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
