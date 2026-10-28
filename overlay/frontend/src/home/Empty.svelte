<script lang="ts">
  // No boards yet: one note, one line, one button. No explanation (R-32).
  import { tr } from '../i18n'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Install from './Install.svelte'

  let {
    pick,
    copy,
    setup = null,
    busy = false,
    canPick = true,
  }: {
    pick: () => void
    copy: (text: string) => Promise<void>
    setup?: 'install' | 'update' | null
    busy?: boolean
    canPick?: boolean
  } = $props()
  const color = $derived(look.palette?.notes[0] ?? 'currentColor')
</script>

<div class="empty">
  <div class="note">
    <NoteFrame {color} id="empty">
      <p class="line">{$tr('home.empty')}</p>
      {#if setup}<Install kind={setup} {copy} />{/if}
      <button type="button" class="pick" disabled={busy || !canPick} onclick={pick}>
        {busy ? $tr('home.initRunning') : $tr('home.pickFolder')}
      </button>
    </NoteFrame>
  </div>
</div>

<style>
  .empty {
    display: flex;
    justify-content: center;
    padding: 48px 24px;
  }
  .note {
    width: min(420px, 100%);
  }
  .line {
    margin: 0 0 14px;
    font-family: var(--font-hand);
    font-size: 24px;
    font-weight: 700;
  }
  .pick {
    font-family: var(--font-hand);
    font-size: 20px;
    padding: 4px 18px;
    color: var(--ink);
    background: color-mix(in srgb, var(--accent) 22%, transparent);
    border: 2px solid var(--accent);
    border-radius: 10px;
    cursor: pointer;
  }
  .pick:disabled {
    opacity: 0.5;
    cursor: default;
  }
</style>
