<script lang="ts">
  // An html note (03 §3.8, 02 §2.7, ADR-0006): the page drawn in a
  // sandboxed frame inside a hand-drawn border, under an "HTML · view
  // only" badge. The frame has an opaque origin and a policy that loads
  // nothing (html.ts); it tells the window only its height, which the
  // window takes from that frame alone. Nothing on the page changes the
  // note: an answer is a form's job.
  import type { Note } from '../board/model'
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import RoughBox from '../lib/doodle/RoughBox.svelte'
  import { seedOf } from '../lib/theme/tokens'
  import { dataOf, type HtmlData } from './data'
  import { heightFrom, SANDBOX, srcdoc } from './html'

  let { note }: { note: Note } = $props()
  const html = $derived(dataOf<HtmlData>(note.data, { html: '' }).html ?? '')
  let frame: HTMLIFrameElement | undefined = $state()
  let height = $state(80)

  // the note's ink and the window's scheme, so the page reads on the note in either mode
  const doc = $derived(srcdoc(html, { dark: look.dark, ink: look.palette?.text }))

  $effect(() => {
    const take = (e: MessageEvent) => {
      const h = heightFrom(e, frame?.contentWindow ?? null)
      if (h !== null) height = h
    }
    window.addEventListener('message', take)
    return () => window.removeEventListener('message', take)
  })
</script>

<div class="html">
  <span class="badge">{$tr('html.badge')}</span>
  {#if html.trim()}
    <div class="sheet">
      <RoughBox stroke="var(--line-strong)" seed={seedOf(note.name)} />
      <!-- a new frame for a new page: the window lets no frame navigate (frame-src none), a changed srcdoc included -->
      {#key doc}
      <iframe
        bind:this={frame}
        title={note.title}
        sandbox={SANDBOX}
        srcdoc={doc}
        referrerpolicy="no-referrer"
        style:height="{height}px"
      ></iframe>
      {/key}
    </div>
  {:else}
    <p class="empty">{$tr('html.empty')}</p>
  {/if}
</div>

<style>
  .badge {
    display: inline-block;
    margin-bottom: var(--space-2);
    padding: 1px var(--space-2);
    border: var(--stroke-hairline) solid var(--line-strong);
    border-radius: var(--radius-pill, 999px);
    color: var(--pencil);
    font: 600 var(--text-body-xs) / var(--leading-body-xs) var(--font-body);
  }
  .sheet {
    position: relative;
    padding: var(--space-2);
  }
  iframe {
    display: block;
    width: 100%;
    border: 0;
    background: transparent;
  }
  .empty {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
