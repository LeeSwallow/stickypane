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
