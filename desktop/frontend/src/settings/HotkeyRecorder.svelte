<script lang="ts">
  // The global hotkey, recorded (03 §3.12): press the field, then the keys.
  // Esc cancels. What is pressed is checked by the shell — a key with no
  // modifier, one macOS keeps, one another app holds — and said in place;
  // a hotkey that passes is saved at once.
  import { tr } from '../i18n'
  import type { HotkeyCheck } from '../services/settings'
  import { glyphs, modsOf, record } from './keys'

  let {
    value,
    check,
    save,
    label,
  }: {
    value: string
    check: (accel: string) => Promise<HotkeyCheck>
    save: (accel: string) => Promise<void>
    label: string
  } = $props()

  let listening = $state(false)
  let held = $state<string[]>([])
  let busy = $state('')
  let message = $state('')
  let bad = $state(false)

  const shown = $derived(listening ? held : glyphs(value))

  function start() {
    listening = true
    held = []
    message = ''
    bad = false
  }

  function stop() {
    listening = false
    held = []
  }

  async function onkeydown(e: KeyboardEvent) {
    if (!listening) {
      if (e.key === 'Enter' || e.key === ' ') {
        e.preventDefault()
        start()
      }
      return
    }
    e.preventDefault()
    e.stopPropagation()
    const r = record(e)
    if (r.kind === 'cancel') return stop()
    if (r.kind === 'held') {
      held = glyphs(r.mods.join('+') + (r.mods.length ? '+' : '')).filter(Boolean)
      return
    }
    stop()
    busy = r.accel
    try {
      const c = await check(r.accel)
      if (!c.ok) {
        bad = true
        message = $tr('settings.hotkey.' + (c.code || 'key'), { key: glyphs(c.accel || r.accel).join('') })
        return
      }
      await save(c.accel)
      bad = false
      message = $tr('settings.hotkey.set', { key: glyphs(c.accel).join('') })
    } catch (err) {
      bad = true
      message = $tr('settings.hotkey.failed', { error: String((err as Error)?.message ?? err) })
    } finally {
      busy = ''
    }
  }

  function onkeyup(e: KeyboardEvent) {
    if (listening) held = glyphs(modsOf(e).join("+"))
  }
</script>

<div class="recorder">
  <button
    type="button"
    class="field"
    class:listening
    class:bad
    aria-label={listening ? $tr('settings.hotkey.listening') : `${label}: ${glyphs(value).join('')}. ${$tr('settings.hotkey.record')}`}
    aria-describedby="hotkey-message"
    onclick={() => (listening ? stop() : start())}
    {onkeydown}
    {onkeyup}
    onblur={stop}
  >
    {#if busy}
      <span class="say">{$tr('settings.hotkey.checking', { key: glyphs(busy).join('') })}</span>
    {:else if listening && !held.length}
      <span class="say">{$tr('settings.hotkey.listening')}</span>
    {:else}
      {#each shown as g, i (i)}<kbd>{g}</kbd>{/each}
    {/if}
  </button>
  <p id="hotkey-message" class="message" class:bad role={bad ? 'alert' : 'status'}>{message}</p>
</div>

<style>
  .recorder {
    display: flex;
    flex-direction: column;
    align-items: flex-end;
    gap: var(--space-1);
  }
  .field {
    display: inline-flex;
    align-items: center;
    justify-content: center;
    gap: 3px;
    min-width: 168px;
    height: var(--control-lg);
    padding: 0 var(--space-3);
    border: 1px dashed var(--line-strong);
    border-radius: var(--radius-md);
    background: var(--surface-sunk);
    color: var(--ink);
    cursor: pointer;
  }
  .field:hover {
    border-color: var(--accent);
  }
  .field:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .field.listening {
    border-style: solid;
    border-color: var(--accent);
    background: var(--accent-wash);
  }
  .field.bad {
    border-color: var(--danger-text);
  }
  kbd {
    min-width: 22px;
    padding: 1px var(--space-1);
    border: 1px solid var(--line-strong);
    border-bottom-width: 2px;
    border-radius: var(--radius-sm);
    background: var(--surface);
    color: var(--ink);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    text-align: center;
  }
  .say {
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .message {
    min-height: var(--leading-body-sm);
    margin: 0;
    max-width: 32ch;
    color: var(--good-text);
    text-align: right;
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .message.bad {
    color: var(--danger-text);
  }
</style>
