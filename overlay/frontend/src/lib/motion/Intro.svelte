<script lang="ts">
  // The launch intro, "꾸물꾸물" (04 §4.3.1): the icon's three sheets drop
  // in the middle of the window and wriggle, fly to the logo, and the notes
  // wriggle once as they appear. Any key or click ends it at once. What
  // plays when is intro.ts; the page around it, stage.ts.
  import { onMount, tick } from 'svelte'
  import { look } from '../doodle/look.svelte'
  import { logoSheet } from '../doodle/logo'
  import Strokes from '../doodle/Strokes.svelte'
  import { STICKYPANE_LIGHT } from '../theme/fallback'
  import { levelOf } from './keyframes'
  import { motion, watchReducedMotion } from './level.svelte'
  import { dayOf, fadeIn, fly, INTRO, introMode, sheet, type IntroMode } from './intro'
  import { frames, lastDay, Reel, remember, shown, storage } from './stage'

  let {
    prefs,
    play,
    store = storage(),
    onend,
  }: {
    /** the app's settings, for "motion" (overlay.json) */
    prefs?: () => Promise<{ motion?: string }>
    /** play this rather than what the day and the settings say (the dev page) */
    play?: IntroMode
    store?: Storage | null
    onend?: (mode: IntroMode) => void
  } = $props()

  let mode = $state<IntroMode>('none')
  let boil = $state(0)
  let veil: HTMLElement | undefined = $state()
  let group: HTMLElement | undefined = $state()
  let sheets: HTMLElement[] = $state([])
  const reel = new Reel()
  const timers: ReturnType<typeof setTimeout>[] = []

  const notes = $derived(look.palette?.notes ?? STICKYPANE_LIGHT.notes)
  const neat = $derived(look.style === 'neat')
  const outline = $derived(look.palette?.text ?? '#2b2b2b')

  function end() {
    reel.skip()
    timers.splice(0).forEach(clearTimeout)
    const was = mode
    mode = 'none'
    if (was !== 'none') onend?.(was)
  }

  // the logo in the title bar, where the sheets go; the window's top left when there is none
  function logoPlace(): { dx: number; dy: number; scale: number } {
    const g = group?.getBoundingClientRect()
    const logo = [...document.querySelectorAll<HTMLElement>('.logo')].find((l) => !l.closest('.intro'))?.getBoundingClientRect()
    if (!g || !g.width) return { dx: 0, dy: 0, scale: 1 }
    const to = logo && logo.width ? logo : { left: 16, top: 12, width: 20, height: 20 }
    return {
      dx: to.left + to.width / 2 - (g.left + g.width / 2),
      dy: to.top + to.height / 2 - (g.top + g.height / 2),
      scale: Math.max(0.1, to.width / g.width),
    }
  }

  async function start(m: IntroMode) {
    mode = m
    if (m === 'none') return
    const app = document.getElementById('app') ?? document.body
    if (m === 'fade') {
      reel.run(app, fadeIn())
    } else if (m === 'short') {
      reel.wriggle(await frames(document, 1500), 0, neat)
    } else {
      await tick()
      sheets.forEach((s, i) => reel.run(s, sheet(i, neat)))
      if (!neat) for (const at of INTRO.boil.slice(1)) timers.push(setTimeout(() => boil++, at))
      const p = logoPlace()
      if (group) reel.run(group, fly(p.dx, p.dy, p.scale))
      if (veil) reel.run(veil, { keyframes: [{ opacity: 1 }, { opacity: 0 }], timing: { delay: INTRO.settle, duration: INTRO.fly, fill: 'both' } })
      timers.push(setTimeout(() => void frames(document, 0).then((f) => reel.wriggle(f, 0, neat)), INTRO.notesAt))
      timers.push(setTimeout(end, INTRO.end))
    }
    await reel.done()
    if (m !== 'full') end()
  }

  onMount(() => {
    const unwatch = watchReducedMotion()
    let first = true
    // a window called back (⌥ Space) wriggles its notes once more, briefly
    const again = () => {
      if (first || document.visibilityState === 'hidden' || mode !== 'none') return
      if (levelOf(motion.setting, motion.systemReduced) === 'full') void start('short')
    }
    document.addEventListener('visibilitychange', again)
    ;(async () => {
      await shown()
      if (play) return void start(play).finally(() => (first = false))
      // the setting, if the app answers quickly; the intro does not wait long for it
      const setting = await Promise.race([
        prefs?.().then((p) => p?.motion ?? '').catch(() => '') ?? Promise.resolve(''),
        new Promise<string>((r) => setTimeout(() => r(''), 300)),
      ])
      if (setting) motion.setting = setting
      const today = dayOf(new Date())
      const m = introMode(levelOf(motion.setting, motion.systemReduced), today, lastDay(store))
      if (m === 'full') remember(store, today)
      first = false
      await start(m)
    })()
    return () => {
      document.removeEventListener('visibilitychange', again)
      unwatch()
      end()
    }
  })

  // any key or click ends it, and does nothing else
  function skip(e: Event) {
    if (mode === 'none') return
    e.stopPropagation()
    if (e.type !== 'keydown') e.preventDefault()
    end()
  }
</script>

<svelte:window onkeydowncapture={skip} onpointerdowncapture={skip} />

{#if mode === 'full'}
  <div class="intro" aria-hidden="true">
    <div class="veil" bind:this={veil}></div>
    <div class="group" bind:this={group}>
      {#each [0, 1, 2] as i (i)}
        <span class="sheet s{i}" bind:this={sheets[i]}>
          <Strokes segs={logoSheet(i, notes, look.style, look.dark, outline, neat ? 0 : boil)} viewBox="0 0 18 18" />
        </span>
      {/each}
    </div>
  </div>
{/if}

<style>
  .intro {
    position: fixed;
    inset: 0;
    z-index: 10000;
    display: grid;
    place-items: center;
    cursor: pointer;
  }
  .veil {
    position: absolute;
    inset: 0;
    background: var(--paper, #fffdf7);
  }
  .group {
    position: relative;
    width: 128px;
    height: 112px;
  }
  .sheet {
    position: absolute;
    width: 72px;
    height: 72px;
  }
  /* the icon's places (logo.ts): blue bottom left, pink, yellow top right */
  .s0 {
    left: 8px;
    top: 36px;
  }
  .s1 {
    left: 28px;
    top: 20px;
  }
  .s2 {
    left: 48px;
    top: 4px;
  }
</style>
