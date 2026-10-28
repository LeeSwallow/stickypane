<script lang="ts">
  // No boards yet: the design system's EmptyState — the three notes, one
  // line, one button. No explanation (R-32, 08 §8.2).
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import EmptyState from '../lib/ui/EmptyState.svelte'
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
</script>

<div class="empty">
  <EmptyState title={$tr('home.empty')}>
    <div class="actions">
      {#if setup}<Install kind={setup} {copy} />{/if}
      <Button variant="primary" size="lg" disabled={busy || !canPick} onclick={pick}>
        {busy ? $tr('home.initRunning') : $tr('home.pickFolder')}
      </Button>
    </div>
  </EmptyState>
</div>

<style>
  .empty {
    display: flex;
    justify-content: center;
    padding-top: var(--space-8);
  }
  .actions {
    display: grid;
    justify-items: center;
    gap: var(--space-4);
  }
</style>
