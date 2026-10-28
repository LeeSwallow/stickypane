<script lang="ts">
  // A .http note (03 §3.8, 10 §10.4): its requests as the terminal lists
  // them — method, name and where it goes, how many checks — with the
  // environment above. Picking a request unfolds it as written and shows
  // its last response under the list, from the .log of the same name the
  // board loads; ▶ Send asks first (the window's question), then
  // note.send sends that request. Choosing the environment is not here
  // yet: the shape has no action for it (10 §10.3).
  import { useBoard } from '../board/context'
  import type { Note } from '../board/model'
  import { tr } from '../i18n'
  import Button from '../lib/ui/Button.svelte'
  import Code from './Code.svelte'
  import { dataOf, type ApiData, type LogData } from './data'
  import { endLine, logOf, section } from './logs'
  import LogLines from './LogLines.svelte'

  let { note }: { note: Note } = $props()
  const board = useBoard()
  const data = $derived(dataOf<ApiData>(note.data, { source: '' }))
  const requests = $derived(data.requests ?? [])
  let picked = $state(0)
  const at = $derived(Math.min(picked, Math.max(requests.length - 1, 0)))
  const request = $derived(requests[at])
  const log = $derived(board.find(logOf(note.name)))
  const response = $derived(request && log ? section(dataOf<LogData>(log.data, { lines: [] }).lines ?? [], request.title) : null)

  /** the terminal's colours for a method: reads, writes, removes; other protocols in the accent */
  const tone = (m: string) =>
    m === 'WS' || m === 'GRPC' ? 'accent' : ['GET', 'HEAD', 'OPTIONS'].includes(m) ? 'info' : m === 'POST' ? 'good' : m === 'DELETE' ? 'bad' : 'warn'
</script>

{#if !data.requests}
  <Code source={data.source} lang="http" label={note.title} />
{:else if !requests.length}
  <p class="none">{$tr('api.none')}</p>
{:else}
  {#if data.env}<p class="env">{$tr('api.env', { env: data.env })}</p>{/if}
  <ul class="list" aria-label={note.title}>
    {#each requests as r, i (i)}
      <li>
        <button class="req" class:on={i === at} aria-pressed={i === at} onclick={() => (picked = i)}>
          <span class="method {tone(r.method)}">{r.method}</span>
          <span class="title">{r.title}</span>
          {#if r.title !== `${r.method} ${r.where}`}<span class="where">{r.where}</span>{/if}
          {#if r.asserts}<span class="meta" title={$tr('api.checks', { n: r.asserts })}>{r.asserts} ✓</span>{/if}
          {#if r.hooks}<span class="meta" title={$tr('api.hooks')}>⚙</span>{/if}
        </button>
      </li>
    {/each}
  </ul>
  {#if request}
    {#if request.block.length}<pre class="block">{request.block.join('\n')}</pre>{/if}
    <div class="send">
      <Button
        variant="primary"
        size="sm"
        icon="send"
        disabled={!board.manage}
        onclick={() => board.manage?.(note, 'send', { at: at + 1, title: request.title })}>{$tr('send.button')}</Button
      >
    </div>
    <section class="response" aria-label={$tr('api.response')}>
      {#if response}
        {@const end = endLine(response)}
        <h4>{$tr('api.response')}{#if end}<span class="end"> · {end}</span>{/if}</h4>
        <LogLines lines={response} />
      {:else}
        <p class="none">{$tr('api.notSent')}</p>
      {/if}
    </section>
  {/if}
{/if}

<style>
  .env,
  .none {
    margin: 0 0 var(--space-1);
    color: var(--pencil);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .list {
    list-style: none;
    margin: 0;
    padding: 0;
  }
  .req {
    display: flex;
    align-items: baseline;
    gap: var(--space-2);
    width: 100%;
    padding: var(--space-1) var(--space-2);
    border: 0;
    border-radius: var(--radius-sm);
    background: none;
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
    text-align: left;
    cursor: pointer;
    --wails-draggable: no-drag;
  }
  .req:hover {
    background: var(--hover);
  }
  .req.on {
    background: var(--select);
  }
  .req:focus-visible {
    outline: var(--focus-width) solid var(--focus);
  }
  .method {
    flex: none;
    min-width: 5ch;
    font: 700 var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .info {
    color: var(--info);
  }
  .good {
    color: var(--good-text, var(--good));
  }
  .bad {
    color: var(--danger-text, var(--bad));
  }
  .warn {
    color: var(--warn-text, var(--warn));
  }
  .accent {
    color: var(--accent-text, var(--accent));
  }
  .title {
    flex: none;
  }
  .where {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    color: var(--pencil);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .meta {
    margin-left: auto;
    flex: none;
    color: var(--pencil);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .meta + .meta {
    margin-left: 0;
  }
  .block {
    margin: var(--space-1) 0 0 var(--space-4);
    color: var(--pencil);
    white-space: pre-wrap;
    overflow-wrap: anywhere;
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .send {
    margin: var(--space-2) 0;
  }
  .response h4 {
    margin: 0 0 var(--space-1);
    color: var(--pencil);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .end {
    font-weight: 400;
  }
</style>
