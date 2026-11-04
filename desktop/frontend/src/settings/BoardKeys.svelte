<script lang="ts">
  // 단축키: the terminal board's keys (docs/configuration.md#keys), listed
  // and changed through the app's own stickypane (`stickypane keys`), so
  // the board checks every change and keys.json stays the one place they
  // live. Each action: its words, its keys, a mark when keys.json changed
  // it, and 바꾸기 / 기본값. The editor takes key names (commas between
  // keys, spaces inside a sequence) or records what is pressed. The app's
  // own hotkey is 이 컴퓨터's; it is named first, with the way there.
  import { onMount } from 'svelte'
  import { language, tr, type Lang } from '../i18n'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Confirm from '../lib/ui/Confirm.svelte'
  import IconButton from '../lib/ui/IconButton.svelte'
  import Input from '../lib/ui/Input.svelte'
  import Kbd from '../lib/ui/Kbd.svelte'
  import { report } from '../services/errors'
  import { GROUPS, formatKeys, keyName, parseKeys, showKey, type Binding, type Keys, type KeysApi } from '../services/keys'
  import type { Section } from '../services/settings'
  import { glyphs } from './keys'

  let {
    api,
    hotkey = '',
    goto,
    pause = 800,
  }: { api: KeysApi; hotkey?: string; goto: (s: Section) => void; pause?: number } = $props()

  let keys = $state<Keys | null>(null)
  let failed = $state('')
  let query = $state('')
  let busy = $state('')
  // the action being edited, what its field holds, and why the board refused it
  let editing = $state('')
  let text = $state('')
  let refused = $state('')
  let unbinding = $state(false)
  let resetting = $state(false)
  let recording = $state(false)
  let pressed: string[] = []
  let first = false
  let base = ''
  let timer: ReturnType<typeof setTimeout> | undefined
  // the language the words come in: the app's, asked again when it changes
  let lang: Lang = 'en'

  // the answer of the last ask wins: the language may change while an
  // older one is still on its way
  let asked = 0

  async function load() {
    failed = ''
    const n = ++asked
    try {
      const k = await api.keys(lang)
      if (n === asked) keys = k
    } catch (e) {
      if (n === asked) failed = report('keys.load', e, (error) => $tr('keys.failed', { error }))
    }
  }
  onMount(() =>
    language.subscribe((l) => {
      lang = l
      void load()
    }),
  )

  const changed = $derived(keys?.bindings.some((b) => b.changed) ?? false)
  const q = $derived(query.trim().toLowerCase())
  const shown = $derived(
    (keys?.bindings ?? []).filter(
      (b) => !q || b.words.toLowerCase().includes(q) || b.name.includes(q) || b.keys.some((k) => k.toLowerCase() === q || showKey(k).toLowerCase() === q),
    ),
  )
  const groups = $derived.by(() => {
    const out: { id: string; items: Binding[] }[] = []
    for (const id of [...GROUPS, '']) {
      const items = shown.filter((b) => (id ? b.group === id : !GROUPS.includes(b.group as (typeof GROUPS)[number])))
      if (items.length) out.push({ id, items })
    }
    return out
  })

  function edit(b: Binding) {
    stopRecording()
    editing = b.name
    text = formatKeys(b.keys)
    refused = ''
  }

  function cancel() {
    stopRecording()
    editing = refused = ''
    unbinding = false
  }

  async function change(name: string, f: () => Promise<Keys>) {
    busy = name
    const n = ++asked
    try {
      const k = await f()
      if (n === asked) keys = k
      return true
    } catch (e) {
      const why = report('keys.change', e)
      if (editing === name) refused = why
      else failed = $tr('keys.failed', { error: why })
      return false
    } finally {
      busy = ''
    }
  }

  async function save(e?: SubmitEvent) {
    e?.preventDefault()
    stopRecording()
    const list = parseKeys(text)
    if (!list.length && !unbinding) {
      unbinding = true
      return
    }
    unbinding = false
    const name = editing
    if (await change(name, () => api.setKeys(lang, name, list))) editing = ''
  }

  const reset = (name: string) => change(name, () => api.resetKeys(lang, name))

  async function resetAll() {
    resetting = false
    cancel()
    await change('*', () => api.resetKeys(lang, ''))
  }

  // --- recording: what is pressed in the field ----------------------------

  let field: HTMLElement | undefined = $state()

  function startRecording() {
    recording = true
    first = true
    pressed = []
    refused = ''
    field?.querySelector('input')?.focus()
  }

  function stopRecording() {
    clearTimeout(timer)
    endKey()
    recording = false
  }

  // a pause ends a key: what was pressed before it is one key, a sequence
  // when it is several. The first key of a recording takes the field's
  // place; the next ones come after a comma.
  function endKey() {
    pressed = []
    first = false
  }

  function onkeydown(e: KeyboardEvent) {
    if (!recording) return
    e.preventDefault()
    e.stopPropagation()
    const k = keyName(e)
    if (!k) return
    if (!pressed.length) base = first ? '' : text.trim().replace(/,\s*$/, '')
    pressed.push(k)
    text = (base ? base + ', ' : '') + pressed.join(' ')
    clearTimeout(timer)
    timer = setTimeout(endKey, pause)
  }
