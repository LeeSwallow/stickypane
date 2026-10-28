<script lang="ts">
  // DEV ONLY: ?demo=system is the design system's reference page (03 §3.11):
  // a composed board window, the icon tiers beside the app icon, every token,
  // and every component in its states. ?theme= ?mode=light|dark ?style=neat
  // ?lang=en ?density=compact. The headings are for the developer, in English.
  import { onMount } from 'svelte'
  import { setLanguage, tr as tx, type Lang } from '../i18n'
  import Check from '../lib/doodle/Check.svelte'
  import Circle from '../lib/doodle/Circle.svelte'
  import Logo from '../lib/doodle/Logo.svelte'
  import { look } from '../lib/doodle/look.svelte'
  import NoteFrame from '../lib/doodle/NoteFrame.svelte'
  import Paper from '../lib/doodle/Paper.svelte'
  import Sheet from '../lib/doodle/Sheet.svelte'
  import { resolveThemeName, watchSystemDark, type Mode } from '../lib/theme/mode'
  import { DENSITY, ELEVATION, MAX_TILT, MOTION, RADIUS, SPACE, STROKE, TYPE, Z, density, type Density } from '../lib/theme/scale'
  import { agentColor, applyTheme } from '../lib/theme/tokens'
  import Badge from '../lib/ui/Badge.svelte'
  import Button from '../lib/ui/Button.svelte'
  import Chip from '../lib/ui/Chip.svelte'
  import Confirm from '../lib/ui/Confirm.svelte'
  import EmptyState from '../lib/ui/EmptyState.svelte'
  import IconButton from '../lib/ui/IconButton.svelte'
  import CrayonIcon from '../lib/ui/icons/CrayonIcon.svelte'
  import { CRAYON_NAMES } from '../lib/ui/icons/crayon'
  import Icon from '../lib/ui/icons/Icon.svelte'
  import { ICON_NAMES } from '../lib/ui/icons/icons'
  import Input from '../lib/ui/Input.svelte'
  import Menu from '../lib/ui/Menu.svelte'
  import type { MenuItem } from '../lib/ui/menu'
  import NoteHeader from '../lib/ui/NoteHeader.svelte'
  import SegmentedControl from '../lib/ui/SegmentedControl.svelte'
  import Tabs from '../lib/ui/Tabs.svelte'
  import TextArea from '../lib/ui/TextArea.svelte'
  import TitleBar from '../lib/ui/TitleBar.svelte'
  import Toast from '../lib/ui/Toast.svelte'
  import Toggle from '../lib/ui/Toggle.svelte'
  import Tooltip from '../lib/ui/Tooltip.svelte'
  import { windowMenu, type WindowActions } from '../lib/ui/chrome'
  import { THEMES } from '../lib/theme/palettes'
  import { FRACTIONS } from '../lib/sizes'
  import appIcon from '../../../build/appicon.png?url'

  const params = new URLSearchParams(location.search)
  let theme = $state(params.get('theme') ?? 'stickypane-dark')
  let mode = $state<Mode>((params.get('mode') as Mode) ?? 'system')
  let style = $state<'doodle' | 'neat'>(params.get('style') === 'neat' ? 'neat' : 'doodle')
  let dens = $state<Density>(params.get('density') === 'compact' ? 'compact' : 'comfortable')
  let lang = $state<Lang>(params.get('lang') === 'en' ? 'en' : 'ko')
  let systemDark = $state(false)

  onMount(() => watchSystemDark((d) => (systemDark = d)))

  const resolved = $derived(resolveThemeName(theme, mode, systemDark))
  const palette = $derived(THEMES.find((t) => t.name === resolved) ?? THEMES[0])

  $effect(() => {
    look.style = style
    look.palette = palette
    look.dark = palette.dark
    applyTheme(palette, palette.dark)
    document.body.style.background = 'var(--paper)'
  })
  $effect(() => setLanguage(lang))

  const say = (what: string) => () => console.info('[system demo]', what)
  const win: WindowActions = { close: say('close'), minimise: say('minimise'), quit: say('quit') }

  // the composed board
  let tab = $state('work')
  let size = $state<string>('1/2')
  let pinned = $state(true)
  let view = $state('grid')
  const noteMenu = $derived<MenuItem[]>([
    { id: 'edit', label: $tx('note.openInEditor'), icon: 'edit-external', shortcut: 'E' },
    { id: 'zoom', label: $tx('ui.zoom'), icon: 'zoom', shortcut: '↵' },
    { id: 'move', label: $tx('note.move'), icon: 'tab' },
    { id: 'archive', label: $tx('note.archive'), icon: 'archive' },
    { kind: 'separator', id: 's1' },
    { id: 'trash', label: $tx('note.trash'), icon: 'trash', danger: true, shortcut: 'D' },
  ])
  const ctxMenu = $derived<MenuItem[]>([
    { id: 'edit', label: $tx('note.openInEditor'), icon: 'edit-external', shortcut: 'E' },
    { id: 'link', label: 'Copy link', icon: 'link', disabled: true },
    ...windowMenu('mac', $tx),
  ])
  const agents = ['claude', 'codex', 'sam']

  // tokens on the page
  const colorTokens = [
    ['--paper', '--chrome', '--surface', '--surface-sunk', '--line', '--line-strong'],
    ['--ink', '--ink-2', '--ink-3', '--accent-text', '--good-text', '--warn-text', '--danger-text'],
    ['--accent-fill', '--accent-wash', '--danger-fill', '--focus', '--tip', '--card-ink'],
  ]
  const states = ['default', 'hover', 'focus', 'active', 'disabled', 'loading'] as const
  const variants = ['primary', 'secondary', 'ghost', 'danger'] as const
  const ds = (s: string) => (s === 'default' || s === 'disabled' || s === 'loading' ? undefined : s)
  const opt = <T extends string>(vals: T[], key: string) => vals.map((v) => ({ value: v, label: $tx(`${key}.${v}`) }))
