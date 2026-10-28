<script lang="ts">
  // A script (03 §3.8, 10 §10.4): what it runs, as a code block, and a
  // hand-drawn "▶ Run" button. Run asks first (the window's question, as
  // the terminal asks), then note.run runs it in the project folder; its
  // output is the .log of the same name, which the board loads, so the
  // last lines show here as they come in, with how the run ended.
  import { useBoard } from '../board/context'
  import type { Note } from '../board/model'
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import Code from './Code.svelte'
  import { dataOf, type LogData, type ScriptData } from './data'
  import { follow } from './follow'
  import { endLine, logOf, tail } from './logs'
  import LogLines from './LogLines.svelte'

  /** how many of the output's last lines show under the script */
  const TAIL = 12

  let { note }: { note: Note } = $props()
  const board = useBoard()
  const data = $derived(dataOf<ScriptData>(note.data, { source: '', lang: 'sh' }))
  const log = $derived(board.find(logOf(note.name)))
  const out = $derived(log ? (dataOf<LogData>(log.data, { lines: [] }).lines ?? []) : [])
  const end = $derived(endLine(out))
</script>

<div class="bar">
  <Button variant="primary" size="sm" icon="run" disabled={!board.manage} onclick={() => board.manage?.(note, 'run')}>{$tr('run.button')}</Button>
  <span class="lang">{data.lang === 'ps1' ? 'PowerShell' : 'sh'}</span>
</div>
<Code source={data.source} lang={data.lang === 'ps1' ? 'powershell' : 'shellscript'} label={note.title} />
<section class="out" aria-label={$tr('run.log')}>
  <h4>{$tr('run.log')}{#if end}<span class="end"> · {end}</span>{/if}</h4>
  {#if out.length}
    <div use:follow={out.length}><LogLines lines={tail(out, TAIL)} /></div>
  {:else}
    <p class="none">{$tr('run.notYet')}</p>
  {/if}
</section>

<style>
  .bar {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    margin-bottom: var(--space-2);
  }
  .lang {
    color: var(--pencil);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
  }
  .out {
    margin-top: var(--space-2);
  }
  h4 {
    margin: 0 0 var(--space-1);
    color: var(--pencil);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .end {
    font-weight: 400;
  }
  .none {
    margin: 0;
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
