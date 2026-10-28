<script lang="ts">
  // Dev only (?demo=motion&mode=dark|light&style=doodle|neat&lang=en|ko
  // &motion=reduce|off&scene=<name>|all&intro=full|short|fade
  // &view=canvas|torn): the real
  // board over a few fixture notes, with a fake bridge that sends the
  // board.changed of each scene (motionScenes.ts), on the grid, the free
  // canvas or a torn-off window of work.md, so every movement of
  // 04 §4.3 can be seen and filmed. scene= plays one 700 ms after the
  // page opens (all: each in turn); intro= plays the launch intro. The
  // plans played are in window.__motionPlans, and window.__motionPlay(name)
  // plays a scene, for a headless check.
  // Nothing here ships.
  import Board from '../board/Board.svelte'
  import type { Board as BoardData } from '../board/model'
  import Intro from '../lib/motion/Intro.svelte'
  import TornNote from '../torn/TornNote.svelte'
  import type { IntroMode } from '../lib/motion/intro'
  import { motion } from '../lib/motion/level.svelte'
  import type { Plan } from '../lib/motion/plan'
  import { THEMES } from '../lib/theme/palettes'
  import type { Bus } from '../services/board'
  import { fakeProject } from './fakeProject'
  import { SCENES, start } from './motionScenes'

  const q = new URLSearchParams(location.search)
  const mode = q.get('mode') === 'light' ? 'light' : 'dark'
  const lang = q.get('lang') === 'ko' ? 'ko' : 'en'
  const style = q.get('style') === 'neat' ? 'neat' : 'doodle'
  const scene = q.get('scene')
  const intro = q.get('intro') as IntroMode | null
  const view = q.get('view')
  motion.setting = q.get('motion') ?? ''

  const palette = THEMES.find((t) => t.name === (mode === 'dark' ? 'stickypane-dark' : 'stickypane-light'))
  const first = start()
  let board: BoardData = {
    ...first,
    settings: { ...first.settings, theme: palette?.name ?? '', style, language: { choice: lang, resolved: lang } },
    palette,
  }
  const torn = view === 'torn' ? 'work.md' : ''
  const api = { ...fakeProject(board, mode, lang, { mode: view === 'canvas' ? 'canvas' : 'grid' }, torn), load: async () => board, prefs: async () => ({ hotkey: 'Option+Space', language: lang, mode, motion: motion.setting }) }

  // the fake bridge's events: what Board subscribes to
  const listeners = new Map<string, (e: { data?: unknown }) => void>()
  const bus: Bus = {
    on(name, fn) {
      listeners.set(name, fn)
      return () => listeners.delete(name)
    },
  }

  const plans: Plan[] = []
  ;(window as unknown as { __motionPlans: Plan[] }).__motionPlans = plans
  ;(window as unknown as { __motionPlay: (name: string) => void }).__motionPlay = (name) => play(name)

  function play(name: string) {
    const s = SCENES[name]
    if (!s) return
    const { board: next, change } = s(board)
    board = next
    listeners.get('board.changed:1')?.({ data: change })
  }

  if (scene)
    setTimeout(() => {
      if (scene !== 'all') return play(scene)
      Object.keys(SCENES)
        .filter((n) => n !== 'many')
        .forEach((n, i) => setTimeout(() => play(n), i * 1800))
    }, 700)
</script>

{#if torn}
  <TornNote {api} {bus} note={torn} />
{:else}
  <Board {api} {bus} played={(p) => plans.push(p)} />
{/if}
{#if intro}
  <Intro play={intro} />
{/if}
<nav class="scenes" aria-label="scenes">
  {#each Object.keys(SCENES) as name (name)}
    <button onclick={() => play(name)}>{name}</button>
  {/each}
</nav>

<style>
  .scenes {
    position: fixed;
    right: 8px;
    bottom: 8px;
    z-index: 1000;
    display: flex;
    gap: 4px;
    opacity: 0.5;
    font: 11px var(--font-mono);
  }
  .scenes:hover {
    opacity: 1;
  }
</style>
