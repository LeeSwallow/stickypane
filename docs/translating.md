# Translating the screen

The board speaks English and Korean. Adding a language is one JSON file.

## What is translated, and what is not

The rule: **what a person reads is translated; what a program, an agent or a
log reads is not.**

- Translated: everything the screen says to the user — key hints, messages
  on the bottom line, dialog titles, the help screen, the empty-note hints
  inside widgets — and what the person's commands print: `setup` (its
  status, plan, questions and refusals), `init`, `config`, `theme`,
  `language`, `keys` and its warnings, `env`, `help`, a usage error, the
  refusal of a setting. These live in `internal/core/i18n/`.
- Not translated: logs (`log.Printf`, files under `~/.stickypane`),
  machine output (`--json`, the bridge protocol, MCP tool results), the
  lines of the commands agents run on notes (`write`, `todo`, `card`,
  `show`, `kinds`, `watch` …) and the agent guide (`plugin/guide.md`,
  `plugin/rules/`, the skills): their reader is a model, and English
  is what models read best; the guide tells the agent to answer in the
  user's language. Nor identifiers: key names (`tab`, `ctrl+f`), action
  names, setting keys and values, command names, flags, file names.
- An error keeps its English for logs and for what wraps it, and is shown
  translated: an error that is one of the sentences is an `i18n.Message`
  (or has a `Say` method, as `prefs.Refused` does), and the edge that
  shows it calls `i18n.Speak(err, tr)`. The details inside it that come
  from the system or from a note command stay as they are.
- The README has translations next to it (`README.ko.md`), each saying
  which commit it was translated at. English is canonical.

## How it works

`internal/core/i18n/i18n.go` holds `Strings`, with English in `English()`. A
translation is `internal/core/i18n/translations/<code>.json` with the same
field names; fields it leaves out keep their English text, so a partial
translation is fine. `labels` maps short phrases (key-hint labels, widget
hints, `%d cards`) to their translation; phrases not in the map stay
English. Messages use `{{.Name}}` placeholders, so a language can order
the parts as it likes. `actions` says what each of the board's keys does,
by the action's name (`note-archive`) or a line of the help (`help.size`);
the English, and the list of names, is the table in
`internal/surface/app/keymap.go`. `says` is what the person's commands print,
keyed like `setup.carries`; the English is in `internal/core/i18n/says.go`,
and a translation must have every key with the same placeholders
(`go test ./internal/core/i18n/` checks). A print in those commands names a
key, never an English sentence (`go test ./cmd/stickypane/` checks).

The language comes from `sticky.json` (`stickypane config language ko`) or, with
`auto`, from `STICKYPANE_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG`.

## Adding a language

1. Copy `translations/ko.json` to `translations/<code>.json` and translate
   the values. Delete any you do not want to translate, except under
   `says`: those are all translated.
2. Keep help lines under 40 cells (`go test ./internal/core/i18n/` checks).
3. `go test ./...`. A field name that does not exist is an error, so a
   typo cannot hide.
4. Add the language to the table in `README.md` and open a pull request
   labelled `docs`.

## Korean

One voice for the app (`desktop/frontend/src/i18n/ko.json`) and the board
(`internal/core/i18n/translations/ko.json`):

- **Buttons, labels, menus:** a bare noun or verb, no ending — `보드 추가`, `연결…`, `기본값으로`.
- **Hints under a setting:** a noun phrase or one plain sentence; say only what the label does not. Cut before you reword.
- **Messages (confirm, result, error):** 해요체, one sentence, the thing first — `'a' 노트를 보관했어요. u 키로 되돌릴 수 있어요.` An error says what happened and what to do.
- **Words:** one term per thing (노트, 보드, 탭, 칸, 카드 n장); code, commands and paths stay as written. `stickypane help` is reference text and keeps 합니다체, like the tagline.
