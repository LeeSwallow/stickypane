# Translating the screen

The board speaks English and Korean. Adding a language is one JSON file.

## What is translated, and what is not

- Translated: everything the screen says to the user — key hints, messages
  on the bottom line, dialog titles, the help screen, the empty-note hints
  inside widgets. These live in `internal/core/i18n/`.
- Not translated: key names (`tab`, `ctrl+f`), file formats, command names,
  the agent guide (`plugin/guide.md` and `plugin/rules/`) and the plugin
  skills. The agent's reader is a model; every agent instruction file we looked at in
  other projects is English, and the guide tells the agent to answer the
  user in the user's language.
- The README has translations next to it (`README.ko.md`), each saying
  which commit it was translated at. English is canonical.

## The app

The desktop app follows the same line, for everything a person sees in
it: labels, hints, buttons, toasts, dialogs, empty states, tooltips and
`aria-label`s (a screen reader reads them), the tray and the menu bar,
the folder picker, a window's title, and an error shown to a person.
Logs, identifiers, file paths, command lines shown as code and the notes
themselves stay as they are. The words are in
`desktop/frontend/src/i18n/{en,ko}.json`; a count is `{n}`, and English
gives its singular as `key.one`. An error the Go shell knows goes to the
window as a code and is said from `error.<code>`; any other comes after
what could not be done (`failed.*`), in its own words. What the app shows
from `stickypane` runs with `STICKYPANE_LANG` set to the window's
language. The few words of Go's own (tray, menu bar, picker) are in
`desktop/internal/shell`. `src/i18n/words.test.ts` fails on a literal
word in a component.

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
`internal/surface/app/keymap.go`.

The language comes from `sticky.json` (`stickypane config language ko`) or, with
`auto`, from `STICKYPANE_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG`.

## Adding a language

1. Copy `translations/ko.json` to `translations/<code>.json` and translate
   the values. Delete any you do not want to translate.
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
