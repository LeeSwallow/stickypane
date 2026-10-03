# Changelog

## Unreleased (towards v0.1.0)

The first version. stickypane shows the files of a project's `.sticky/`
folder as a board in a terminal pane, for a coding agent and its user to
share.

- Notes are plain files: Markdown notes with boards, checklists, charts and
  forms by front matter; `.log` files followed live; `.sh` scripts with a
  Run button; folders as books that scroll as one note; Mermaid blocks
  drawn as diagrams.
- The screen is tiled with fixed panes that scroll inside; the arrangement
  (open, size, color, pin, order, names) lives in `sticky.json`.
- Keys and mouse: focus, open, zoom, resize, reorder, move to a folder,
  trash with undo, a built-in vi-like editor, ten color themes.
- Command line and MCP server for agents: `show`, one-thing edits (`todo`,
  `card`, `chart`, `log`, `set`), `rm`/`mv`/`restore`/`archive`/`link`,
  forms with `answers` and `wait`, `init --skill`.

Known limits: macOS and Linux only; a program that keeps a log file open and
appends to it stops being followed after the board edits that file; no
counts or visual mode in the editor; wide diagrams in narrow panes show
their source.
