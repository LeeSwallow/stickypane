# Third-party notices

stickypane is MIT licensed (see LICENSE). It builds on these projects.

## Libraries

| Project | License | Used for |
| --- | --- | --- |
| [charm.land/bubbletea](https://github.com/charmbracelet/bubbletea), [lipgloss](https://github.com/charmbracelet/lipgloss), [x/ansi](https://github.com/charmbracelet/x) | MIT | the terminal UI and styling |
| [AlexanderGrooff/mermaid-ascii](https://github.com/AlexanderGrooff/mermaid-ascii) | MIT | drawing Mermaid blocks as text |
| [fsnotify/fsnotify](https://github.com/fsnotify/fsnotify) | BSD-3-Clause | following the notes folder |

These are the modules stickypane uses directly. With the modules they use
in turn, the program links 22, all under MIT or BSD-3-Clause. Every release
archive carries `THIRD_PARTY_LICENSES.txt` with each one's license as it
ships it; `scripts/licenses.sh` writes that file and fails when a module's
license is missing or not MIT, BSD or Apache-2.0, and CI runs it on every
pull request.

Markdown and the one-line editor are stickypane's own (`internal/markdown`,
`internal/lineedit`).

## The desktop app

The desktop app (`overlay/`, shipped as `Stickypane.app` and
`stickypane-app-<version>-linux-<arch>.tar.gz`) carries the stickypane
program above and, in addition, these.

| Project | License | Used for |
| --- | --- | --- |
| [wailsapp/wails](https://github.com/wailsapp/wails) v3 and `@wailsio/runtime` | MIT | the window, tray and webview |
| [modernc.org/sqlite](https://gitlab.com/cznic/sqlite) | BSD-3-Clause | the index of boards |
| [sveltejs/svelte](https://github.com/sveltejs/svelte) | MIT | the interface |
| [rough-stuff/rough](https://github.com/rough-stuff/rough) (roughjs) | MIT | hand-drawn strokes, in the app and the DMG window |
| [Gaegu](https://github.com/fontsource/font-files/tree/main/fonts/google/gaegu) | OFL-1.1 | the handwriting font |
| [Pretendard](https://github.com/orioncactus/pretendard) | OFL-1.1 | the body font |
| [JetBrains Mono](https://github.com/JetBrains/JetBrainsMono) | OFL-1.1 | the code font |

`overlay/scripts/licenses.sh` checks every Go module and npm package of the
app against MIT, BSD, Apache-2.0, ISC and OFL-1.1, with a few listed
exceptions. One of them ships in the app: robust-predicates (pulled in by
mermaid through d3-delaunay), under the Unlicense, a public-domain
dedication.

The copyright lines of the projects above, as their licenses state them:

- Wails: Copyright (c) 2018-Present Lea Anthony (MIT)
- modernc.org/sqlite: Copyright (c) 2017 The Sqlite Authors (BSD-3-Clause)
- Svelte: Copyright (c) 2016-2025 Svelte Contributors (MIT)
- roughjs: Copyright (c) 2019 Preet Shihn (MIT)
- Gaegu: Copyright 2018 The Gaegu Project Authors (OFL-1.1)
- Pretendard: Copyright (c) 2021, Kil Hyung-jin, with Reserved Font Name
  Pretendard (OFL-1.1)
- JetBrains Mono: Copyright 2020 The JetBrains Mono Project Authors (OFL-1.1)

The full texts travel with the app. `Stickypane.app/Contents/Resources/` and
the Linux archive hold `LICENSE`, this file and a `licenses/` folder:
`licenses/stickypane-app.txt` has the license of every Go module linked into
the app and every npm package bundled in it (the SIL Open Font License 1.1
with each font's copyright among them), as each package ships it, written by
`overlay/scripts/licenses.sh --write`; `licenses/stickypane.txt` has those of
the bundled stickypane program, written by `scripts/licenses.sh`.

## Color palettes

The built-in themes use the palettes of these editor themes, with the color
values taken from their sources as they are. The palettes are used under the
projects' licenses; nothing else of those projects is included.

| Theme | Source | License |
| --- | --- | --- |
| `catppuccin-mocha`, `catppuccin-latte` | [catppuccin/nvim](https://github.com/catppuccin/nvim), `lua/catppuccin/palettes/` | MIT |
| `tokyonight-night`, `tokyonight-day` | [folke/tokyonight.nvim](https://github.com/folke/tokyonight.nvim), `lua/tokyonight/colors/`, `extras/lua/tokyonight_day.lua` | Apache-2.0 |
| `gruvbox-dark`, `gruvbox-light` | [ellisonleao/gruvbox.nvim](https://github.com/ellisonleao/gruvbox.nvim), `lua/gruvbox.lua` | MIT |
| `nord` | [gbprod/nord.nvim](https://github.com/gbprod/nord.nvim), `lua/nord/colors.lua` | WTFPL |
| `dracula` | [Mofiqul/dracula.nvim](https://github.com/Mofiqul/dracula.nvim), `lua/dracula/palette.lua` | MIT |

tokyonight.nvim is Copyright (c) Folke Lemaitre and licensed under the Apache
License, Version 2.0; the color values are used in accordance with that
license. The theme names are those of the original projects.
