<script lang="ts">
  // One field of a shape's create form, drawn from its option schema:
  // text and number as a line, lines as a box (one value a line), choice as
  // segments, bool as a switch. It checks nothing: the shape's Go code
  // does, and its answer comes back as `error`.
  import { tr } from '../i18n'
  import Input from '../lib/ui/Input.svelte'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import TextArea from '../lib/ui/TextArea.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import type { ShapeOption } from '../services/create'
  import { choiceLabel, optionLabel } from './labels'

  let { option, value = $bindable(), error }: { option: ShapeOption; value: string | boolean; error?: string } = $props()

  const label = $derived(optionLabel($tr, option.label))
  const id = $props.id()
</script>

<div class="opt" data-option={option.name}>
  {#if option.type === 'bool'}
    <Toggle label={label} checked={value === true} onchange={(on) => (value = on)} />
    {#if error}<p class="err" role="alert">{error}</p>{/if}
  {:else if option.type === 'choice'}
    <span class="label" id="{id}-l">{label}</span>
    <SegmentedControl
      label={label}
      size="sm"
      value={String(value)}
      options={(option.choices ?? []).map((c) => ({ value: c, label: choiceLabel($tr, c) }))}
      onchange={(v) => (value = v)}
    />
    {#if error}<p class="err" role="alert">{error}</p>{/if}
  {:else if option.type === 'lines'}
    <TextArea label={label} rows={4} hint={$tr('create.linesHint')} {error} value={String(value)} oninput={(e) => (value = e.currentTarget.value)} />
  {:else}
    <Input
      label={label}
      {error}
      inputmode={option.type === 'number' ? 'decimal' : undefined}
      value={String(value)}
      oninput={(e) => (value = e.currentTarget.value)}
    />
  {/if}
</div>

<style>
  .opt {
    display: flex;
    flex-direction: column;
    gap: var(--space-1);
  }
  .label {
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .err {
    margin: 0;
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
