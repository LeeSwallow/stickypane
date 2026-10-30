<script lang="ts">
  // The home with no board yet (08 §8.2): the logo, the name and what it
  // is, then three steps as taped notes, each checking where it stands
  // now: ① open or make a board, ② connect the agents, ③ the terminal
  // command (optional). ② and ③ are the settings' own services.
  import { onMount } from 'svelte'
  import { tr } from '../i18n'
  import Logo from '../lib/doodle/Logo.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import { checkAgents, linked, type AgentsState, type SetupApi } from '../services/install'
  import Install from './Install.svelte'

  let {
    pick,
    copy,
    setup,
    openSettings,
    install = null,
    busy = false,
    canPick = true,
  }: {
    pick: () => void
    copy: (text: string) => Promise<void>
    setup: SetupApi
    /** the settings window on a section: where ② and ③ go on from */
    openSettings: (section: string) => void
    /** stickypane is missing or too old: the line to install it */
    install?: 'install' | 'update' | null
    busy?: boolean
    canPick?: boolean
  } = $props()

  let agents = $state<AgentsState | null>(null)
  let cli = $state<'ok' | 'todo' | 'none' | null>(null)
  let installing = $state(false)

  function readCli() {
    setup.command().then(
      (s) => (cli = !s.bundled ? 'none' : linked(s) ? 'ok' : 'todo'),
      () => (cli = 'todo'),
    )
  }

  onMount(() => {
    checkAgents(setup).then(
      (a) => (agents = a),
      () => (agents = 'todo'),
    )
    readCli()
    // back from the settings window, the checks are read again
    const again = () => {
      checkAgents(setup).then((a) => (agents = a), () => {})
      readCli()
    }
    window.addEventListener('focus', again)
    return () => window.removeEventListener('focus', again)
  })

  async function installCli() {
    installing = true
    try {
      const a = await setup.install('user', false)
      if (a.occupied) openSettings('cli')
      cli = linked(a.state) ? 'ok' : 'todo'
    } catch {
      openSettings('cli')
    } finally {
      installing = false
    }
  }

  const notes = $derived(look.palette?.notes ?? [])
  const neat = $derived(look.style === 'neat')
</script>

<div class="welcome">
  <header class="hello">
    <Logo size={84} />
    <h1 class:neat>stickypane</h1>
    <p class="tagline">{$tr('home.tagline')}</p>
  </header>

  {#if install}<div class="install"><Install kind={install} {copy} /></div>{/if}

  <h2 class:neat>{$tr('home.welcome')}</h2>
  <ol class="steps">
    <li>
      <NoteFrame color={notes[0] ?? 'currentColor'} id="step-1">
        <div class="step">
          <span class="num" aria-hidden="true">1</span>
          <h3 class:neat>{$tr('home.step1')}</h3>
          <p>{$tr('home.step1Body')}</p>
          <div class="foot">
            <Button variant="primary" size="sm" disabled={busy || !canPick} onclick={pick}>
              {busy ? $tr('home.initRunning') : $tr('home.pickFolder')}
            </Button>
          </div>
        </div>
      </NoteFrame>
    </li>
    <li>
      <NoteFrame color={notes[2] ?? 'currentColor'} id="step-2">
        <div class="step">
          <span class="num" aria-hidden="true">2</span>
          <h3 class:neat>{$tr('home.step2')}</h3>
          <p>{$tr('home.step2Body')}</p>
          <div class="foot">
            {#if agents === 'connected'}
              <Badge tone="good">{$tr('agents.connected')}</Badge>
            {:else}
              <Badge tone={agents === 'none' ? 'warn' : 'neutral'}>{$tr(agents === null ? 'status.checking' : agents === 'none' ? 'status.none' : 'status.todo')}</Badge>
              <Button size="sm" variant="secondary" onclick={() => openSettings('cli')}>{$tr('home.connect')}</Button>
            {/if}
          </div>
        </div>
      </NoteFrame>
    </li>
    <li>
      <NoteFrame color={notes[3] ?? 'currentColor'} id="step-3">
        <div class="step">
          <span class="num" aria-hidden="true">3</span>
          <h3 class:neat>{$tr('home.step3')} <span class="opt">{$tr('home.optional')}</span></h3>
          <p>{$tr('home.step3Body')}</p>
          <div class="foot">
            {#if cli === 'ok'}
              <Badge tone="good">{$tr('status.ok')}</Badge>
            {:else if cli === 'none'}
              <Badge tone="neutral">{$tr('status.none')}</Badge>
            {:else}
              <Badge tone="neutral">{$tr(cli === null ? 'status.checking' : 'status.todo')}</Badge>
              <Button size="sm" variant="secondary" loading={installing} disabled={cli === null} onclick={installCli}>{$tr('home.installCli')}</Button>
            {/if}
          </div>
        </div>
      </NoteFrame>
    </li>
  </ol>
</div>

<style>
  .welcome {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: var(--space-4);
    padding-top: var(--space-6);
  }
  .hello {
    display: grid;
    justify-items: center;
    gap: var(--space-1);
    text-align: center;
  }
  h1 {
    margin: var(--space-2) 0 0;
    font: var(--weight-hand-display) 40px / 44px var(--font-hand);
  }
  h1.neat {
    font: 650 28px / 34px var(--font-body);
  }
  .tagline {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-lg) var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .install {
    width: min(100%, 520px);
  }
  h2 {
    margin: var(--space-6) 0 var(--space-2);
    color: var(--ink-2);
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
  }
  h2.neat {
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .steps {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(190px, 1fr));
    gap: var(--space-6) var(--space-5);
    width: 100%;
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .step {
    display: flex;
    flex-direction: column;
    gap: var(--space-2);
    min-height: 172px;
  }
  .num {
    display: grid;
    place-items: center;
    width: 28px;
    height: 28px;
    border: 2px solid var(--ink);
    border-radius: 50%;
    font: 700 16px / 1 var(--font-hand);
  }
  h3 {
    margin: 0;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  h3.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .opt {
    color: var(--ink-3);
    font: var(--weight-body-sm) var(--text-body-sm) / 1 var(--font-body);
  }
  p {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .foot {
    display: flex;
    flex-wrap: wrap;
    align-items: center;
    gap: var(--space-2);
    margin-top: auto;
  }
</style>
