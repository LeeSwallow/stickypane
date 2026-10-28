<script lang="ts">
  // A chat (03 §3.8, 10 §10.4): no bubbles, one "@name HH:MM text" line a
  // message under the heading of its day, the name in its agent colour,
  // handwritten. The line at the end says something (note.act `say`);
  // the bridge names the person as the terminal does. It follows the end
  // like a log.
  import { useBoard } from '../board/context'
  import type { Note } from '../board/model'
  import { dayLabel, language, tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { agentColor, noteTitleColor } from '../lib/theme/tokens'
  import AddLine from './AddLine.svelte'
  import { dataOf, type ChatData, type ChatMessage } from './data'
  import { follow } from './follow'

  let { note }: { note: Note } = $props()
  const board = useBoard()
  const messages = $derived(dataOf<ChatData>(note.data, { messages: [] }).messages ?? [])
  const days = $derived.by(() => {
    const out: { day: string; messages: ChatMessage[] }[] = []
    for (const m of messages) {
      if (out.at(-1)?.day !== m.day) out.push({ day: m.day, messages: [] })
      out[out.length - 1].messages.push(m)
    }
    return out
  })
  const colorOf = (who: string) =>
    look.palette ? noteTitleColor(agentColor(who, look.palette.notes), look.palette, look.dark) : 'var(--accent)'
</script>

<div class="chat" use:follow={messages.length}>
  {#if !messages.length}
    <p class="empty">{$tr('chat.empty')}</p>
  {/if}
  {#each days as d, di (di)}
    <section aria-label={d.day}>
      {#if d.day}<h4 class="day">{dayLabel(d.day, board.now(), $language)}</h4>{/if}
      {#each d.messages as m, i (i)}
        <p class="msg">
          <span class="who" class:neat={look.style === 'neat'} style:color={colorOf(m.who)}>@{m.who}</span>
          <time class="clock">{m.clock}</time>
          <span class="text">{m.text}</span>
        </p>
      {/each}
    </section>
  {/each}
</div>
<AddLine label={$tr('chat.say')} onadd={(text) => board.act(note, { name: 'say', args: { text } })} />

<style>
  .day {
    margin: var(--space-2) 0 var(--space-1);
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    text-align: center;
  }
  section:first-child .day {
    margin-top: 0;
  }
  .msg {
    margin: 0 0 var(--space-1);
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    overflow-wrap: anywhere;
  }
  .who {
    font: var(--weight-hand-label) var(--text-hand-label) / 1 var(--font-hand);
  }
  .who.neat {
    font: 600 var(--text-body-md) / 1 var(--font-body);
  }
  .clock {
    margin: 0 var(--space-1);
    color: var(--pencil);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
  }
  .empty {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
