# Translating the screen

The board speaks English and Korean. Adding a language is one JSON file.

## What is translated, and what is not

- Translated: everything the screen says to the user — key hints, messages
  on the bottom line, dialog titles, the help screen, the empty-note hints
  inside widgets. These live in `internal/i18n/`.
- Not translated: key names (`tab`, `ctrl+f`), file formats, command names,
  the agent guide (`internal/initcmd/guide.md`) and the plugin skills. The
  agent's reader is a model; every agent instruction file we looked at in
  other projects is English, and the guide tells the agent to answer the
  user in the user's language.
- The README has translations next to it (`README.ko.md`), each saying
  which commit it was translated at. English is canonical.

## How it works

`internal/i18n/i18n.go` holds `Strings`, with English in `English()`. A
translation is `internal/i18n/translations/<code>.json` with the same
field names; fields it leaves out keep their English text, so a partial
translation is fine. `labels` maps short phrases (key-hint labels, widget
hints, `%d cards`) to their translation; phrases not in the map stay
English. Messages use `{{.Name}}` placeholders, so a language can order
the parts as it likes.

The language comes from `sticky.json` (`stickypane language ko`) or, with
`auto`, from `STICKYPANE_LANG`, `LC_ALL`, `LC_MESSAGES`, `LANG`.

## Adding a language

1. Copy `translations/ko.json` to `translations/<code>.json` and translate
   the values. Delete any you do not want to translate.
2. Keep help lines under 40 cells (`go test ./internal/i18n/` checks).
3. `go test ./...`. A field name that does not exist is an error, so a
   typo cannot hide.
4. Add the language to the table in `README.md` and open a pull request
   labelled `docs`.
