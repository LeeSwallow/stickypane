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

  async function pick(c: Choice) {
    chosen = c
    await chooseLanguage(c, host)
  }
</script>

<div class="language" role="radiogroup" aria-label={$tr('settings.language')} data-lang={$language}>
  {#each options as c (c)}
    <button type="button" role="radio" aria-checked={chosen === c} onclick={() => pick(c)}>
      {$tr('settings.language.' + c)}
    </button>
  {/each}
</div>

<style>
  .language {
    display: flex;
    gap: 8px;
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
