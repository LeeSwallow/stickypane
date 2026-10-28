<script lang="ts">
  // 터미널 명령 설치 and 에이전트 연결 (09 §9.4): the app's own stickypane
  // linked onto the PATH, an older copy that comes first named with its
  // fix, and the agents taught through the app's own stickypane setup —
  // its plan shown before it runs. In a project window, init can follow.
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Confirm from '../lib/ui/Confirm.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import { errorText } from '../services/errors'
  import { linked, type CommandState, type InstallAnswer, type RunResult, type SetupApi, type Target } from '../services/install'

  let { setup, open = true, project = false }: { setup: SetupApi; open?: boolean; project?: boolean } = $props()

  let cmd = $state<CommandState | null>(null)
  let busy = $state('')
  let cliError = $state('')
  let cliNote = $state('')
  let occupied = $state<{ path: string; target: Target } | null>(null)

  let mcp = $state(true)
  let plan = $state<RunResult | null>(null)
  let ran = $state<RunResult | null>(null)
  let agentsError = $state('')
  let inited = $state<RunResult | null>(null)
  let initError = $state('')

  $effect(() => {
    if (!open) return
    cliError = cliNote = agentsError = initError = ''
    plan = ran = inited = null
    setup.command().then(
      (s) => (cmd = s),
      (e) => (cliError = $tr('cli.failed', { error: errorText(e) })),
    )
  })

  const link = $derived(cmd ? linked(cmd) : undefined)
  const userLinked = $derived(link?.target === 'user')

  function took(a: InstallAnswer, target: Target) {
    cmd = a.state
    if (a.occupied) occupied = { path: a.occupied, target }
    if (a.cancelled) cliNote = $tr('cli.cancelled')
  }

  async function install(target: Target, replace = false) {
    occupied = null
    cliError = cliNote = ''
    busy = 'install-' + target
    try {
      took(await setup.install(target, replace), target)
    } catch (e) {
      cliError = $tr('cli.failed', { error: errorText(e) })
    } finally {
      busy = ''
    }
  }

  async function uninstall() {
    cliError = cliNote = ''
    busy = 'uninstall'
    try {
      took(await setup.uninstall(), 'user')
    } catch (e) {
      cliError = $tr('cli.failed', { error: errorText(e) })
    } finally {
      busy = ''
    }
  }

  async function showPlan() {
    agentsError = ''
    plan = ran = inited = null
    busy = 'plan'
    try {
      plan = await setup.connectPlan(mcp)
    } catch (e) {
      agentsError = $tr('agents.failed', { error: errorText(e) })
    } finally {
      busy = ''
    }
  }

  async function connect() {
    agentsError = ''
    busy = 'connect'
    try {
      ran = await setup.connect(mcp)
      plan = null
    } catch (e) {
      agentsError = $tr('agents.failed', { error: errorText(e) })
    } finally {
      busy = ''
    }
  }

  async function initHere() {
    initError = ''
    busy = 'init'
    try {
      inited = await setup.initHere()
    } catch (e) {
      initError = $tr('agents.failed', { error: errorText(e) })
    } finally {
      busy = ''
    }
  }
</script>

