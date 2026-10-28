<script lang="ts">
  // A yes/no question on a taped note. A dangerous answer (delete, run) is
  // the red button and focus starts on Cancel, so Enter never deletes.
  import { tr } from '../../i18n'
  import Button from './Button.svelte'
  import Dialog from './Dialog.svelte'

  let {
    open = $bindable(false),
    title,
    message,
    confirmLabel,
    cancelLabel,
    danger = false,
    busy = false,
    inline = false,
    onconfirm,
    oncancel,
  }: {
    open?: boolean
    title: string
    message?: string
    confirmLabel: string
    cancelLabel?: string
    danger?: boolean
    busy?: boolean
    inline?: boolean
    onconfirm: () => void
    oncancel?: () => void
  } = $props()
</script>

<Dialog bind:open {title} {inline} role="alertdialog" width={380} onclose={oncancel}>
  {#if message}<p class="message">{message}</p>{/if}
  {#snippet actions()}
    <Button variant="ghost" data-autofocus={danger ? '' : undefined} onclick={() => ((open = false), oncancel?.())}>
      {cancelLabel ?? $tr('ui.cancel')}
    </Button>
    <Button variant={danger ? 'danger' : 'primary'} loading={busy} data-autofocus={danger ? undefined : ''} onclick={onconfirm}>
      {confirmLabel}
    </Button>
  {/snippet}
</Dialog>

<style>
  .message {
    margin: 0;
    white-space: pre-line;
    overflow-wrap: anywhere;
  }
</style>
