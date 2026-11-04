<script lang="ts">
  // ⋯ → 옮기기: where a note goes (10 §10.3) — a tab, a book or a new
  // name, written or picked from the board's tabs. The board window and a
  // torn note ask the same; manage.move does it.
  import { t, tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import Dialog from '../lib/ui/Dialog.svelte'
  import Input from '../lib/ui/Input.svelte'
  import { moveTargets } from './manage'
  import type { Note, Tab } from './model'

  let {
    note,
    tabs,
    width,
    onmove,
    onclose,
  }: { note: Note; tabs: Tab[]; width: number; onmove: (to: string) => void; onclose: () => void } = $props()

  let to = $state('')
  const targets = $derived(moveTargets(tabs, note.tab))
  // the window closes the dialog; an empty name moves nothing
  const move = () => onmove(to)
</script>

<Dialog open title={t('note.moveTitle', { title: note.title })} {width} {onclose}>
  <form class="move" onsubmit={(e) => (e.preventDefault(), move())}>
    <Input label={$tr('note.moveTo')} hint={$tr('note.moveHint')} bind:value={to} data-autofocus />
    {#if targets.length}
      <div class="targets">
        {#each targets as target (target)}
          <Button size="sm" variant="secondary" icon="tab" onclick={() => (to = target)}>{target === '.' ? $tr('note.moveTop') : target}</Button>
        {/each}
      </div>
    {/if}
  </form>
  {#snippet actions()}
    <Button variant="ghost" onclick={onclose}>{$tr('ui.cancel')}</Button>
    <Button variant="primary" disabled={!to.trim()} onclick={move}>{$tr('note.move').replace(/…$/, '')}</Button>
  {/snippet}
</Dialog>

<style>
  .move {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  .targets {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
  }
</style>
