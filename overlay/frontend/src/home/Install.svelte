<script lang="ts">
  // The install line (stickypane not found) or the update line (every copy
  // found is too old), with a copy button.
  import { tr } from '../i18n'

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
    <button type="button" onclick={doCopy}>{copied ? $tr('home.copied') : $tr('home.copy')}</button>
  </div>
</div>

<style>
  p {
    margin: 0 0 8px;
    font-family: var(--font-hand);
    font-size: 18px;
  }
  .line {
    display: flex;
    gap: 10px;
    align-items: center;
    flex-wrap: wrap;
  }
  code {
    font-family: var(--font-mono);
    font-size: 12.5px;
    padding: 4px 8px;
    background: color-mix(in srgb, var(--ink) 9%, transparent);
    border-radius: 4px;
    user-select: all;
  }
  button {
    font-family: var(--font-hand);
    font-size: 16px;
    padding: 2px 12px;
    color: var(--ink);
    background: transparent;
    border: 1.5px solid currentColor;
    border-radius: 8px;
    cursor: pointer;
  }
</style>