</script>

<div class="page" use:density={dens}>
  <Paper>
    <!-- the page's own controls, made of the system -->
    <div class="top">
      <Logo size={28} />
      <h1>Design system</h1>
      <span class="sub">{resolved} · {style} · {dens}</span>
      <span class="grow"></span>
      <SegmentedControl label={$tx('settings.style')} bind:value={style} options={opt(['doodle', 'neat'], 'settings.style')} />
      <SegmentedControl label={$tx('settings.mode')} bind:value={mode} options={opt(['system', 'dark', 'light'], 'settings.mode')} />
      <SegmentedControl label={$tx('settings.density')} bind:value={dens} options={opt(['comfortable', 'compact'], 'settings.density')} />
      <SegmentedControl label={$tx('settings.language')} bind:value={lang} options={[{ value: 'ko', label: '한국어' }, { value: 'en', label: 'English' }]} />
      <select class="theme" bind:value={theme} aria-label={$tx('settings.theme')}>
        {#each THEMES as t (t.name)}<option>{t.name}</option>{/each}
      </select>
    </div>

    <!-- 1. at a glance -->
    <section class="glance">
      <div class="leftcol">
      <div class="window">
        <TitleBar name="demo-project" path="/Users/min/Projects/Personal/stickypane/.claude/worktrees/agent/demo-project" hotkey="⌥Space" chrome="mac" window={win}>
          {#snippet actions()}
            <SegmentedControl label={$tx('ui.view')} bind:value={view} iconOnly size="sm" options={[{ value: 'grid', label: $tx('ui.view.grid'), icon: 'grid' }, { value: 'canvas', label: $tx('ui.view.canvas'), icon: 'canvas' }]} />
            <IconButton icon="search" label={$tx('ui.search')} shortcut="/" />
            <Button variant="primary" size="sm" icon="plus">{$tx('create.make')}</Button>
            <IconButton icon="pin" label={pinned ? $tx('ui.unpin') : $tx('ui.pin')} pressed={pinned} crayon={pinned} onclick={() => (pinned = !pinned)} />
          {/snippet}
        </TitleBar>
        <div class="glance-row">
          {#each agents as a, i (a)}
            <Chip name={a} color={agentColor(a, palette.notes)} detail={['Wire up the login form', 'Refresh tokens', ''][i] || undefined} time={['4m', '28m', ''][i] || undefined} stalled={i === 1} onclick={() => {}} selected={i === 0} />
          {/each}
          <span class="grow"></span>
          <Badge tone="warn">{$tx('badge.stale', { time: '28m' })}</Badge>
        </div>
        <div class="tabs-row">
          <Tabs label="tabs" bind:value={tab} tabs={[{ id: 'work', label: 'work', count: 4 }, { id: 'notes', label: 'notes', count: 2 }, { id: 'log', label: 'log' }, { id: 'ask', label: 'ask', count: 1, attention: true }]} />
        </div>
        <div class="board">
          <div class="note-wrap">
            <NoteFrame color={palette.notes[0]} id="release">
              <NoteHeader title="릴리스 체크 — v0.2 macOS build and the cask" color={palette.notes[0]} summary="1/3" {size} onsize={(s) => (size = s)} {pinned} updated menu={noteMenu} book={{ page: 2, total: 5, onprev: () => {}, onnext: () => {} }} onpin={(v) => (pinned = v)} />
              <ul class="items">
                <li><Check checked seed={3} /><s>write tests</s></li>
                <li><Check seed={4} />update README</li>
                <li><Check seed={5} />tag 0.2.0</li>
              </ul>
            </NoteFrame>
          </div>
          <div class="note-wrap">
            <NoteFrame color={palette.notes[3]} id="sprint">
              <NoteHeader title="Auth sprint" color={palette.notes[3]} summary="3 cards" size="1/2" menu={noteMenu} />
              <div class="cards">
                <div class="col"><span class="colh">Doing <Badge count={2} /></span>
                  <Sheet color={agentColor('claude', palette.notes)} id="c1">login form<small>@claude · 4m</small></Sheet>
                  <span class="ringed"><Sheet color={agentColor('codex', palette.notes)} id="c2">refresh token<small>@codex · 28m</small></Sheet><Circle /></span>
                </div>
                <div class="col"><span class="colh">Done <Badge count={1} /></span>
                  <Sheet color={agentColor('sam', palette.notes)} id="c3">schema<small>@sam · 1h</small></Sheet>
                </div>
              </div>
            </NoteFrame>
          </div>
        </div>
        <div class="toast-slot">
          <Toast message={$tx('note.trashed') + ' · "Auth sprint"'} icon="trash" action={{ label: $tx('note.restore'), run: () => {} }} onclose={() => {}} />
        </div>
      </div>
        <h2>Linux title bar (own buttons)</h2>
        <div class="strip">
          <TitleBar name="a-very-long-project-name-that-would-have-wrapped" path="/home/sam/src/a-very-long-project-name-that-would-have-wrapped" hotkey="Alt+Space" chrome="own" window={win} />
        </div>
        <h2>Hand-drawn, where</h2>
        <div class="rules">
          <div><b>Rough</b> note borders · cards · checks/radios · highlight marks · stalled rings · the active tab's underline · empty-state art · crayon icons</div>
          <div><b>Crisp</b> title bar · toolbars · menus · inputs · tabs · toasts · dialog controls · all text</div>
          <div><b>Tilt</b> ≤ {MAX_TILT}°, cards and stickers only — never anything that holds a control; text never on a jittering line</div>
          <div><b>Neat</b> roughness 0, no tilt, no tape, flat crayon icons, Pretendard titles</div>
        </div>
      </div>

      <div class="side">
        <h2>Confirm (taped note)</h2>
        <Confirm inline danger title={$tx('note.confirmTrash', { title: 'Auth sprint' })} message=".sticky/work/auth-sprint.md" confirmLabel={$tx('note.trash')} onconfirm={() => {}} />
        <h2>Context menu</h2>
        <Menu inline items={ctxMenu} label="context" />
        <h2>Tray hint (shown once)</h2>
        <Toast message={$tx('notice.stayInTray')} icon="tab" action={{ label: $tx('app.quit'), run: () => {} }} onclose={() => {}} />
      </div>
    </section>

    <!-- 2. icons -->
    <section>
      <h2 class="h">Icons</h2>
      <div class="icons">
        <div class="appicon">
          <img src={appIcon} alt="" />
          <Logo size={64} />
          <span class="cap">app icon · drawn logo</span>
        </div>
        <div class="tier">
          <h3>Crayon — icons that stand alone <span class="cap">wash + −41° hatch + heavy outline; flat in neat</span></h3>
          <div class="grid crayon">
            {#each CRAYON_NAMES as n (n)}
              <figure>
                <CrayonIcon name={n} size={48} />
                <span class="small"><CrayonIcon name={n} size={20} /><CrayonIcon name={n} size={16} /></span>
                <figcaption>{n}</figcaption>
              </figure>
            {/each}
          </div>
          <h3>Line — controls and inline <span class="cap">20px · stroke 1.75 · round caps · pen-bent in doodle</span></h3>
          <div class="grid line">
            {#each ICON_NAMES as n (n)}
              <figure>
                <span class="big"><Icon name={n} size={28} /></span>
                <span class="small"><Icon name={n} size={20} /><Icon name={n} size={16} /></span>
                <figcaption>{n}</figcaption>
              </figure>
            {/each}
          </div>
        </div>
      </div>
    </section>

    <!-- 3. components -->
    <section>
      <h2 class="h">Components</h2>
      <div class="panel">
        <h3>Button</h3>
        <table class="matrix">
          <thead><tr><th></th>{#each states as s (s)}<th>{s}</th>{/each}</tr></thead>
          <tbody>
            {#each variants as v (v)}
              <tr>
                <th>{v}</th>
                {#each states as s (s)}
                  <td><Button variant={v} data-state={ds(s)} disabled={s === 'disabled'} loading={s === 'loading'} icon={v === 'danger' ? 'trash' : v === 'primary' ? 'plus' : undefined}>{v === 'danger' ? $tx('note.trash') : v === 'primary' ? $tx('create.make') : $tx('ui.cancel')}</Button></td>
                {/each}
              </tr>
            {/each}
          </tbody>
        </table>
        <div class="row">
          <Button size="sm">sm</Button><Button>md</Button><Button size="lg">lg</Button>
          <Button variant="primary" size="lg" icon="run">{$tx('run.button')}</Button>
          <Button variant="secondary" icon="send">{$tx('send.button')}</Button>
        </div>

        <h3>IconButton <span class="cap">label = name + tooltip</span></h3>
        <div class="row">
          {#each ['default', 'hover', 'focus', 'active'] as s (s)}
            <span class="st"><IconButton icon="tear-off" label={$tx('ui.tearOff')} data-state={ds(s)} /><em>{s}</em></span>
          {/each}
          <span class="st"><IconButton icon="pin" label={$tx('ui.pin')} pressed={false} /><em>off</em></span>
          <span class="st"><IconButton icon="pin" label={$tx('ui.unpin')} pressed crayon /><em>pressed</em></span>
          <span class="st"><IconButton icon="trash" label={$tx('note.trash')} tone="danger" data-state="hover" /><em>danger</em></span>
          <span class="st"><IconButton icon="settings" label={$tx('settings.title')} disabled /><em>disabled</em></span>
          <span class="st"><IconButton icon="close" size="sm" label={$tx('ui.close')} /><em>sm</em></span>
          <span class="st tipdemo"><Tooltip text={$tx('ui.tearOff')} shortcut="T" placement="top" pinned><IconButton icon="tear-off" label={$tx('ui.tearOff')} tooltip={false} /></Tooltip><em>tooltip</em></span>
        </div>

        <div class="cols">
          <div>
            <h3>Input / TextArea</h3>
            <div class="stack">
              <Input label={$tx('create.titlePrompt')} placeholder="Auth sprint" />
              <Input label={$tx('ui.search')} icon="search" value="login" data-state="focus" hideLabel />
              <Input label={$tx('create.titlePrompt')} value="auth/sprint?" error="A title cannot have / or ?" />
              <Input label={$tx('create.titlePrompt')} value="read only" disabled hint="Locked while the agent writes" />
              <TextArea label="Body" rows={2} placeholder="Markdown…" />
            </div>
          </div>
          <div>
            <h3>Toggle · SegmentedControl · Tabs</h3>
            <div class="stack">
              <div class="row"><Toggle label={$tx('ui.pin')} checked /><Toggle label="Reduce motion" /><Toggle label="Off and disabled" disabled /></div>
              <div class="row">
                <SegmentedControl label={$tx('ui.view')} bind:value={view} options={[{ value: 'grid', label: $tx('ui.view.grid'), icon: 'grid' }, { value: 'canvas', label: $tx('ui.view.canvas'), icon: 'canvas' }]} />
                <SegmentedControl label={$tx('ui.size')} bind:value={size} size="sm" options={FRACTIONS.map((f) => ({ value: f, label: f }))} />
              </div>
              <Tabs label="demo tabs" value="notes" tabs={[{ id: 'work', label: 'work' }, { id: 'notes', label: 'notes', count: 12 }, { id: 'archive', label: 'archive' }]} />
            </div>
            <h3>Chip · Badge</h3>
            <div class="row wrap">
              <Chip name="claude" color={agentColor('claude', palette.notes)} detail="login form" time="4m" />
              <Chip name="codex" color={agentColor('codex', palette.notes)} detail="refresh token" time="28m" stalled />
              <Chip name="sam" color={agentColor('sam', palette.notes)} onclick={() => {}} selected />
              <Chip name="gpt-5" color={agentColor('gpt-5', palette.notes)} onclick={() => {}} data-state="hover" />
            </div>
            <div class="row wrap">
              <Badge count={3} /><Badge count={12} tone="accent" /><Badge count={1} tone="warn" />
              <Badge tone="good">{$tx('badge.answer')}</Badge><Badge tone="warn">{$tx('badge.stale', { time: '12m' })}</Badge><Badge tone="bad">error</Badge><Badge tone="accent">HTML</Badge>
            </div>
          </div>
          <div>
            <h3>Toast</h3>
            <div class="stack">
              <Toast message={$tx('home.removed', { n: 2 })} action={{ label: $tx('home.undo'), run: () => {} }} onclose={() => {}} />
              <Toast message={$tx('home.copied')} icon="check" tone="good" />
              <Toast message={$tx('board.cannotWatch')} icon="warn" tone="bad" action={{ label: $tx('board.refresh'), run: () => {} }} />
            </div>
            <h3>Menu</h3>
            <Menu inline items={noteMenu} label="note" />
          </div>
        </div>

        <div class="cols two">
          <div>
            <h3>EmptyState</h3>
            <div class="emptybox">
              <EmptyState title={$tx('board.emptyTitle')} body={$tx('board.emptyHint')}>
                <Button variant="primary" icon="plus">{$tx('create.make')}</Button>
              </EmptyState>
            </div>
          </div>
          <div>
            <h3>NoteHeader <span class="cap">no tilt — it holds controls</span></h3>
            <div class="stack">
              <div class="nh"><NoteHeader title="Plain" color={palette.notes[2]} size="1/4" menu={noteMenu} /></div>
              <div class="nh"><NoteHeader title="A very long note title that is cut with an ellipsis instead of wrapping" color={palette.notes[4]} summary="12/40" size="1" pinned updated menu={noteMenu} /></div>
              <div class="nh"><NoteHeader title="책 · Book" color={palette.notes[5]} summary="p.1" book={{ page: 1, total: 3, onprev: () => {}, onnext: () => {} }} menu={noteMenu} /></div>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- 4. tokens -->
    <section>
      <h2 class="h">Tokens</h2>
      <div class="panel tokens">
        <div>
          <h3>Type</h3>
          {#each Object.entries(TYPE) as [fam, styles] (fam)}
            {#each Object.entries(styles) as [n, s] (n)}
              <div class="spec">
                <span class="tk">{fam}.{n} <em>{s.size}/{s.line}</em></span>
                <span style:font-family="var(--font-{fam})" style:font-size="{s.size}px" style:line-height="{s.line}px" style:font-weight={s.weight}>
                  {fam === 'mono' ? '4m · 3/6 · ⌥Space' : fam === 'hand' ? '릴리스 체크 Release' : '노트는 테이프로 붙은 메모지처럼 보인다'}
                </span>
              </div>
            {/each}
          {/each}
        </div>
        <div>
          <h3>Space (4px grid)</h3>
          {#each SPACE as n, i (i)}
            <div class="spec"><span class="tk">space-{i + 1} <em>{n}</em></span><i class="bar" style:width="{n}px"></i></div>
          {/each}
          <h3>Radius · Stroke</h3>
          <div class="row wrap">
            {#each Object.entries(RADIUS).filter(([k]) => k !== 'pill' && k !== 'none') as [k, r] (k)}
              <span class="rad" style:border-radius="{r}px">{k}<em>{r}</em></span>
            {/each}
          </div>
          <div class="row wrap">
            {#each Object.entries(STROKE) as [k, w] (k)}
              <span class="strk"><i style:height="{w}px"></i>{k} <em>{w}</em></span>
            {/each}
          </div>
          <p class="cap">roughness: note 0.8 · card 1.1 · mark 1.6 · neat 0 · max tilt {MAX_TILT}°</p>
        </div>
        <div>
          <h3>Elevation</h3>
          <div class="row wrap">
            {#each Object.entries(ELEVATION) as [k, v] (k)}
              <span class="elev" style:box-shadow={v}>{k}</span>
            {/each}
          </div>
          <h3>Layers · Motion · Density</h3>
          <p class="cap">{Object.entries(Z).map(([k, v]) => `${k} ${v}`).join(' · ')}</p>
          <p class="cap">short {MOTION.short} · base {MOTION.base} · long {MOTION.long} · carry {MOTION.carry}ms · ease / slap / exit · reduced → opacity {MOTION.reduced}ms</p>
          <p class="cap">controls {DENSITY.comfortable.controlSm}/{DENSITY.comfortable.controlMd}/{DENSITY.comfortable.controlLg} · compact {DENSITY.compact.controlSm}/{DENSITY.compact.controlMd}/{DENSITY.compact.controlLg} · title bar {DENSITY.comfortable.titlebar}/{DENSITY.compact.titlebar} (52 on macOS)</p>
          <h3>Colour</h3>
          {#each colorTokens as row, i (i)}
            <div class="row wrap sw">
              {#each row as v (v)}<span><i style:background="var({v})"></i>{v}</span>{/each}
            </div>
          {/each}
        </div>
      </div>
    </section>

  </Paper>
</div>

<style>
  .page {
    min-height: 100vh;
    color: var(--ink);
    font: var(--weight-body-md) var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  .top {
    display: flex;
    align-items: center;
    gap: var(--space-3);
    padding: var(--space-4) var(--space-8);
    border-bottom: 1px solid var(--line);
    background: var(--chrome);
  }
  h1 {
    margin: 0;
    font: 600 var(--text-body-lg) / 1 var(--font-body);
  }
  .sub,
  .cap {
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
  }
  .grow {
    flex: 1;
  }
  .theme {
    height: var(--control-md);
    border: 1px solid var(--line-strong);
    border-radius: var(--radius-md);
    background: var(--surface-sunk);
    color: var(--ink);
    font: inherit;
    padding: 0 var(--space-2);
  }
  section {
    padding: var(--space-5) var(--space-8) 0;
  }
  .h {
    margin: 0 0 var(--space-3);
    font: var(--weight-hand-title) var(--text-hand-title) / var(--leading-hand-title) var(--font-hand);
  }
  h2 {
    margin: var(--space-4) 0 var(--space-2);
    color: var(--ink-2);
    font: 600 var(--text-body-xs) / var(--leading-body-xs) var(--font-body);
    letter-spacing: 0.06em;
    text-transform: uppercase;
  }
  h3 {
    margin: var(--space-5) 0 var(--space-3);
    font: 600 var(--text-body-md) / var(--leading-body-md) var(--font-body);
  }
  h3:first-child {
    margin-top: 0;
  }
  h3 .cap {
    margin-left: var(--space-2);
    font-weight: 400;
  }

  /* the composed window */
  .glance {
    display: grid;
    grid-template-columns: 840px 1fr;
    gap: var(--space-8);
    align-items: start;
  }
  .window {
    position: relative;
    overflow: hidden;
    border-radius: var(--radius-window);
    background: var(--paper);
    background-image:
      linear-gradient(var(--grid-major) 1px, transparent 1px),
      linear-gradient(90deg, var(--grid-major) 1px, transparent 1px),
      linear-gradient(var(--grid) 1px, transparent 1px),
      linear-gradient(90deg, var(--grid) 1px, transparent 1px);
    background-size: 90px 90px, 90px 90px, 18px 18px, 18px 18px;
    box-shadow: var(--elev-floating);
    outline: 1px solid var(--line);
  }
  .window::before {
    /* the macOS traffic lights, drawn here only to show the room left for them */
    content: '';
    position: absolute;
    z-index: 20;
    top: 20px;
    left: 20px;
    width: 12px;
    height: 12px;
    border-radius: 50%;
    background: #ff5f57;
    box-shadow:
      20px 0 #febc2e,
      40px 0 #28c840;
  }
  .glance-row,
  .tabs-row {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    padding: var(--space-2) var(--space-4);
    background: var(--chrome);
  }
  .tabs-row {
    padding-top: 0;
    padding-bottom: var(--space-1);
    border-bottom: 1px solid var(--line);
  }
  .board {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: var(--space-6);
    padding: var(--space-6) var(--space-5) 96px;
  }
  .items {
    margin: var(--space-3) 0 0;
    padding: 0;
    list-style: none;
    display: grid;
    gap: var(--space-2);
  }
  .items li {
    display: flex;
    align-items: center;
    gap: var(--space-2);
  }
  .items s {
    color: var(--ink-3);
  }
  .note-wrap {
    min-width: 0;
  }
  .cards {
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 1fr);
    gap: var(--space-4);
    margin-top: var(--space-3);
  }
  .col {
    display: grid;
    gap: var(--space-3);
    align-content: start;
  }
  .colh {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--ink-2);
    font: var(--weight-hand-label) var(--text-hand-label) / var(--leading-hand-label) var(--font-hand);
  }
  .cards small {
    display: block;
    font: 400 var(--text-mono-sm) / var(--leading-mono-sm) var(--font-mono);
    opacity: 0.85;
  }
  .ringed {
    position: relative;
    display: block;
  }
  .toast-slot {
    position: absolute;
    left: 50%;
    bottom: var(--space-5);
    translate: -50% 0;
    z-index: var(--z-toast);
  }
  .side {
    min-width: 0;
  }
  .leftcol {
    min-width: 0;
  }
  .side h2:first-child {
    margin-top: 0;
  }
  .strip {
    overflow: visible;
    border: 1px solid var(--line);
    border-radius: var(--radius-lg);
  }
  .strip :global(.bar) {
    border-radius: var(--radius-lg);
    border-bottom: 0;
  }

  /* icons */
  .icons {
    display: grid;
    grid-template-columns: 180px 1fr;
    gap: var(--space-6);
    padding: var(--space-5);
    border: 1px solid var(--line);
    border-radius: var(--radius-xl);
    background: var(--surface);
  }
  .appicon {
    display: grid;
    justify-items: center;
    gap: var(--space-3);
    align-content: start;
  }
  .appicon img {
    width: 120px;
  }
  .tier h3 {
    margin-top: var(--space-5);
  }
  .tier h3:first-child {
    margin-top: 0;
  }
  .grid {
    display: grid;
    grid-template-columns: repeat(auto-fill, minmax(70px, 1fr));
    gap: var(--space-2);
  }
  figure {
    display: grid;
    justify-items: center;
    gap: 6px;
    margin: 0;
    padding: var(--space-2) var(--space-1);
    border-radius: var(--radius-md);
    background: var(--paper);
  }
  .big {
    color: var(--ink);
  }
  .small {
    display: flex;
    align-items: center;
    gap: var(--space-2);
    color: var(--ink-2);
  }
  figcaption {
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
  }

  /* panels */
  .panel {
    padding: var(--space-5) var(--space-6);
    border: 1px solid var(--line);
    border-radius: var(--radius-xl);
    background: var(--surface);
  }
  .matrix {
    border-collapse: collapse;
  }
  .matrix th {
    padding: 0 var(--space-4) var(--space-2) 0;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
    text-align: left;
  }
  .matrix td {
    padding: 0 var(--space-4) var(--space-3) 0;
  }
  .row {
    display: flex;
    align-items: center;
    gap: var(--space-3);
  }
  .row.wrap {
    flex-wrap: wrap;
    margin-bottom: var(--space-3);
  }
  .st {
    display: grid;
    justify-items: center;
    gap: var(--space-1);
  }
  .st em,
  .tk em,
  .rad em,
  .strk em {
    color: var(--ink-3);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
    font-style: normal;
  }
  .tipdemo {
    margin-left: 40px;
  }
  .cols {
    display: grid;
    grid-template-columns: 1fr 1.3fr 1fr;
    gap: var(--space-8);
    margin-top: var(--space-2);
  }
  .cols.two {
    grid-template-columns: 1fr 1.4fr;
  }
  .stack {
    display: grid;
    gap: var(--space-3);
  }
  .emptybox {
    border: 1px dashed var(--line-strong);
    border-radius: var(--radius-lg);
    background: var(--paper);
  }
  .nh {
    padding: var(--space-2) var(--space-3);
    border-radius: var(--radius-md);
    background: var(--paper);
  }

  /* tokens */
  .tokens {
    margin-bottom: var(--space-8);
    display: grid;
    grid-template-columns: 1.2fr 1fr 1.2fr;
    gap: var(--space-8);
  }
  .spec {
    display: grid;
    grid-template-columns: 120px 1fr;
    align-items: center;
    gap: var(--space-3);
    min-height: 24px;
  }
  .tk {
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
  }
  .bar {
    height: 8px;
    border-radius: 2px;
    background: var(--accent);
  }
  .rad {
    display: grid;
    place-items: center;
    width: 56px;
    height: 44px;
    border: 1.5px solid var(--line-strong);
    background: var(--paper);
    color: var(--ink-2);
    font-size: var(--text-body-xs);
  }
  .strk {
    display: grid;
    gap: 4px;
    width: 72px;
    color: var(--ink-2);
    font-size: var(--text-body-xs);
  }
  .strk i {
    background: var(--ink);
    border-radius: 2px;
  }
  .elev {
    display: grid;
    place-items: center;
    width: 84px;
    height: 56px;
    border-radius: var(--radius-md);
    background: var(--surface);
    color: var(--ink-2);
    font-size: var(--text-body-xs);
    margin: 0 var(--space-2) var(--space-3) 0;
  }
  .sw {
    gap: var(--space-2) var(--space-3);
  }
  .sw span {
    display: inline-flex;
    align-items: center;
    gap: 6px;
    color: var(--ink-2);
    font: var(--weight-mono-sm) var(--text-mono-sm) / 1 var(--font-mono);
  }
  .sw i {
    width: 18px;
    height: 18px;
    border-radius: var(--radius-sm);
    box-shadow: inset 0 0 0 1px var(--line-strong);
  }
  .rules {
    margin-bottom: var(--space-2);
    display: grid;
    grid-template-columns: repeat(4, 1fr);
    gap: var(--space-3) var(--space-5);
    color: var(--ink-2);
    font-size: var(--text-body-sm);
    line-height: var(--leading-body-sm);
  }
  .rules b {
    display: block;
    color: var(--ink);
    margin-bottom: var(--space-1);
  }
</style>
