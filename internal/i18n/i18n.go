// Package i18n holds every sentence the board says to the user, in English
// with translations laid over it. The model is lazygit's: one struct of
// strings, English as the base, a JSON file per language that overrides
// the fields it has, so a partial translation is safe and a missing key
// falls back to English. Key names are never translated; what a key does
// is.
package i18n

import (
	"github.com/LeeSwallow/stickypane/internal/env"

	"bytes"
	"embed"
	"encoding/json"
	"fmt"
	"io/fs"
	"path"
	"sort"
	"strings"
	"text/template"
)

//go:embed translations/*.json
var translations embed.FS

// Strings is everything the board says. Fields with {{.Name}}-style
// placeholders are filled with Fill, so a language can put the parts in
// its own order.
type Strings struct {
	// Labels maps a short English phrase used in key hints and inside
	// notes ("next", "tick", "no items yet: …", "%d cards") to its
	// translation. A phrase that is not in the map stays English.
	Labels map[string]string `json:"labels"`

	// Dialog titles and prompts.
	AddANote  string `json:"addANote"`
	Example   string `json:"example"`
	Keys      string `json:"keys"`
	Title     string `json:"title"`
	Name      string `json:"name"`
	Jot       string `json:"jot"`
	Folder    string `json:"folder"`
	TopLevel  string `json:"topLevel"`
	NewFolder string `json:"newFolder"`
	MoveTo    string `json:"moveTo"`

	// Empty screens.
	NoNotes         string `json:"noNotes"`
	NoNotesHint     string `json:"noNotesHint"`
	NothingOpen     string `json:"nothingOpen"`
	NothingOpenHint string `json:"nothingOpenHint"`

	// Messages on the bottom line.
	ThemeChosen         string `json:"themeChosen"`
	ThemeNotSaved       string `json:"themeNotSaved"`
	LanguageChosen      string `json:"languageChosen"`
	ArrangementNotSaved string `json:"arrangementNotSaved"`
	CannotReadNotes     string `json:"cannotReadNotes"`
	ViewsBroken         string `json:"viewsBroken"`
	CannotShowNote      string `json:"cannotShowNote"`
	NoteRemoved         string `json:"noteRemoved"`
	Conflict            string `json:"conflict"`
	WriteFailed         string `json:"writeFailed"`
	Running             string `json:"running"`
	CannotCreate        string `json:"cannotCreate"`
	CannotReadNote      string `json:"cannotReadNote"`
	CannotReadFile      string `json:"cannotReadFile"`
	FileChangedSave     string `json:"fileChangedSave"`
	FileChangedEdit     string `json:"fileChangedEdit"`
	EditorFailed        string `json:"editorFailed"`
	CannotChange        string `json:"cannotChange"`
	OnlyMarkdownTitle   string `json:"onlyMarkdownTitle"`
	CannotArchive       string `json:"cannotArchive"`
	Archived            string `json:"archived"`
	ConfirmDelete       string `json:"confirmDelete"`
	CannotDelete        string `json:"cannotDelete"`
	Deleted             string `json:"deleted"`
	NothingToUndo       string `json:"nothingToUndo"`
	CannotRestore       string `json:"cannotRestore"`
	Restored            string `json:"restored"`
	CannotMove          string `json:"cannotMove"`
	AlreadyRunning      string `json:"alreadyRunning"`
	ConfirmRun          string `json:"confirmRun"`
	CannotWriteLog      string `json:"cannotWriteLog"`
	CouldNotRun         string `json:"couldNotRun"`
	RunEnded            string `json:"runEnded"`
	ConfirmSend         string `json:"confirmSend"`
	Sending             string `json:"sending"`
	WatchUnavailable    string `json:"watchUnavailable"`

	// The help screen, one key per line, under 40 cells wide.
	Help string `json:"help"`
}

