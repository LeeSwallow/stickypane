<script lang="ts">
  // A quiet line at the end of a list: type and press Enter to add one
  // (a checklist item, a card). The shape's `add` does the writing.
  let { label, onadd }: { label: string; onadd: (text: string) => Promise<void> } = $props()
  let text = $state('')
  let busy = $state(false)

  async function add(e: KeyboardEvent) {
    if (e.key === 'Escape') {
      text = ''
      ;(e.currentTarget as HTMLInputElement).blur()
      return
    }
    if (e.key !== 'Enter' || e.isComposing || !text.trim() || busy) return
    e.preventDefault()
    busy = true
    try {
      await onadd(text.trim())
      text = ''
    } finally {
      busy = false
    }
  }
</script>

<input class="add" placeholder={'+ ' + label} aria-label={label} bind:value={text} disabled={busy} onkeydown={add} />

<style>
  .add {
    width: 100%;
    box-sizing: border-box;
    margin-top: var(--space-1);
    padding: var(--space-1) 0;
    border: 0;
    border-bottom: 1px dashed transparent;
    background: none;
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    outline: none;
    --wails-draggable: no-drag;
  }
  .add::placeholder {
    color: var(--pencil);
  }
  .add:hover,
  .add:focus {
    border-bottom-color: var(--line-strong);
  }
</style>
