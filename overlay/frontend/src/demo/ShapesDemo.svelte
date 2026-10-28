<script lang="ts">
  // Dev only (?demo=shapes&mode=dark|light&style=doodle|neat&lang=en|ko
  // &zoom=<note>&probe=<url>): the real board over the shapes' fixture
  // (src/shapes/fixtures/board.json, a real bridge's board.load), so every
  // shape can be seen and screenshotted in both modes and both styles.
  // probe= adds an html note that tries to reach that URL every way a page
  // can and to run its own script (probe.ts): a server there sees no
  // request when the sandbox holds.
  // Nothing here ships.
  import Board from '../board/Board.svelte'
  import type { Board as BoardData, Note } from '../board/model'
  import { THEMES } from '../lib/theme/palettes'
  import { fixture } from '../shapes/fixtures/fixtures'
  import { fakeProject } from './fakeProject'
  import { probePage } from './probe'

  const q = new URLSearchParams(location.search)
  const mode = q.get('mode') === 'light' ? 'light' : 'dark'
  const lang = q.get('lang') === 'ko' ? 'ko' : 'en'
  const style = q.get('style') === 'neat' ? 'neat' : 'doodle'
  const probe = q.get('probe')

  const probeNote = (url: string): Note => ({
    ...fixture.tabs[0].notes.find((n) => n.kind === 'html')!,
    name: 'probe.html',
    open: true,
    title: 'Sandbox probe',
    size: '1/2',
    data: { html: probePage(url) },
  })

  const notes = fixture.tabs.flatMap((t) => t.notes).map((n) => ({ ...n, open: true, tab: '' }))
  const palette = THEMES.find((t) => t.name === (mode === 'dark' ? 'stickypane-dark' : 'stickypane-light'))
  const board: BoardData = {
    ...fixture,
    title: 'shapes',
    tabs: [{ name: '', title: 'shapes', notes: probe ? [probeNote(probe), ...notes] : notes }],
    settings: { ...fixture.settings, theme: palette?.name ?? '', style, language: { choice: lang, resolved: lang }, tab: '' },
    palette,
  }
  const api = fakeProject(board, mode, lang)
</script>

<Board {api} zoom={q.get('zoom') ?? undefined} />