// English is the base every language starts from.
func English() Strings {
	return Strings{
		Labels: map[string]string{},

		AddANote: "Add a note", Example: "example", Keys: "Keys", Title: "Title", Name: "Name",
		Jot: "Jot", Folder: "Folder", TopLevel: "(top level)", NewFolder: "New folder…",
		MoveTo: "Move {{.Name}} to",

		NoNotes: "No notes yet.", NoNotesHint: "Press N to jot one down,\nor ask your agent to stick a note here.",
		NothingOpen: "Nothing is open.", NothingOpenHint: "Pick a note with tab and press enter.",

		ThemeChosen:         "Theme: {{.Name}}. T tries the next one.",
		ThemeNotSaved:       "The theme was not saved: {{.Err}}",
		LanguageChosen:      "Language: {{.Name}}.",
		ArrangementNotSaved: "The arrangement was not saved: {{.Err}}",
		CannotReadNotes:     "Cannot read notes: {{.Err}}",
		ViewsBroken:         "{{.Err}}. Notes are shown as they arrange themselves.",
		CannotShowNote:      "Cannot show this note: {{.Err}}",
		NoteRemoved:         "The note was removed.",
		Conflict:            "The file changed on disk, so the change was not applied.",
		WriteFailed:         "Write failed: {{.Err}}",
		Running:             "running…",
		CannotCreate:        "Cannot create the note: {{.Err}}",
		CannotReadNote:      "Cannot read the note: {{.Err}}",
		CannotReadFile:      "Cannot read the file: {{.Err}}",
		FileChangedSave:     "The file changed on disk. :w! overwrites it, :e! loads it again.",
		FileChangedEdit:     "The file changed on disk. :e! loads it, :w! overwrites it.",
		EditorFailed:        "Editor failed",
		CannotChange:        "This note cannot be changed from here: {{.Err}}",
		OnlyMarkdownTitle:   "Only a Markdown note has a title.",
		CannotArchive:       "Cannot archive the note: {{.Err}}",
		Archived:            "Moved {{.Name}} to archive/. u brings it back.",
		ConfirmDelete:       "Delete {{.Name}}? (y/n)",
		CannotDelete:        "Cannot delete the note: {{.Err}}",
		Deleted:             "Deleted {{.Name}}. u brings it back.",
		NothingToUndo:       "Nothing to undo.",
		CannotRestore:       "Cannot bring {{.Name}} back: it is still in {{.Where}} ({{.Err}})",
		Restored:            "Restored {{.Name}}.",
		CannotMove:          "Cannot move the note: {{.Err}}",
		AlreadyRunning:      "{{.Name}} is already running.",
		ConfirmRun:          "Run {{.Name}} in {{.Dir}}? (y/n)",
		CannotWriteLog:      "Cannot write {{.Log}}: {{.Err}}",
		CouldNotRun:         "{{.Name}} could not be run: {{.Err}}",
		RunEnded:            "{{.Name}} ended: exit {{.Code}} after {{.Took}}. Its output is in {{.Log}}.",
		ConfirmSend:         "Send {{.Request}}?{{if .Hooks}} It runs {{.Hooks}}.{{end}} (y/n)",
		Sending:             "Sending {{.Name}}…",
		WatchUnavailable:    "File watching is unavailable. Press r to refresh.",

		Help: `Notes
  tab              next note
  shift+tab        previous note
  enter            open, then zoom
  z                zoom
  o                open or close
  + -              bigger, smaller
  { }              move earlier, later
  , .              turn a folder's pages
  m                move to a folder
  N                jot a note
  a                add by shape
  e                edit here (like vi)
  E                edit in $EDITOR
  p                pin
  c                change color
  R                rename
  x                move to archive
  D                delete (to .trash)
  u                undo x or D
  r                reload
  T                next theme
  i /              index of every note
  S                settings
  j k g G          scroll in the note
  pgdn pgup        a page down, up
  [ ]              previous, next screen
  1-9 ( )          switch tab
  ?                this help
  q                quit

Zoomed note
  esc              back
  j k g G          scroll

Open board
  h l              change column
  j k              change card
  H L              move a card sideways
  J K              reorder a card
  n                new card

Open checklist
  j k              change item
  space            tick an item
  n                new item

Open form
  j k              change control
  enter space      choose, type, press

Editing a note
  i a o            insert text
  esc              back to commands
  h j k l w b      move
  x dd D u         delete, undo
  /text  n         search
  ctrl+f ctrl+b    next, previous page
  :w  :q  :wq      save, quit, both

Mouse
  click a title    open, focus, close
  click a note     focus; tick, choose
  double click     zoom
  wheel            scroll that note`,
	}
}

// Languages returns the codes a translation exists for, English first.
func Languages() []string {
	codes := []string{"en"}
	entries, _ := fs.ReadDir(translations, "translations")
	for _, e := range entries {
		codes = append(codes, strings.TrimSuffix(e.Name(), path.Ext(e.Name())))
	}
	sort.Strings(codes[1:])
	return codes
}

// Load returns the strings for a language code such as "ko": English with
// the translation's fields laid over it. "en" and "" are English. A code
// without a translation is an error.
func Load(code string) (Strings, error) {
	s := English()
	code = normalize(code)
	if code == "en" || code == "" {
		return s, nil
	}
	b, err := translations.ReadFile("translations/" + code + ".json")
	if err != nil {
		return s, fmt.Errorf("no translation for %q", code)
	}
	dec := json.NewDecoder(bytes.NewReader(b))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&s); err != nil {
		return English(), fmt.Errorf("translations/%s.json: %w", code, err)
	}
	return s, nil
}

// Pick returns the strings for a choice: a language code, or "auto" (or
// nothing) for what the environment says. A choice without a translation
// follows the environment too.
func Pick(choice string) Strings {
	if c := normalize(choice); c != "" && c != "auto" {
		if s, err := Load(c); err == nil {
			return s
		}
	}
	s, _ := Load(Detect())
	return s
}

// Detect returns the language the environment asks for, as "ko" or "en":
// STICKYPANE_LANG, then LC_ALL, LC_MESSAGES and LANG, then what the system
// itself is set to (macOS's and Windows' own locale, for a terminal that
// sets no variable). Nothing usable means English.
func Detect() string {
	if code := normalize(env.Detect().Locale); code != "" {
		return code
	}
	return "en"
}

// normalize turns "ko_KR.UTF-8", "ko-KR", "Korean" or "ko" into "ko", and
// "C", "POSIX" or "" into "".
func normalize(v string) string {
	v = strings.TrimSpace(v)
	if i := strings.IndexAny(v, ".@"); i >= 0 {
		v = v[:i]
	}
	v = strings.ToLower(strings.ReplaceAll(v, "_", "-"))
	if i := strings.Index(v, "-"); i >= 0 {
		v = v[:i]
	}
	switch v {
	case "c", "posix":
		return ""
	}
	return v
}

// Fill puts values into a string's {{.Name}} placeholders. A string that
// cannot be filled is returned as it is, so a broken translation never
// takes a message away.
func Fill(s string, values map[string]any) string {
	if !strings.Contains(s, "{{") {
		return s
	}
	t, err := template.New("").Parse(s)
	if err != nil {
		return s
	}
	var out strings.Builder
	if err := t.Execute(&out, values); err != nil {
		return s
	}
	return out.String()
}

// L translates a short phrase through Labels, keeping it when there is no
// translation.
func (s Strings) L(phrase string) string {
	if t, ok := s.Labels[phrase]; ok && t != "" {
		return t
	}
	return phrase
}
