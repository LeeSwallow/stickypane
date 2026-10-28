<script lang="ts">
  import { language, tr, chooseLanguage, type Choice, type LanguageHost } from '../i18n'

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

<div class="language" role="radiogroup" aria-label={$tr('settings.language')} data-lang={$language}>
  {#each options as c (c)}
    <button type="button" role="radio" aria-checked={chosen === c} onclick={() => pick(c)}>
      {$tr('settings.language.' + c)}
    </button>
  {/each}
  {#if failed}<span class="err" role="alert">{$tr('settings.language.failed')}</span>{/if}
</div>

<style>
  .language {
    display: flex;
    gap: 8px;
  }
  .err {
    font-family: var(--font-hand);
    align-self: center;
  }
  button {
    font-family: var(--font-hand);
    font-size: 16px;
    padding: 4px 12px;
    color: var(--ink);
    background: transparent;
    border: 1.5px solid currentColor;
    border-radius: 8px;
    cursor: pointer;
  }
  button[aria-checked='true'] {
    font-weight: 700;
    background: color-mix(in srgb, currentColor 14%, transparent);
  }
</style>
