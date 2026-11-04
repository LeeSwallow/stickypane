<script lang="ts">
  // 원격 호스트 (docs/desktop/02 §2.10): the computers whose boards open
  // over the person's own ssh. Each host: its name, ssh target and
  // command, 연결 확인 (its stickypane's version, or ssh's words), its
  // boards' folders there with 열기 and 빼기, and a field to add one. Below,
  // a form to add a host, by hand or from the ssh config's aliases. The app
  // keeps the list only; ssh signs in.
  import { onMount } from 'svelte'
  import { tr } from '../i18n'
  import { look } from '../lib/doodle/look.svelte'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Confirm from '../lib/ui/Confirm.svelte'
  import CrayonIcon from '../lib/ui/icons/CrayonIcon.svelte'
  import Input from '../lib/ui/Input.svelte'
  import { report } from '../services/errors'
  import { shortPath } from '../lib/path'
  import type { Host, HostCheck, HostsApi, SshAlias } from '../services/hosts'

  let { api }: { api: HostsApi } = $props()

  let hosts = $state<Host[]>([])
  let loaded = $state(false)
  let failed = $state('')
  let checks = $state<Record<string, HostCheck | 'busy'>>({})
  let paths = $state<Record<string, string>>({})
  let removing = $state('')
  // the form: a new host (editing "") or the host named editing
  let form = $state<{ open: boolean; editing: string; name: string; target: string; command: string }>({ open: false, editing: '', name: '', target: '', command: '' })
  let aliases = $state<SshAlias[] | null>(null)
  let reading = $state(false)

  const neat = $derived(look.style === 'neat')
  // a long folder keeps its end, where its name is (wider here than on a
  // home card); the whole is its tooltip
  const short = (p: string) => shortPath(p, 48)

  async function load() {
    try {
      hosts = (await api.hosts()) ?? []
    } catch (e) {
      failed = report('hosts.load', e)
    } finally {
      loaded = true
    }
  }
  onMount(() => void load())

  async function run(f: () => Promise<unknown>) {
    failed = ''
    try {
      await f()
    } catch (e) {
      failed = report('hosts.run', e, (error) => $tr('settings.failed', { error }))
    }
    await load()
  }

  async function check(name: string) {
    checks = { ...checks, [name]: 'busy' }
    let c: HostCheck
    try {
      c = await api.checkHost(name)
    } catch (e) {
      c = { ok: false, bridge: false, error: report('hosts.check', e) }
    }
    checks = { ...checks, [name]: c }
  }

  function edit(h?: Host) {
    form = h
      ? { open: true, editing: h.name, name: h.name, target: h.target, command: h.command ?? '' }
      : { open: true, editing: '', name: '', target: '', command: '' }
  }

  async function save(e: SubmitEvent) {
    e.preventDefault()
    const old = form.editing
    const was = hosts.find((h) => h.name === old)
    const h: Host = { name: form.name.trim(), target: form.target.trim(), command: form.command.trim(), boards: was?.boards ?? null }
    failed = ''
    try {
      await api.saveHost(h, old)
      form = { ...form, open: false }
    } catch (err) {
      failed = report('hosts.save', err, (error) => $tr('settings.failed', { error }))
    }
    await load()
  }

  async function addBoard(e: SubmitEvent, name: string) {
    e.preventDefault()
    const p = (paths[name] ?? '').trim()
    if (!p) return
    await run(() => api.addBoard(name, p))
    if (!failed) paths = { ...paths, [name]: '' }
  }

  async function readAliases() {
    reading = true
    try {
      aliases = (await api.aliases()) ?? []
    } catch (e) {
      failed = report('hosts.readAliases', e)
      aliases = []
    } finally {
      reading = false
    }
  }

  async function take(a: SshAlias) {
    await run(() => api.saveHost({ name: a.name.replace(/[^A-Za-z0-9._-]/g, '-'), target: a.name }, ''))
    aliases = (aliases ?? []).filter((x) => x.name !== a.name)
  }
</script>