</script>

<div class="keys">
  <div class="app-key">
    <div class="what">
      <span class="label">{$tr('keys.app')}</span>
      <p class="hint">{$tr('keys.appHint')}</p>
    </div>
    <span class="chips">
      {#if hotkey}<Kbd keys={glyphs(hotkey).join('')} />{:else}<span class="quiet small">{$tr('keys.none')}</span>{/if}
    </span>
    <Button size="sm" variant="ghost" onclick={() => goto('app')}>{$tr('keys.appChange')}</Button>
  </div>

  {#if keys?.missing}
    <div class="missing">
      <p class="warn" role="note">{$tr('keys.missing')}</p>
      <div class="buttons">
        <Button size="sm" variant="secondary" onclick={() => goto('cli')}>{$tr('keys.toCli')}</Button>
      </div>
    </div>
  {:else if !keys}
    {#if failed}
      <p class="err" role="alert">{failed}</p>
      <div class="buttons"><Button size="sm" variant="secondary" onclick={load}>{$tr('keys.retry')}</Button></div>
    {:else}
      <p class="quiet">{$tr('keys.loading')}</p>
    {/if}
  {:else}
    <div class="bar">
      <div class="search">
        <Input label={$tr('keys.search')} hideLabel icon="search" placeholder={$tr('keys.search')} bind:value={query} spellcheck={false} autocapitalize="off" />
      </div>
      <Button size="sm" variant="ghost" icon="restore" disabled={!changed || !!busy} loading={busy === '*'} onclick={() => (resetting = true)}>{$tr('keys.resetAll')}</Button>
    </div>
    {#each keys.problems ?? [] as p (p)}
      <p class="warn" role="note">{p}</p>
    {/each}

    {#each groups as g (g.id)}
      <section class="group" aria-label={$tr('keys.group.' + (g.id || 'other'))}>
        <h2>{$tr('keys.group.' + (g.id || 'other'))}</h2>
        <ul>
          {#each g.items as b (b.name)}
            <li class:open={editing === b.name}>
              <div class="row">
                <div class="what">
                  <span class="words">{b.words}</span>
                  <code class="name">{b.name}</code>
                </div>
                {#if b.changed}<Badge tone="accent">{$tr('keys.changed')}</Badge>{/if}
                <span class="chips" aria-label={$tr('keys.keysOf', { action: b.words })}>
                  {#each b.keys as k (k)}<Kbd keys={showKey(k)} />{:else}<span class="quiet small">{$tr('keys.none')}</span>{/each}
                </span>
                <span class="acts">
                  {#if b.changed}
                    <IconButton size="sm" align="end" icon="restore" label={$tr('keys.reset', { keys: (b.defaults ?? []).map(showKey).join(' ') || '—' })} disabled={!!busy} onclick={() => reset(b.name)} />
                  {/if}
                  <IconButton size="sm" align="end" icon="edit-external" label={$tr('keys.edit', { action: b.words })} disabled={!!busy} onclick={() => edit(b)} />
                </span>
              </div>
              {#if editing === b.name}
                <form class="editor" onsubmit={save} aria-label={$tr('keys.edit', { action: b.words })}>
                  <div class="field" bind:this={field}>
                    <Input
                      label={$tr('keys.field')}
                      hint={recording ? $tr('keys.recording') : $tr('keys.fieldHint')}
                      error={refused}
                      bind:value={text}
                      spellcheck={false}
                      autocapitalize="off"
                      autocomplete="off"
                      {onkeydown}
                      onblur={() => recording && stopRecording()}
                    />
                  </div>
                  {#if b.defaults?.length}<p class="small quiet">{$tr('keys.defaults', { keys: b.defaults.map(showKey).join(' ') })}</p>{/if}
                  <div class="buttons">
                    <Button size="sm" variant={recording ? 'secondary' : 'ghost'} onclick={() => (recording ? stopRecording() : startRecording())}>{recording ? $tr('keys.stop') : $tr('keys.record')}</Button>
                    <span class="grow"></span>
                    <Button size="sm" variant="ghost" onclick={cancel}>{$tr('ui.cancel')}</Button>
                    <Button size="sm" variant="primary" type="submit" loading={busy === b.name}>{$tr('keys.save')}</Button>
                  </div>
                  {#if unbinding}
                    <Confirm
                      open
                      inline
                      title={$tr('keys.unbind', { action: b.words })}
                      message={$tr('keys.unbindBody')}
                      confirmLabel={$tr('keys.unbindYes')}
                      danger
                      onconfirm={() => save()}
                      oncancel={() => (unbinding = false)}
                    />
                  {/if}
                </form>
              {/if}
            </li>
          {/each}
        </ul>
      </section>
    {:else}
      <p class="quiet">{$tr('keys.noMatch')}</p>
    {/each}
    {#if failed}<p class="err" role="alert">{failed}</p>{/if}
  {/if}
</div>

<Confirm
  open={resetting}
  title={$tr('keys.resetAllTitle')}
  message={$tr('keys.resetAllBody')}
  confirmLabel={$tr('keys.resetAll')}
  danger
  onconfirm={resetAll}
  oncancel={() => (resetting = false)}
/>

<style>
  .keys {
    display: flex;
    flex-direction: column;
    gap: var(--space-3);
  }
  .app-key {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-3) 0 var(--space-4);
    border-bottom: var(--stroke-hairline) solid var(--line);
  }
  .what {
    flex: 1;
    min-width: 0;
  }
  .label {
    color: var(--ink);
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .hint {
    margin: 2px 0 0;
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .bar {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .search {
    flex: 1;
    min-width: 0;
  }
  .group h2 {
    margin: var(--space-3) 0 var(--space-1);
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    letter-spacing: 0.02em;
  }
  ul {
    margin: 0;
    padding: 0;
    list-style: none;
  }
  li {
    border-top: var(--stroke-hairline) solid var(--line);
  }
  li:first-child {
    border-top: 0;
  }
  li.open {
    background: var(--surface-sunk);
    border-radius: var(--radius-md);
  }
  .row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    min-height: 40px;
    padding: var(--space-1) var(--space-2);
  }
  .row .what {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: 2px var(--space-2);
  }
  .words {
    color: var(--ink);
    font: 500 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .name {
    color: var(--ink-3);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .chips {
    display: inline-flex;
    flex-wrap: wrap;
    justify-content: flex-end;
    gap: var(--space-1);
    min-width: 72px;
  }
  .acts {
    display: inline-flex;
    flex: none;
    justify-content: flex-end;
    width: 64px;
    gap: 2px;
  }
  .editor {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    padding: 0 var(--space-2) var(--space-3);
  }
  .buttons {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .grow {
    flex: 1;
  }
  .quiet {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .small {
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .missing {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  .warn,
  .err {
    margin: 0;
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    overflow-wrap: anywhere;
  }
  .warn {
    padding: var(--space-2) var(--space-3);
    border-left: 3px solid var(--warn);
    border-radius: var(--radius-sm);
    background: var(--warn-wash);
    color: var(--ink);
  }
  .err {
    color: var(--danger-text);
  }
</style>
