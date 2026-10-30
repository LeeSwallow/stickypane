# Vision

stickypane is sticky notes for coding agents: they pin, you glance. The
board lives in the terminal beside the agent, and the desktop app draws the
same board for those who want it. These are the principles decisions are
checked against. When a feature fails one of them, it is not added.

1. **The folder is the product.** A note is a file the user owns, readable
   without stickypane, in the project's repository. The board keeps no
   state that is not a file in that folder. What lives outside it, such as
   the app's list of boards and its window places, belongs to one computer
   and can be thrown away and rebuilt.
2. **The agent needs nothing new.** Writing a Markdown file is enough; the
   commands exist so that an agent never has to read a file to change one
   thing in it. The guide the agent reads fits on one screen.
3. **The user acts on the board, and the agent sees it.** Ticking an item,
   moving a card, answering a form changes the same file the agent reads,
   in the terminal or in the app alike. Nothing the user does is told to
   the agent in words.
4. **Nothing to configure.** One program and a folder are all the board
   needs. The agent plugin, the MCP server and the desktop app are each
   opt-in, and the board works the same without them. Nothing listens on
   the network, and no config file is written by hand: the one settings
   file is written by the board as the user arranges it.
5. **Any terminal, any agent.** A plain window, tmux, WezTerm, Zellij;
   Claude Code, Codex, anything that can edit files. The terminal stays
   the first screen: the app does nothing the terminal cannot, except
   draw.
6. **The screen is calm.** Fixed panes that scroll inside, one accent for
   focus, muted everything else. A note is never cut and never hidden.
   What changed moves once and settles; nothing loops or blinks.

## What it is not

Not a task manager with its own data model, not a chat client, not a
replacement for the agent's own UI, not a web app or a cloud service, and
not a general dashboard framework.
