<script lang="ts">
  // "+ 생성" (10 §10.3): pick a shape, fill in what it asks, make the note.
  // The shapes and their fields come from kinds.list; the dialog sends the
  // values with note.create and the shape's Go code writes the file
  // (RA12). Nothing is checked here: a value the shape refuses comes back
  // as the bridge's words, next to its field.
  import { tick } from 'svelte'
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import Dialog from '../lib/ui/Dialog.svelte'
  import Input from '../lib/ui/Input.svelte'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import { errorText } from '../services/errors'
  import { fieldOf, initialValues, reason, toOptions, type CreateApi, type FormValues, type Shape as ShapeT } from '../services/create'
  import type { Note } from '../board/model'
  import Shape from '../shapes/Shape.svelte'
  import { shapeBlurb, shapeLabel } from './labels'
  import OptionField from './OptionField.svelte'

  let {
    open = $bindable(false),
    api,
    tabs,
    tab = '',
    start,
    example = true,
    oncreated,
  }: {
    open?: boolean
    api: CreateApi
    /** the board's tabs: where the note can go */
    tabs: { name: string; label: string }[]
    /** the tab shown: where the note goes unless another is picked */
    tab?: string
    /** the shape picked first (the dev screens page) */
    start?: string
    /** draw the shape's example beside the form */
    example?: boolean
    oncreated: (note: string) => void
  } = $props()

  let shapes = $state<ShapeT[] | null>(null)
  let loadError = $state('')
  let picked = $state('')
  let title = $state('')
  let where = $state('')
  let values = $state<FormValues>({})
  let errors = $state<Record<string, string>>({})
  let failed = $state('')
  let busy = $state(false)
  let formEl: HTMLFormElement | undefined = $state()

  const shape = $derived(shapes?.find((s) => s.name === picked) ?? null)

  // each opening starts afresh, on the tab shown
  $effect(() => {
    if (!open) return
    title = ''
    where = tab
    errors = {}
    failed = ''
    loadError = ''
    api.kinds().then(
      (list) => {
        shapes = list
        pick(list.find((s) => s.name === start)?.name ?? list[0]?.name ?? '')
        // the title is the first thing to write, once there is a form
        tick().then(() => formEl?.querySelector<HTMLInputElement>('.form input')?.focus())
      },
      (e) => (loadError = errorText(e)),
    )
  })

  function pick(name: string) {
    picked = name
    const s = shapes?.find((x) => x.name === name)
    values = s ? initialValues(s) : {}
    errors = {}
    failed = ''
  }

  async function create() {
    if (!shape || busy) return
    busy = true
    errors = {}
    failed = ''
    try {
      const r = await api.create({ tab: where, kind: shape.name, title: title.trim(), options: toOptions(shape, values) })
      open = false
      oncreated(r.note)
    } catch (e) {
      const msg = errorText(e)
      const f = fieldOf(msg, shape)
      if (f) errors = { [f]: reason(msg) }
      else failed = msg
    } finally {
      busy = false
    }
  }

  const exampleNote = (s: ShapeT): Note =>
    ({
      name: `example:${s.name}`, tab: '', title: '', kind: s.name, icon: s.icon, open: true, size: '1', rows: 0, color: 'yellow',
      pinned: false, version: '', summary: s.example.split('\n').find((l) => l.trim() && l !== '---' && !/^\w+:/.test(l)) ?? '',
      data: s.data, modTime: '', linked: false,
    }) as Note
</script>

