<script lang="ts">
  // A form (03 §3.8, 10 §10.4): questions handwritten, options as
  // hand-drawn radios and checkboxes, "> " lines as fields, and its
  // buttons. Every change is one note.act: `choose` flips an option (a
  // radio clears the others of its question), `fill` writes a field when
  // it is left or Enter is pressed, `submit` presses a button. Changing an
  // answer after a submit takes the submit back, as on the terminal; the
  // shape does that, not the app. Waiting or answered is said in words.
  import { useBoard } from '../board/context'
  import type { Note } from '../board/model'
  import { language, relativeTime, tr } from '../i18n'
  import Check from '../lib/doodle/Check.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Radio from '../lib/doodle/Radio.svelte'
  import { seedOf } from '../lib/theme/tokens'
  import Button from '../lib/ui/Button.svelte'
  import { dataOf, type FormData } from './data'

  let { note }: { note: Note } = $props()
  const board = useBoard()
  const data = $derived(dataOf<FormData>(note.data, { lines: [], submitted: '', at: '' }))
  const lines = $derived(data.lines ?? [])
  let busy = $state(false)

  async function act(name: string, args: Record<string, string>) {
    if (busy) return
    busy = true
    try {
      await board.act(note, { name, args })
    } finally {
      busy = false
    }
  }

  const heading = (text: string) => text.match(/^#{1,6}\s+(.*)$/)?.[1] ?? null

  function fieldKey(e: KeyboardEvent, old: string) {
    const el = e.currentTarget as HTMLInputElement
    if (e.key === 'Enter' && !e.isComposing) {
      e.preventDefault()
      el.blur()
    } else if (e.key === 'Escape') {
      el.value = old
      el.blur()
    }
  }

  function fieldDone(e: FocusEvent, i: number, old: string) {
    const text = (e.currentTarget as HTMLInputElement).value.trim()
    if (text !== old) void act('fill', { line: String(i), text })
  }

  const when = $derived(data.at ? relativeTime(new Date(data.at), board.now(), $language) : '')
</script>

<div class="form" class:neat={look.style === 'neat'} aria-busy={busy || undefined}>
  {#each lines as line, i (i)}
    {#if line.kind === 'option'}
      <button
        class="option"
        role={line.radio ? 'radio' : 'checkbox'}
        aria-checked={!!line.checked}
        disabled={busy}
        onclick={() => act('choose', { line: String(i) })}
      >
        {#if line.radio}
          <Radio selected={!!line.checked} seed={seedOf(note.name + ':' + i)} />
        {:else}
          <Check checked={!!line.checked} seed={seedOf(note.name + ':' + i)} />
        {/if}
        <span>{line.text}</span>
      </button>
    {:else if line.kind === 'detail'}
      <p class="detail">{line.text}</p>
    {:else if line.kind === 'field'}
      {#key line.text}
        <input
          class="field"
          value={line.text}
          aria-label={line.group || $tr('form.answer')}
          placeholder={$tr('form.answer')}
          disabled={busy}
          onkeydown={(e) => fieldKey(e, line.text)}
          onblur={(e) => fieldDone(e, i, line.text)}
        />
      {/key}
    {:else if line.kind === 'buttons'}
      <div class="buttons">
        {#each line.labels ?? [] as label, b (b)}
          <Button
            variant={b === 0 ? 'primary' : 'secondary'}
            size="sm"
            disabled={busy}
            aria-pressed={data.submitted === label}
            onclick={() => act('submit', { label })}>{label}</Button
          >
        {/each}
      </div>
    {:else if heading(line.text) !== null}
      <h4 class="question">{heading(line.text)}</h4>
    {:else if line.text.trim()}
      <p class="text">{line.text}</p>
    {/if}
  {/each}
  <p class="state" class:done={!!data.submitted} role="status">
    {#if data.submitted}
      {$tr('form.answered', { button: data.submitted })}{#if when}<span class="when"> · {when}</span>{/if}
    {:else}
      {$tr('form.waiting')}
    {/if}
  </p>
</div>

<style>
  .form {
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .question {
    margin: var(--space-2) 0 var(--space-1);
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
  }
  .neat .question {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .question:first-child {
    margin-top: 0;
  }
  .text {
    margin: 0 0 var(--space-1);
  }
  .option {
    display: flex;
    gap: var(--space-2);
    align-items: center;
    width: 100%;
    padding: var(--space-1) 0;
    border: 0;
    background: none;
    color: inherit;
    font: inherit;
    text-align: left;
    cursor: pointer;
    --wails-draggable: no-drag;
  }
  .option:focus-visible,
  .field:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .detail {
    margin: 0 0 var(--space-1) calc(18px + var(--space-2));
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .field {
    display: block;
    width: 100%;
    box-sizing: border-box;
    margin: var(--space-1) 0;
    padding: var(--space-1) var(--space-2);
    border: var(--stroke-hairline) solid var(--line-strong);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    color: var(--ink);
    font: inherit;
    --wails-draggable: no-drag;
  }
  .buttons {
    display: flex;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin: var(--space-2) 0 var(--space-1);
  }
  .state {
    margin: var(--space-1) 0 0;
    color: var(--warn-text, var(--warn));
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .state.done {
    color: var(--good-text, var(--good));
  }
  .when {
    color: var(--pencil);
  }
</style>
