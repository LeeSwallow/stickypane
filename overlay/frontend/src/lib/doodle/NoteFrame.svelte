<script lang="ts">
  // A note on the paper: a calm hand-drawn border in the note colour, taped on.
  import type { Snippet } from 'svelte'
  import { ROUGH, STROKE, tiltOf } from '../theme/scale'
  import { seedOf } from '../theme/tokens'
  import { look } from './look.svelte'
  import RoughBox from './RoughBox.svelte'
  import Tape from './Tape.svelte'

  // lean: only a note that holds no controls may lean (03 §3.11.3); a note with a header does not.
  let {
    color,
    id = 'note',
    tape = '50%',
    lean = false,
    children,
  }: { color: string; id?: string; tape?: string; lean?: boolean; children?: Snippet } = $props()
  const seed = $derived(seedOf(id))
  const tilt = $derived(lean ? tiltOf(seed, look.style) : 0)
</script>

<div class="note" style:transform="rotate({tilt}deg)">
  <Tape left={tape} />
  <RoughBox stroke={color} {seed} strokeWidth={STROKE.note} roughness={ROUGH.note} />
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
