<script lang="ts">
  // TEMPORARY: ?demo=look shows every doodle part in both styles with a theme and mode picker.
  // Not part of the app; it goes when the real screens exist.
  import { onMount } from 'svelte'
  import Arrow from '../lib/doodle/Arrow.svelte'
  import Check from '../lib/doodle/Check.svelte'
  import Circle from '../lib/doodle/Circle.svelte'
  import Crayon from '../lib/doodle/Crayon.svelte'
  import Logo from '../lib/doodle/Logo.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Mark from '../lib/doodle/Mark.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import Paper from '../lib/doodle/Paper.svelte'
  import Radio from '../lib/doodle/Radio.svelte'
  import RoughBox from '../lib/doodle/RoughBox.svelte'
  import Sheet from '../lib/doodle/Sheet.svelte'
  import Tape from '../lib/doodle/Tape.svelte'
  import Underline from '../lib/doodle/Underline.svelte'
  import { THEMES } from '../lib/theme/palettes'
  import { resolveThemeName, watchSystemDark, type Mode } from '../lib/theme/mode'
  import { agentColor, applyTheme } from '../lib/theme/tokens'

  const params = new URLSearchParams(location.search)
  let theme = $state(params.get('theme') ?? 'stickypane-dark')
  let mode = $state<Mode>((params.get('mode') as Mode) ?? 'system')
  let style = $state<'doodle' | 'neat'>(params.get('style') === 'neat' ? 'neat' : 'doodle')
  let systemDark = $state(false)

  onMount(() => watchSystemDark((d) => (systemDark = d)))

  const resolved = $derived(resolveThemeName(theme, mode, systemDark))
  const palette = $derived(THEMES.find((t) => t.name === resolved) ?? THEMES[0])

  $effect(() => {
    look.style = style
    look.palette = palette
    look.dark = palette.dark
    applyTheme(palette, palette.dark)
  })

  const noteNames = ['yellow', 'pink', 'blue', 'green', 'purple', 'orange']
  const agents = ['claude', 'codex', 'sam', 'gpt-5']
</script>

<Paper>
  <div class="bar">
    <Logo size={32} />
    <label>theme <select bind:value={theme}>{#each THEMES as t (t.name)}<option>{t.name}</option>{/each}</select></label>
    <label>mode <select bind:value={mode}><option>system</option><option>dark</option><option>light</option></select></label>
    <label>style <select bind:value={style}><option>doodle</option><option>neat</option></select></label>
    <span class="r">draws with: {resolved}</span>
  </div>

  <section>
    <h2>Sheets (crayon cards)</h2>
    <div class="row">
      {#each noteNames as n, i (n)}
        <div class="cell"><Sheet color={palette.notes[i]} id={n}>{n} card</Sheet></div>
      {/each}
    </div>
    <div class="row">
      {#each agents as a (a)}
        <div class="cell"><Sheet color={agentColor(a, palette.notes)} id={a}>@{a}</Sheet></div>
      {/each}
    </div>
  </section>

  <section>
    <h2>Notes (rough border + tape)</h2>
    <div class="row">
      {#each noteNames as n, i (n)}
        <div class="cell"><NoteFrame color={palette.notes[i]} id={n + 'f'}>{n} note<br />some text</NoteFrame></div>
      {/each}
    </div>
  </section>

  <section>
    <h2>Small parts</h2>
    <div class="row parts">
      <span class="p"><Check checked={false} /> todo</span>
      <span class="p"><Check checked seed={2} /> <s>done</s></span>
      <span class="p"><Radio /> option</span>
      <span class="p"><Radio selected seed={3} /> chosen</span>
      <span class="p word">underlined<Underline /></span>
      <span class="p word"><Mark color={palette.notes[0]} />highlighted</span>
      <span class="p word ring">stalled<Circle /></span>
      <span class="p"><Arrow /></span>
      <span class="p box">rough box<RoughBox stroke="var(--accent)" /></span>
      <span class="p box">filled<RoughBox stroke="var(--warn)" fill="var(--warn)" roughness={1.6} /></span>
      <span class="p box tapedbox"><Tape />tape</span>
      <span class="p box">crayon<Crayon color={palette.notes[2]} seed={9} /></span>
    </div>
  </section>

  <section>
    <h2>Tokens</h2>
    <div class="row">
      {#each ['--paper', '--base', '--ink', '--pencil', '--accent', '--select', '--good', '--warn', '--bad', '--info', '--grid', '--grid-major'] as v (v)}
        <span class="sw"><i style:background="var({v})"></i>{v}</span>
      {/each}
    </div>
  </section>
</Paper>

<style>
  .bar {
    display: flex;
    gap: 16px;
    align-items: center;
    padding: 12px 20px;
    font-family: var(--font-body);
  }
  .r {
    color: var(--pencil);
    margin-left: auto;
  }
  section {
    padding: 8px 24px 16px;
  }
  h2 {
    font-family: var(--font-hand);
    font-size: 22px;
    margin: 8px 0 12px;
  }
  .row {
    display: flex;
    flex-wrap: wrap;
    gap: 24px;
    align-items: center;
    margin-bottom: 18px;
  }
  .cell {
    width: 150px;
  }
  .parts {
    font-family: var(--font-hand);
    font-size: 18px;
    gap: 28px;
  }
  .p {
    position: relative;
    display: inline-flex;
    align-items: center;
    gap: 6px;
  }
  .word {
    display: inline-block;
  }
  .ring {
    padding: 4px 8px;
  }
  .box {
    padding: 10px 18px;
    min-width: 70px;
    justify-content: center;
  }
  .tapedbox {
    margin-top: 12px;
  }
  .sw {
    display: inline-flex;
    gap: 6px;
    align-items: center;
    font: 12px var(--font-mono);
  }
  .sw i {
    width: 22px;
    height: 22px;
    border: 1px solid var(--pencil);
    display: inline-block;
  }
</style>