<Dialog bind:open title={$tr('create.dialog')} width={760}>
  {#if loadError}
    <p class="err" role="alert">{$tr('create.loadFailed', { error: loadError })}</p>
  {:else if !shapes}
    <p class="quiet">{$tr('ui.loading')}</p>
  {:else}
    <form class="create" bind:this={formEl} onsubmit={(e) => (e.preventDefault(), create())}>
      <div class="shapes" role="radiogroup" aria-label={$tr('create.shapes')}>
        {#each shapes as s (s.name)}
          <button
            type="button"
            role="radio"
            class="shape"
            class:on={s.name === picked}
            aria-checked={s.name === picked}
            data-shape={s.name}
            onclick={() => pick(s.name)}
          >
            <span class="icon" aria-hidden="true">{s.icon}</span>
            <span class="name">{shapeLabel($tr, s)}</span>
          </button>
        {/each}
      </div>
      {#if shape}
        <div class="form">
          <p class="blurb">{shapeBlurb($tr, shape)}</p>
          <Input label={$tr('create.titlePrompt')} hint={$tr('create.titleHint')} bind:value={title} data-autofocus />
          {#if tabs.length > 1}
            <div class="opt">
              <span class="label">{$tr('create.tab')}</span>
              <SegmentedControl
                label={$tr('create.tab')}
                size="sm"
                bind:value={where}
                options={tabs.map((t) => ({ value: t.name, label: t.label }))}
              />
            </div>
          {/if}
          {#each shape.options as o (shape.name + ':' + o.name)}
            <OptionField option={o} bind:value={values[o.name]} error={errors[o.name]} />
          {/each}
          {#if !shape.options.length}<p class="quiet">{$tr('create.noOptions')}</p>{/if}
          {#if shape.name === 'script' || shape.name === 'api'}<p class="quiet">{$tr('create.runNotice')}</p>{/if}
          {#if example}
            <div class="example">
              <span class="label">{$tr('create.example')}</span>
              <div class="sample" inert>
                {#key shape.name}<Shape note={exampleNote(shape)} />{/key}
              </div>
            </div>
          {/if}
          {#if failed}<p class="err" role="alert">{$tr('create.failed', { error: failed })}</p>{/if}
        </div>
      {/if}
      <button type="submit" hidden aria-hidden="true" tabindex="-1"></button>
    </form>
  {/if}
  {#snippet actions()}
    <Button variant="ghost" onclick={() => (open = false)}>{$tr('create.cancel')}</Button>
    <Button variant="primary" loading={busy} disabled={!shape} onclick={create}>{$tr('create.make')}</Button>
  {/snippet}
</Dialog>

<style>
  .create {
    display: grid;
    grid-template-columns: minmax(140px, 180px) 1fr;
    grid-template-rows: minmax(0, 1fr);
    gap: var(--space-5);
    max-height: min(68vh, 640px);
    min-height: 0;
  }
  .shapes {
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
    overflow: auto;
    padding: var(--space-1);
  }
  .shape {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-3);
    border: 1px solid transparent;
    border-radius: var(--radius-sm);
    background: none;
    color: var(--ink);
    text-align: left;
    cursor: pointer;
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .shape:hover {
    background: var(--grid-major);
  }
  .shape.on {
    border-color: var(--line-strong);
    background: var(--surface);
    font-weight: 600;
  }
  .shape:focus-visible {
    outline: var(--focus-width) solid var(--focus);
    outline-offset: var(--focus-offset);
  }
  .icon {
    width: 1.4em;
    text-align: center;
    font-family: var(--font-mono);
  }
  .form {
    min-height: 0;
    display: flex;
    flex-direction: column;
    gap: var(--space-4);
    overflow: auto;
    min-width: 0;
    padding: var(--space-1) var(--space-2) var(--space-1) 0;
  }
  .blurb {
    margin: 0;
    color: var(--ink-2);
  }
  .opt,
  .example {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .label {
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .sample {
    padding: var(--space-3);
    border: 1px dashed var(--line);
    border-radius: var(--radius-sm);
    max-height: 200px;
    overflow: hidden;
    color: var(--ink);
  }
  /* the example is to look at: its add lines are not there */
  .sample :global(.add) {
    display: none;
  }
  .quiet {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .err {
    margin: 0;
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  @media (max-width: 640px) {
    .create {
      grid-template-columns: 1fr;
    }
    .shapes {
      flex-direction: row;
      flex-wrap: wrap;
    }
  }
</style>
