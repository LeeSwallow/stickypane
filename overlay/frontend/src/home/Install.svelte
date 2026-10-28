<script lang="ts">
  // The install line (stickypane not found) or the update line (every copy
  // found is too old), with a copy button.
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'

  let { kind, copy }: { kind: 'install' | 'update'; copy: (text: string) => Promise<void> } = $props()
  const command = $derived($tr(kind === 'install' ? 'home.installCommand' : 'home.updateCommand'))
  let copied = $state(false)

  async function doCopy() {
    try {
      await copy(command)
      copied = true
    } catch {
      copied = false
    }
  }
</script>

<div class="install">
  <p>{$tr(kind === 'install' ? 'home.installHint' : 'home.updateHint')}</p>
  <div class="line">
    <code>{command}</code>
    <Button size="sm" icon={copied ? 'check' : undefined} onclick={doCopy}>{copied ? $tr('home.copied') : $tr('home.copy')}</Button>
  </div>
</div>

<style>
  .install {
    text-align: left;
  }
  p {
    margin: 0 0 var(--space-2);
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .line {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    flex-wrap: wrap;
  }
  code {
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    color: var(--ink);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    user-select: all;
  }
</style>
