<script lang="ts">
  import { language, tr, chooseLanguage, type Choice, type LanguageHost } from '../i18n'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'

  // host says where the choice is kept: the bridge in a project window,
  // the shell's overlay.json in the home window. current is the saved choice.
  // A board's language and the app's are two settings (08 §8.2.1): label
  // names which one this is, autoLabel what "auto" means for it, and hint
  // says it in a line. pick, when given, keeps the choice in place of host
  // (the app's language from a board's window, which draws in the board's).
  let {
    host,
    current = 'auto',
    label = '',
    autoLabel = '',
    hint = '',
    pick: keep,
  }: {
    host?: LanguageHost
    current?: Choice
    label?: string
    autoLabel?: string
    hint?: string
    pick?: (c: Choice) => Promise<void>
  } = $props()
  let chosen = $state<Choice>('auto')
  $effect(() => {
    chosen = current
  })

  const options: Choice[] = ['ko', 'en', 'auto']
  const id = $props.id()

  let failed = $state(false)

  async function pick(c: Choice) {
    const before = chosen
    chosen = c
    failed = false
    try {
      if (keep) await keep(c)
      else if (host) await chooseLanguage(c, host)
    } catch {
      chosen = before
      failed = true
    }
  }
</script>

<div class="language" data-lang={$language}>
  <div class="line">
    <span class="label">{label || $tr('settings.language')}</span>
    <SegmentedControl
      label={label || $tr('settings.language')}
      size="sm"
      value={chosen}
      options={options.map((c) => ({ value: c, label: c === 'auto' && autoLabel ? autoLabel : $tr('settings.language.' + c) }))}
      onchange={(v) => pick(v as Choice)}
    />
  </div>
  {#if hint}<p class="hint" id="{id}-hint">{hint}</p>{/if}
  {#if failed}<span class="err" role="alert">{$tr('settings.language.failed')}</span>{/if}
</div>

<style>
  .language {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    width: 100%;
  }
  .line {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-3);
  }
  .label {
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .hint {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .err {
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
