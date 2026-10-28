<script lang="ts">
  // A note on the paper: a hand-drawn border in the note colour, taped on, tilted per note.
  import type { Snippet } from 'svelte'
  import { seedOf } from '../theme/tokens'
  import { look } from './look.svelte'
  import RoughBox from './RoughBox.svelte'
  import Tape from './Tape.svelte'

  let { color, id = 'note', tape = '50%', children }: { color: string; id?: string; tape?: string; children?: Snippet } = $props()
  const seed = $derived(seedOf(id))
  const tilt = $derived(look.style === 'neat' ? 0 : ((seed % 9) - 4) * 0.45)
</script>

<div class="note" style:transform="rotate({tilt}deg)">
  <Tape left={tape} />
  <RoughBox stroke={color} {seed} strokeWidth={2} />
  <div class="body">{@render children?.()}</div>
</div>

<style>
  .note {
    position: relative;
    background: var(--paper);
    padding: 14px 16px;
  }
  .body {
    position: relative;
  }
</style>
