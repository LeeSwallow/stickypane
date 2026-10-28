<script lang="ts">
  // PLACEHOLDER until Task A6/A7: shows what the shell answers, as data.
  import { Call, Events } from '@wailsio/runtime'
  import { onMount } from 'svelte'
  import { changedEvent, home, noticesEvent, project, restartedEvent, statusEvent, windowKind } from '../lib/shell'

  const kind: string = 'project'
  let shown: Record<string, unknown> = $state({})

  async function put(key: string, p: Promise<unknown>) {
    try {
      shown[key] = await p
    } catch (e) {
      shown[key] = { error: String(e) }
    }
  }

  async function loadHome() {
    await put('ensure', Call.ByName(home('Ensure')))
    await Promise.all([
      put('stickypane', Call.ByName(home('StickypaneStatus'))),
      put('notices', Call.ByName(home('Notices'))),
      put('sessions', Call.ByName(home('Sessions'))),
    ])
  }

  async function loadProject(): Promise<number> {
    const st = (await Call.ByName(project('Status'))) as { window: number; state: string }
    shown.status = st
    if (st.state === 'ready') await put('board', Call.ByName(project('Load')))
    return st.window
  }

  onMount(() => {
    const offs: (() => void)[] = []
    if (kind === 'home') {
      loadHome()
      offs.push(Events.On(noticesEvent, (e) => (shown.notices = e.data)))
    } else {
      loadProject().then((id) => {
        offs.push(Events.On(statusEvent(id), () => loadProject()))
        offs.push(Events.On(changedEvent(id), () => loadProject()))
        offs.push(Events.On(restartedEvent(id), () => loadProject()))
      })
    }
    return () => offs.forEach((off) => off())
  })
</script>

<main class={kind}>
  {#if kind === 'project'}<div class="grip"></div>{/if}
  <pre>{JSON.stringify(shown, null, 2)}</pre>
</main>

<style>
  main {
    height: 100%;
    overflow: auto;
  }
  .grip {
    height: 28px;
    --wails-draggable: drag;
  }
  pre {
    margin: 0;
    padding: 12px 16px;
    font-family: var(--font-mono, monospace);
    font-size: 12px;
    white-space: pre-wrap;
  }
</style>