{#if loaded && !hosts.length && !form.open}
  <div class="none">
    <CrayonIcon name="remote" size={40} />
    <p class="none-title" class:neat>{$tr('settings.hosts.none')}</p>
    <p class="quiet">{$tr('settings.hosts.noneHint')}</p>
  </div>
{/if}

{#each hosts as h (h.name)}
  {@const c = checks[h.name]}
  <section class="host" aria-label={h.name}>
    <div class="top">
      <div class="who">
        <span class="line">
          <span class="name">{h.name}</span>
          <code class="target" title={h.target}>{h.target}</code>
        </span>
        {#if h.command && h.command !== 'stickypane'}<span class="cmd" title={h.command}>{h.command}</span>{/if}
      </div>
      <div class="acts">
        <Button size="sm" variant="secondary" loading={c === 'busy'} onclick={() => check(h.name)}>{$tr('settings.hosts.check')}</Button>
        <Button size="sm" variant="ghost" onclick={() => edit(h)}>{$tr('settings.hosts.edit')}</Button>
        <Button size="sm" variant="ghost" icon="trash" onclick={() => (removing = h.name)}>{$tr('settings.hosts.remove')}</Button>
      </div>
    </div>
    {#if c && c !== 'busy'}
      <p class="check" role="status">
        {#if c.ok && c.bridge}
          <Badge tone="good">{$tr('settings.hosts.ok', { version: c.version || '?' })}</Badge>
        {:else if c.ok}
          <Badge tone="warn">{$tr('settings.hosts.noBridge')}</Badge>
          {#if c.version}<span class="quiet small">stickypane {c.version}</span>{/if}
        {:else}
          <Badge tone="bad">{$tr('settings.hosts.failed')}</Badge>
          <span class="said">{c.error}</span>
        {/if}
      </p>
    {/if}
    <ul class="boards" aria-label={$tr('settings.hosts.boards')}>
      {#each h.boards ?? [] as p (p)}
        <li>
          <code class="path" title={p}>{short(p)}</code>
          <Button size="sm" variant="secondary" onclick={() => run(() => api.openRemote(h.name, p))}>{$tr('settings.hosts.open')}</Button>
          <Button size="sm" variant="ghost" onclick={() => run(() => api.removeBoard(h.name, p))}>{$tr('settings.hosts.drop')}</Button>
        </li>
      {:else}
        <li class="quiet small">{$tr('settings.hosts.noBoards')}</li>
      {/each}
    </ul>
    <form class="add-path" onsubmit={(e) => addBoard(e, h.name)}>
      <Input
        label={$tr('settings.hosts.path')}
        hideLabel
        placeholder={$tr('settings.hosts.path') + ' — /home/me/project'}
        value={paths[h.name] ?? ''}
        oninput={(e: Event) => (paths = { ...paths, [h.name]: (e.currentTarget as HTMLInputElement).value })}
        spellcheck={false}
        autocapitalize="off"
      />
      <Button size="sm" variant="secondary" icon="plus" type="submit" disabled={!(paths[h.name] ?? '').trim()}>{$tr('settings.hosts.addBoard')}</Button>
    </form>
  </section>
{/each}

{#if form.open}
  <form class="form" onsubmit={save} aria-label={form.editing ? $tr('settings.hosts.edit') : $tr('settings.hosts.add')}>
    <div class="fields">
      <Input label={$tr('settings.hosts.name')} hint={$tr('settings.hosts.nameHint')} bind:value={form.name} required spellcheck={false} autocapitalize="off" />
      <Input label={$tr('settings.hosts.target')} hint={$tr('settings.hosts.targetHint')} bind:value={form.target} required spellcheck={false} autocapitalize="off" placeholder="me@server" />
      <Input label={$tr('settings.hosts.command')} hint={$tr('settings.hosts.commandHint')} bind:value={form.command} spellcheck={false} autocapitalize="off" placeholder="stickypane" />
    </div>
    <div class="form-acts">
      <Button size="sm" variant="ghost" onclick={() => (form = { ...form, open: false })}>{$tr('settings.hosts.cancel')}</Button>
      <Button size="sm" variant="primary" type="submit" disabled={!form.name.trim() || !form.target.trim()}>{$tr('settings.hosts.save')}</Button>
    </div>
  </form>
{:else}
  <div class="more">
    <Button size="sm" variant="secondary" icon="plus" onclick={() => edit()}>{$tr('settings.hosts.add')}</Button>
    <Button size="sm" variant="ghost" loading={reading} onclick={readAliases}>{$tr('settings.hosts.import')}</Button>
  </div>
{/if}

{#if aliases}
  <ul class="aliases" aria-label={$tr('settings.hosts.import')}>
    {#each aliases as a (a.name)}
      <li>
        <span class="name">{a.name}</span>
        {#if a.resolved}<code>{a.resolved}</code>{/if}
        <span class="grow"></span>
        <Button size="sm" variant="secondary" icon="plus" onclick={() => take(a)}>{$tr('settings.hosts.importAdd')}</Button>
      </li>
    {:else}
      <li class="quiet small">{$tr('settings.hosts.importNone')}</li>
    {/each}
  </ul>
{/if}

{#if failed}<p class="err" role="alert">{failed}</p>{/if}

{#if removing}
<Confirm
  open
  title={$tr('settings.hosts.remove') + ' — ' + removing}
  confirmLabel={$tr('settings.hosts.remove')}
  cancelLabel={$tr('settings.hosts.cancel')}
  danger
  onconfirm={() => {
    const n = removing
    removing = ''
    void run(() => api.removeHost(n))
  }}
  oncancel={() => (removing = '')}
/>
{/if}

<style>
  .none {
    display: grid;
    justify-items: center;
    gap: var(--space-2);
    padding: var(--space-6) var(--space-4);
    text-align: center;
  }
  .none-title {
    margin: 0;
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
    letter-spacing: var(--hand-track, 0);
  }
  .none-title.neat {
    font: 600 var(--text-body-lg) / var(--leading-body-lg) var(--font-body);
  }
  .quiet {
    margin: 0;
    color: var(--ink-2);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .small {
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .host {
    display: grid;
    grid-template-columns: minmax(0, 1fr);
    gap: var(--space-2);
    padding: var(--space-4) 0;
    border-top: var(--stroke-hairline) solid var(--line);
  }
  .host:first-of-type {
    border-top: 0;
  }
  .top {
    display: flex;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: var(--space-2) var(--space-3);
  }
  .who {
    display: grid;
    flex: 1;
    gap: 2px;
    min-width: 0;
  }
  .line {
    display: flex;
    align-items: baseline;
    gap: var(--space-2);
    min-width: 0;
  }
  .target {
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
  }
  .name {
    flex: none;
    white-space: nowrap;
    color: var(--ink);
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .cmd {
    min-width: 0;
    overflow: hidden;
    white-space: nowrap;
    text-overflow: ellipsis;
    color: var(--ink-3);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .acts {
    display: flex;
    flex: none;
    gap: var(--space-1);
  }
  code {
    padding: 1px var(--space-2);
    border-radius: var(--radius-sm);
    background: var(--surface-sunk);
    color: var(--ink-2);
    font: var(--weight-mono-md) var(--text-mono-md) / var(--leading-mono-md) var(--font-mono);
  }
  .check {
    display: flex;
    align-items: baseline;
    flex-wrap: wrap;
    gap: var(--space-2);
    margin: 0;
  }
  .said {
    flex-basis: 100%;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    overflow-wrap: anywhere;
  }
  .boards,
  .aliases {
    display: grid;
    gap: var(--space-1);
    margin: 0;
    padding: 0;
    list-style: none;
  }
  .boards li,
  .aliases li {
    display: flex;
    min-width: 0;
    align-items: center;
    gap: var(--space-2);
    min-height: var(--control-sm);
  }
  .boards li > :global(.btn),
  .aliases li > :global(.btn) {
    flex: none;
  }
  .path {
    flex: 1;
    min-width: 0;
    overflow: hidden;
    padding: 0;
    background: none;
    white-space: nowrap;
    text-overflow: ellipsis;
    color: var(--ink);
  }
  .add-path {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .add-path > :global(.field) {
    flex: 1;
  }
  .add-path :global(.box) {
    height: var(--control-sm);
  }
  .add-path :global(input) {
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
  .form {
    display: grid;
    gap: var(--space-3);
    margin-top: var(--space-2);
    padding: var(--space-4);
    border: var(--stroke-hairline) solid var(--line);
    border-radius: var(--radius-md);
    background: var(--surface);
  }
  .fields {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(150px, 1fr));
    gap: var(--space-3);
  }
  .form-acts,
  .more {
    display: flex;
    justify-content: flex-end;
    gap: var(--space-2);
  }
  .more {
    justify-content: flex-start;
    padding-top: var(--space-3);
    border-top: var(--stroke-hairline) solid var(--line);
  }
  .aliases {
    margin-top: var(--space-3);
  }
  .aliases li {
    padding: 0 var(--space-1);
  }
  .grow {
    flex: 1;
  }
  .err {
    margin: var(--space-3) 0 0;
    color: var(--danger-text);
    font: var(--weight-body-sm) var(--text-body-sm) / var(--leading-body-sm) var(--font-body);
  }
</style>
