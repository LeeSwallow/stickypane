<script lang="ts">
  // A checklist (03 §3.8): hand-drawn boxes, a pen tick, done items struck
  // through in pencil, and a progress bar. A click (or Space) ticks an
  // item through note.act `toggle`; the bridge's board.changed redraws it.
  import { useBoard } from '../board/context'
  import { itemOrder, toggleAction, type ChecklistData, type Note } from '../board/model'
  import { language, relativeTime, tr } from '../i18n'
  import Check from '../lib/doodle/Check.svelte'
  import { seedOf } from '../lib/theme/tokens'

  let { note }: { note: Note } = $props()
  const board = useBoard()
  const data = $derived((note.data as ChecklistData | null) ?? { items: [], prose: [], done: 0, total: 0 })
  const order = $derived(itemOrder(data.items))
  let busy = $state(-1)

  async function toggle(i: number) {
    if (busy >= 0) return
    busy = i
    try {
      await board.act(note, toggleAction(i))
    } finally {
      busy = -1
    }
  }
</script>

{#each data.prose as line, i (i)}
  <p class="prose">{line}</p>
{/each}
{#if data.total > 0}
  <div class="progress" aria-hidden="true"><span style:width="{(100 * data.done) / data.total}%"></span></div>
  <span class="sr">{$tr('checklist.progress', { done: data.done, total: data.total })}</span>
{/if}
<ul>
  {#each order as i (i)}
    {@const item = data.items[i]}
    <li>
      <button
        class="item"
        class:done={item.checked}
        role="checkbox"
        aria-checked={item.checked}
        disabled={busy === i}
        onclick={() => toggle(i)}
      >
        <Check checked={item.checked} seed={seedOf(note.name + ':' + i)} />
        <span class="text">{item.text}</span>
        {#if item.checked && item.done}<span class="when">{relativeTime(new Date(item.done), board.now(), $language)}</span>{/if}
      </button>
    </li>
  {/each}
</ul>

<style>
  .prose {
    margin: 0 0 var(--space-1);
    color: var(--pencil);
    font: var(--text-body) / var(--line-body) var(--font-body);
  }
  .progress {
    height: var(--space-1);
    background: var(--grid-major);
    border-radius: var(--space-1);
    margin: var(--space-1) 0 var(--space-2);
  }
  .progress span {
    display: block;
    height: 100%;
    background: var(--good);
    border-radius: inherit;
  }
  ul {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .item {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    width: 100%;
    padding: var(--space-1) 0;
    border: 0;
    background: none;
    color: var(--ink);
    font: var(--text-md) / 1.35 var(--font-body);
    text-align: left;
    cursor: pointer;
  }
  .item:focus-visible {
    outline: var(--stroke) solid var(--accent);
  }
  .item.done .text {
    color: var(--pencil);
    text-decoration: line-through;
  }
  .when {
    margin-left: auto;
    color: var(--pencil);
    font: var(--text-xs) var(--font-mono);
  }
  .sr {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }
</style>
