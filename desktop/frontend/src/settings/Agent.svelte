<script lang="ts">
  // One agent, Claude Code or Codex, on its own (09 §9.4): where it stands,
  // a scope, its skills (the board plugin) connected, updated or removed,
  // and its MCP server registered or not. Every action names this agent
  // alone, shows its plan (setup --dry-run) and runs on Run.
  import { tr } from '../i18n'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import { report } from '../services/errors'
  import type { AgentRequest, AgentStatus, RunResult, Scope, SetupApi } from '../services/install'
  import Row from './Row.svelte'

  let {
    setup,
    agent,
    want,
    project = '',
    ondone,
  }: {
    setup: SetupApi
    agent: AgentStatus
    /** the plugin version the app's stickypane carries */
    want: string
    /** the board folder the settings are for: its scopes are offered; "" for none */
    project?: string
    /** a run finished: read the status again */
    ondone?: (r: RunResult) => void
  } = $props()

  const name = $derived($tr('agent.' + agent.agent))
  // the scopes on offer: a project's only with a project to put them in
  const scopes = $derived<Scope[]>(project ? agent.scopes : agent.scopes.filter((s) => s === 'user'))
  let scope = $state<Scope>('user')
  const plugin = $derived(agent.plugin[scope])
  const mcp = $derived(agent.mcp[scope])

  let busy = $state('')
  let plan = $state<{ req: AgentRequest; result: RunResult } | null>(null)
  let ran = $state<RunResult | null>(null)
  let error = $state('')
  // the switch shows what is registered, and what is asked while a plan waits
  let mcpOn = $state(false)
  $effect(() => {
    mcpOn = plan?.req.part === 'mcp' ? !plan.req.undo : !!mcp
  })

  const tone = $derived(
    agent.state === 'ok' ? 'good' : agent.state === 'broken' ? 'bad' : agent.state === 'outdated' ? 'warn' : 'neutral',
  )
  const version = (v?: string) => (v ? 'v' + v : '')
  const shown = $derived(
    agent.state === 'missing'
      ? $tr('agent.missing')
      : agent.state === 'off'
        ? $tr('agents.notConnected')
        : agent.state === 'ok'
          ? $tr('agent.ok', { version: version(Object.values(agent.plugin).find((p) => p?.version)?.version) }).trim()
          : $tr('agent.' + agent.state),
  )
  // "사용자: v0.2.1 · 이 프로젝트: 없음"
  const summary = $derived(
    scopes.map((s) => `${$tr('scope.' + s)}: ${agent.plugin[s] ? version(agent.plugin[s]!.version) || '✓' : $tr('agent.none')}`).join(' · '),
  )
  const gone = $derived(agent.marketplace?.gone ? agent.marketplace.where : '')
  const pluginAction = $derived(
    !plugin ? 'connect' : gone || plugin.state === 'broken' ? 'repair' : plugin.state === 'outdated' ? 'update' : '',
  )

  async function ask(req: AgentRequest) {
    error = ''
    ran = null
    busy = 'plan-' + req.part
    try {
      plan = { req, result: await setup.agentPlan(req) }
    } catch (e) {
      error = report('terminal.agentPlan', e, (err) => $tr('agents.failed', { error: err }))
    } finally {
      busy = ''
    }
  }

  async function run() {
    if (!plan) return
    error = ''
    busy = 'run'
    try {
      ran = await setup.agentRun(plan.req)
      plan = null
      ondone?.(ran)
    } catch (e) {
      error = report('terminal.agentRun', e, (err) => $tr('agents.failed', { error: err }))
    } finally {
      busy = ''
    }
  }

  const req = (part: 'plugin' | 'mcp', undo: boolean): AgentRequest => ({ agent: agent.agent, part, scope, undo })
</script>

