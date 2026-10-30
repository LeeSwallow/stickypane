<script lang="ts">
  // A radio group of picture cards (ChoiceCard): one tab stop, the arrows
  // move to the next card and pick it, as a radio group does. min is the
  // narrowest a card gets; the cards wrap into as many columns as fit.
  import type { Snippet } from 'svelte'

  let { label, min = 132, children }: { label: string; min?: number; children: Snippet } = $props()
  let el: HTMLElement | undefined = $state()

  function onkeydown(e: KeyboardEvent) {
    const next = { ArrowRight: 1, ArrowDown: 1, ArrowLeft: -1, ArrowUp: -1 }[e.key]
    const items = [...(el?.querySelectorAll<HTMLButtonElement>('[role="radio"]') ?? [])]
    if (!items.length) return
    const i = items.indexOf(document.activeElement as HTMLButtonElement)
    let j: number | null = null
    if (next !== undefined) j = (Math.max(i, 0) + next + items.length) % items.length
    else if (e.key === 'Home') j = 0
    else if (e.key === 'End') j = items.length - 1
    if (j === null) return
    e.preventDefault()
    items[j].focus()
    items[j].click()
  }

  // with nothing chosen, the first card is the group's tab stop
  $effect(() => {
    const items = [...(el?.querySelectorAll<HTMLButtonElement>('[role="radio"]') ?? [])]
    if (items.length && !items.some((b) => b.getAttribute('aria-checked') === 'true')) items[0].tabIndex = 0
  })
</script>

<div bind:this={el} class="group" role="radiogroup" aria-label={label} tabindex="-1" style:--min="{min}px" {onkeydown}>
  {@render children()}
</div>

<style>
  .group {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(var(--min), 1fr));
    gap: var(--space-3);
  }
  .group:focus {
    outline: none;
  }
</style>
