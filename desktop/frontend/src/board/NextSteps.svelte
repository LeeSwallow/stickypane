<script lang="ts">
  // An empty tab says what to do next (03 §3.6): make a note, ask an agent
  // (a sentence to paste), or open the same board in a terminal (the line
  // to type). Three taped notes, upright, since they hold controls.
  import { tr } from '../i18n'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Button from '../lib/ui/Button.svelte'
  import CrayonIcon from '../lib/ui/icons/CrayonIcon.svelte'
  import type { CrayonName } from '../lib/ui/icons/crayon'

  let {
    path,
    oncreate,
    copy,
  }: { path: string; oncreate?: () => void; copy: (text: string) => Promise<void> } = $props()

  // the person's home is ~ (a shell reads it so when it is not quoted)
  const short = $derived(path.replace(/^\/(Users|home)\/[^/]+(?=\/|$)/, '~'))
  const command = $derived(`cd ${quote(short)} && stickypane`)
  let copied = $state('')

  function quote(p: string): string {
    if (/^[\w@%+=:,./~-]+$/.test(p)) return p
    const home = p.startsWith('~/') ? '~/' : ''
    return home + `'${p.slice(home.length).replace(/'/g, `'\\''`)}'`
  }

  async function doCopy(what: string, text: string) {
    try {
      await copy(text)
      copied = what
    } catch {
      copied = ''
    }
  }

  const notes = $derived(look.palette?.notes ?? [])
  const steps = $derived<{ id: string; icon: CrayonName; color: number }[]>([
    { id: 'create', icon: 'plus', color: 0 },
    { id: 'ask', icon: 'board', color: 2 },
    { id: 'terminal', icon: 'terminal', color: 3 },
  ])
</script>

<section class="next" aria-labelledby="next-title">
  <h2 id="next-title" class:neat={look.style === 'neat'}>{$tr('board.next')}</h2>
  <div class="cards">
    {#each steps as s (s.id)}
      <NoteFrame color={notes[s.color] ?? 'currentColor'} id="next-{s.id}">
        <div class="card">
          <CrayonIcon name={s.icon} size={28} />
          <h3 class:neat={look.style === 'neat'}>{$tr('board.next.' + s.id)}</h3>
          <p>{$tr('board.next.' + s.id + 'Body')}</p>
          <div class="act">
            {#if s.id === 'create'}
              <Button variant="primary" size="sm" icon="plus" disabled={!oncreate} onclick={() => oncreate?.()}>{$tr('create.new')}</Button>
            {:else}
              {@const text = s.id === 'ask' ? $tr('board.next.askPrompt') : command}
              <code class:prompt={s.id === 'ask'} class:line={s.id !== 'ask'} title={text}>{text}</code>
              <Button size="sm" variant="secondary" icon={copied === s.id ? 'check' : undefined} onclick={() => doCopy(s.id, text)}>
                {copied === s.id ? $tr('board.next.copied') : $tr('board.next.copy')}
              </Button>
            {/if}
          </div>
        </div>
      </NoteFrame>
    {/each}
  </div>
</section>

<style>
  .next {
    width: 100%;
    max-width: 760px;
    margin: 0 auto;
    padding: var(--space-8) var(--space-5);
    box-sizing: border-box;
  }
  h2 {
    margin: 0 0 var(--space-6);
    text-align: center;
    color: var(--ink);
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
    letter-spacing: var(--hand-track, 0);
  }
  h2.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .cards {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
    gap: var(--space-6) var(--space-5);
  }
  .cards > :global(.note) {
    display: flex;
    flex-direction: column;
  }
  .cards > :global(.note) > :global(.body) {
    flex: 1;
    display: flex;
  }
  .card {
    flex: 1;
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
    min-height: 168px;
  }
  .act {
    display: flex;
    flex-direction: column;
    align-items: flex-start;
    gap: var(--space-2);
    width: 100%;
    margin-top: auto;
  }
  h3 {
    margin: 0;
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
    letter-spacing: var(--hand-track, 0);
  }
  h3.neat {
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  p {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  code {
    display: block;
    width: 100%;
    box-sizing: border-box;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    color: var(--ink);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    overflow-wrap: anywhere;
    user-select: all;
  }
  code.line {
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
  }
  code.prompt {
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
