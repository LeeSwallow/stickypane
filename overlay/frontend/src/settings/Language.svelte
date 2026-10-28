<script lang="ts">
  import { language, tr, chooseLanguage, type Choice, type LanguageHost } from '../i18n'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'

  // host says where the choice is kept: the bridge in a project window,
  // the shell's overlay.json in the home window. current is the saved choice.
  let { host, current = 'auto' }: { host: LanguageHost; current?: Choice } = $props()
  let chosen = $state<Choice>('auto')
  $effect(() => {
    chosen = current
  })

  const options: Choice[] = ['ko', 'en', 'auto']

  let failed = $state(false)

  async function pick(c: Choice) {
    const before = chosen
    chosen = c
    failed = false
    try {
      await chooseLanguage(c, host)
    } catch {
      chosen = before
      failed = true
    }
  }
</script>

<div class="language" data-lang={$language}>
  <span class="label">{$tr('settings.language')}</span>
  <SegmentedControl
    label={$tr('settings.language')}
    size="sm"
    value={chosen}
    options={options.map((c) => ({ value: c, label: $tr('settings.language.' + c) }))}
    onchange={(v) => pick(v as Choice)}
  />
  {#if failed}<span class="err" role="alert">{$tr('settings.language.failed')}</span>{/if}
</div>

<style>
  .language {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .label {
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .err {
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