<section class="agent" aria-label={name}>
  <Row label={name} hint={agent.found ? (agent.path ?? '') : $tr('agent.missingHint', { cmd: agent.agent })}>
    {#snippet status()}<Badge {tone}>{shown}</Badge>{/snippet}
  </Row>
  {#if agent.found}
    <div class="body">
      {#if scopes.length > 1}
        <div class="scope">
          <SegmentedControl
            size="sm"
            label={$tr('agent.scope')}
            options={scopes.map((s) => ({ value: s, label: $tr('scope.' + s) }))}
            bind:value={scope}
            onchange={() => (plan = ran = null)}
          />
          <p class="hint">{$tr('scope.' + scope + 'Hint')}</p>
        </div>
      {:else if project && agent.agent === 'codex'}
        <p class="hint">{$tr('agent.codexUserOnly')}</p>
      {/if}
      <p class="line mono">{summary}</p>
      {#if gone}<p class="warn" role="note">{$tr('agent.gone', { where: gone })}</p>{/if}
      {#if plugin?.error && !gone}<p class="err">{plugin.error}</p>{/if}
      {#if plugin?.state === 'outdated'}<p class="hint">{$tr('agent.newer', { version: version(want) })}</p>{/if}

      <div class="part">
        <span class="name">{$tr('agent.skills')}</span>
        <div class="buttons">
          {#if pluginAction}
            <Button size="sm" variant="primary" loading={busy === 'plan-plugin'} disabled={!!busy || !!plan} onclick={() => ask(req('plugin', false))}>{$tr('agent.' + pluginAction)}</Button>
          {/if}
          {#if plugin}
            <Button size="sm" variant="ghost" disabled={!!busy || !!plan} onclick={() => ask(req('plugin', true))}>{$tr('agent.remove')}</Button>
          {/if}
        </div>
      </div>
      <div class="part">
        <span class="name">{$tr('agent.mcp')}</span>
        <Toggle label={$tr('agent.mcpOn')} bind:checked={mcpOn} disabled={!!busy || !!plan} onchange={(on) => ask(req('mcp', !on))} />
      </div>
      {#if mcp}<p class="line mono">{$tr('agent.mcpAt', { command: mcp.command })}</p>{/if}

      {#if plan}
        <p class="hint">{$tr('agents.plan')}</p>
        <pre class="out" aria-label={plan.result.command}>{plan.result.output}</pre>
        <div class="buttons">
          <Button size="sm" variant="ghost" disabled={!!busy} onclick={() => (plan = null)}>{$tr('ui.cancel')}</Button>
          <Button size="sm" variant="primary" loading={busy === 'run'} disabled={!!busy} onclick={run}>{$tr('agents.run')}</Button>
        </div>
      {/if}
      {#if ran}
        <p class="hint" role="status">{$tr('agents.initDone')}</p>
        <pre class="out" aria-label={ran.command}>{ran.output}</pre>
      {/if}
      {#if error}<p class="err" role="alert">{error}</p>{/if}
    </div>
  {/if}
</section>

<style>
  .agent {
    display: flex;
    flex-direction: column;
  }
  .body {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    margin-top: calc(-1 * var(--space-2));
    padding-bottom: var(--space-3);
  }
  .scope {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .part {
    display: flex;
    align-items: center;
    justify-content: space-between;
    gap: var(--space-2);
    min-height: 28px;
  }
  .name {
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    color: var(--ink);
  }
  .line,
  .hint,
  .warn,
  .err {
    margin: 0;
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    overflow-wrap: anywhere;
  }
  .line,
  .hint {
    color: var(--ink-2);
  }
  .mono {
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .warn {
    padding: var(--space-2) var(--space-3);
    border-left: 3px solid var(--warn);
    border-radius: var(--radius-sm);
    background: var(--warn-wash);
    color: var(--ink);
  }
  .err {
    color: var(--danger-text);
  }
  .buttons {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
  }
  .out {
    margin: 0;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    color: var(--ink);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    user-select: all;
    overflow-wrap: anywhere;
    max-height: 10rem;
    overflow: auto;
    white-space: pre-wrap;
  }
</style>