<div class="part" aria-labelledby="t-cli">
  <h4 id="t-cli" class:neat={look.style === 'neat'}>{$tr('cli.title')}</h4>
  {#if cmd && !cmd.bundled}
    <p class="line">{$tr('cli.none')}</p>
  {:else if cmd}
    {#if link}
      <p class="line">{$tr('cli.linked', { path: link.path, version: cmd.version })}</p>
      <div class="buttons">
        <Button size="sm" variant="secondary" loading={busy === 'uninstall'} disabled={!!busy} onclick={uninstall}>{$tr('cli.uninstall')}</Button>
      </div>
    {:else}
      <p class="line">{$tr('cli.notLinked')}</p>
      <div class="buttons">
        <Button size="sm" variant="secondary" loading={busy === 'install-user'} disabled={!!busy} onclick={() => install('user')}>{$tr('cli.install')}</Button>
        <Button size="sm" variant="ghost" loading={busy === 'install-system'} disabled={!!busy} onclick={() => install('system')}>{$tr('cli.installSystem')}</Button>
      </div>
    {/if}
    {#if userLinked && !cmd.onPath}
      <p class="warn" role="note">{$tr('cli.notOnPath', { dir: cmd.userDir, file: cmd.rcFile })}</p>
      <code class="cmd">{cmd.rcLine}</code>
    {/if}
    {#if cmd.shadow && cmd.first}
      <p class="warn" role="note">
        {$tr('cli.shadow', { path: cmd.first.path, version: cmd.first.version || $tr('cli.shadowUnknown') })}
        {$tr('cli.fix.' + cmd.shadow.kind)}
      </p>
      <code class="cmd">{cmd.shadow.command}</code>
      {#if cmd.shadow.before && cmd.shadow.after && cmd.shadow.before !== cmd.shadow.after}
        <p class="hint">{$tr('cli.fix.order', { before: cmd.shadow.before, after: cmd.shadow.after })}</p>
      {/if}
    {/if}
  {/if}
  {#if cliNote}<p class="hint" role="status">{cliNote}</p>{/if}
  {#if cliError}<p class="err" role="alert">{cliError}</p>{/if}
</div>

<div class="part" aria-labelledby="t-agents">
  <h4 id="t-agents" class:neat={look.style === 'neat'}>{$tr('agents.title')}</h4>
  <p class="line">{$tr('agents.body')}</p>
  <div class="buttons">
    <Toggle label={$tr('agents.mcp')} bind:checked={mcp} disabled={!!busy} />
    <Button size="sm" variant="secondary" loading={busy === 'plan'} disabled={!!busy || !!plan} onclick={showPlan}>{$tr('settings.connectAgents')}</Button>
  </div>
  {#if plan}
    <p class="hint">{$tr('agents.plan')}</p>
    <pre class="out" aria-label={plan.command}>{plan.output}</pre>
    <div class="buttons">
      <Button size="sm" variant="ghost" disabled={!!busy} onclick={() => (plan = null)}>{$tr('ui.cancel')}</Button>
      <Button size="sm" variant="primary" loading={busy === 'connect'} disabled={!!busy} onclick={connect}>{$tr('agents.run')}</Button>
    </div>
  {/if}
  {#if ran}
    <p class="hint" role="status">{$tr('agents.done')}</p>
    <pre class="out" aria-label={ran.command}>{ran.output}</pre>
    {#if project}
      <p class="line">{$tr('agents.initBody')}</p>
      <div class="buttons">
        <Button size="sm" variant="secondary" loading={busy === 'init'} disabled={!!busy || !!inited} onclick={initHere}>{$tr('agents.init')}</Button>
      </div>
      {#if inited}
        <p class="hint" role="status">{$tr('agents.initDone')}</p>
        {#if inited.output}<pre class="out">{inited.output}</pre>{/if}
      {/if}
      {#if initError}<p class="err" role="alert">{initError}</p>{/if}
    {/if}
  {/if}
  {#if agentsError}<p class="err" role="alert">{agentsError}</p>{/if}
</div>

<Confirm
  open={occupied !== null}
  title={$tr('cli.occupied', { path: occupied?.path ?? '' })}
  message={$tr('cli.occupiedBody')}
  confirmLabel={$tr('cli.replace')}
  danger
  onconfirm={() => occupied && install(occupied.target, true)}
  oncancel={() => (occupied = null)}
/>

<style>
  .part {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
  }
  h4 {
    margin: 0;
    color: var(--ink-2);
    font: 600 var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .line,
  .hint,
  .warn,
  .err {
    margin: 0;
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
    overflow-wrap: anywhere;
  }
  .line {
    color: var(--ink-2);
  }
  .hint {
    color: var(--pencil);
  }
  .warn {
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
  .cmd,
  .out {
    margin: 0;
    padding: var(--space-1) var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    color: var(--ink);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
    user-select: all;
    overflow-wrap: anywhere;
  }
  .out {
    max-height: 10rem;
    overflow: auto;
    white-space: pre-wrap;
  }
</style>
