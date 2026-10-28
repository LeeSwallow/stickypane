<script lang="ts">
  // The tabs (folders, as on the terminal), drawn by the design system's
  // Tabs, and below them the closed notes of the tab as small buttons: a
  // click opens one (10 §10.1). A tab with a changed note carries a dot.
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import UiTabs from '../lib/ui/Tabs.svelte'
  import type { Note, Tab } from './model'

  let {
    tabs,
    current,
    changed,
    closed,
    onselect,
    onopen,
  }: {
    tabs: Tab[]
    current: string
    changed: (tab: Tab) => boolean
    closed: Note[]
    onselect: (tab: string) => void
    onopen: (note: string) => void
  } = $props()
  const label = (t: Tab) => t.title || t.name.replace(/^\d+[-_. ]*/, '') || $tr('board.tabRoot')
  const items = $derived(
    tabs.map((t, i) => ({
      id: t.name,
      label: label(t),
      count: t.notes.length,
      changed: changed(t),
      hint: i < 9 ? `⌘${i + 1}` : undefined,
    })),
  )
</script>

{#if tabs.length > 1 || closed.length}
  <div class="row">
    {#if tabs.length > 1}
      <UiTabs tabs={items} value={current} label={$tr('board.tabs')} changedLabel={$tr('board.changedTab')} onchange={onselect} />
    {/if}
    {#if closed.length}
      <div class="shelf" role="group" aria-label={$tr('board.shelf')}>
        {#each closed as n (n.name)}
          <Button variant="ghost" size="sm" title={$tr('note.open', { title: n.title })} onclick={() => onopen(n.name)}>
            <span class="closed">{n.icon} {n.title}</span>
          </Button>
        {/each}
      </div>
    {/if}
  </div>
{/if}

<style>
  .row {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-1) var(--space-4);
    padding: 0 var(--space-4) var(--space-1);
    border-bottom: var(--stroke-hairline) solid var(--line);
    background: var(--chrome);
  }
  .shelf {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-1);
    margin-left: auto;
  }
  .closed {
    max-width: 16em;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--ink-2);
  }
</style>
