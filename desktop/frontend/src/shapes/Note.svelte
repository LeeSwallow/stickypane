<script lang="ts">
  // A Markdown note (03 §3.8): body in Pretendard, headings by hand, code
  // in colour, diagrams and math drawn. [[links]] go to their note; a line
  // that is only ![[name]] draws that note, one level deep (10 §10.2).
  import 'katex/dist/katex.min.css'
  import { useBoard } from '../board/context'
  import { htmlFor, splitEmbeds, type Note as NoteT, type NoteData, type Segment } from '../board/model'
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import { mermaidRenderer, render } from './markdown'
  import Shape from './Shape.svelte'

  let { note, depth = 0 }: { note: NoteT; depth?: number } = $props()
  const board = useBoard()

  const markdown = $derived((note.data as NoteData | null)?.markdown ?? '')
  const segments = $derived(splitEmbeds(markdown))
  // the HTML of the segments it was made for: a render still running for an
  // older text is never drawn against the new segments
  let rendered = $state.raw<{ segs: Segment[]; html: string[] }>({ segs: [], html: [] })
  const html = $derived(htmlFor(rendered, segments))

  $effect(() => {
    const segs = segments
    const style = look.style
    const mermaid = look.palette ? mermaidRenderer(look.palette, style) : undefined
    let live = true
    Promise.all(segs.map((s) => ('md' in s ? render(s.md, { mermaid }) : Promise.resolve(''))))
      .then((out) => live && (rendered = { segs, html: out }))
      .catch(() => {})
    return () => (live = false)
  })

  function click(e: MouseEvent) {
    const a = (e.target as HTMLElement).closest('a')
    if (!a) return
    const link = a.dataset.note
    if (link !== undefined) {
      e.preventDefault()
      board.navigate(link)
    } else if (!a.getAttribute('href')?.startsWith('#')) {
      e.preventDefault() // nothing leaves the board window
    }
  }
</script>

<!-- links are followed by the click on them; keys reach them as links -->
<div class="md" role="presentation" onclick={click}>
  {#if !markdown.trim()}
    <p class="empty">{$tr('note.empty')}</p>
  {/if}
  {#each segments as seg, i (i)}
    {#if 'md' in seg}
      {#if html[i] !== undefined}
        {@html html[i]}
      {:else}
        <pre class="raw">{seg.md}</pre>
      {/if}
    {:else}
      {@const target = board.find(seg.embed)}
      {#if !target}
        <p class="missing">{$tr('note.embedMissing', { name: seg.embed })}</p>
      {:else if depth >= 1}
        <p><a class="wikilink" data-note={seg.embed} href="#{seg.embed}">{target.title}</a></p>
      {:else}
        <figure class="embed">
          <figcaption><a class="wikilink" data-note={seg.embed} href="#{seg.embed}">{target.icon} {target.title}</a></figcaption>
          <Shape note={target} depth={depth + 1} />
        </figure>
      {/if}
    {/if}
  {/each}
</div>

<style>
  .md {
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    color: var(--ink);
    overflow-wrap: anywhere;
  }
  .md :global(h1),
  .md :global(h2),
  .md :global(h3),
  .md :global(h4) {
    font-family: var(--font-hand);
    font-weight: 700;
    line-height: 1.2;
    margin: 0.7em 0 0.3em;
  }
  .md :global(h1) {
    font-size: var(--text-hand-title);
  }
  .md :global(h2) {
    font-size: var(--text-hand-label);
  }
  .md :global(h3) {
    font-size: var(--text-body-lg);
  }
  .md :global(p),
  .md :global(ul),
  .md :global(ol),
  .md :global(table),
  .md :global(blockquote) {
    margin: 0.4em 0;
  }
  .md :global(:first-child) {
    margin-top: 0;
  }
  .md :global(ul),
  .md :global(ol) {
    padding-left: 1.4em;
  }
  .md :global(li.task-list-item) {
    list-style: none;
    margin-left: -1.3em;
  }
  .md :global(a) {
    color: var(--accent);
  }
  .md :global(a.wikilink) {
    text-decoration: underline dashed;
    text-underline-offset: 3px;
    cursor: pointer;
  }
  .md :global(code) {
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    background: var(--select);
    padding: 1px var(--space-1);
    border-radius: var(--radius-sm);
  }
  .md :global(pre) {
    background: var(--select);
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-md);
    overflow: auto;
  }
  .md :global(pre code) {
    background: none;
    padding: 0;
  }
  .md :global(pre.shiki span) {
    color: var(--shiki-light);
  }
  :global([data-mode='dark']) .md :global(pre.shiki span) {
    color: var(--shiki-dark);
  }
  .md :global(table) {
    border-collapse: collapse;
  }
  .md :global(th),
  .md :global(td) {
    border: 1px solid var(--grid-major);
    padding: var(--space-1) var(--space-2);
  }
  .md :global(blockquote) {
    border-left: var(--stroke-emphasis) solid var(--pencil);
    padding-left: var(--space-3);
    color: var(--pencil);
  }
  .md :global(.mermaid svg) {
    max-width: 100%;
    height: auto;
  }
  .md :global(.mermaid-error .why) {
    color: var(--bad);
  }
  .md :global(.katex-display) {
    overflow-x: auto;
    overflow-y: hidden;
  }
  .md :global(.footnotes) {
    font-size: var(--text-body-sm);
    color: var(--pencil);
  }
  .md :global(.sr-only) {
    position: absolute;
    width: 1px;
    height: 1px;
    overflow: hidden;
    clip: rect(0 0 0 0);
  }
  .raw {
    white-space: pre-wrap;
    font: inherit;
    margin: 0;
    background: none;
  }
  .empty,
  .missing {
    color: var(--pencil);
    margin: 0;
  }
  .embed {
    margin: var(--space-2) 0;
    padding: var(--space-2) var(--space-3);
    border: var(--stroke-hairline) dashed var(--line-strong);
    border-radius: var(--radius-md);
  }
  .embed figcaption {
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
    margin-bottom: var(--space-1);
  }
</style>
