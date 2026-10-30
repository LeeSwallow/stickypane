package shell

import (
	"net/url"
	"testing"

	"github.com/wailsapp/wails/v3/pkg/application"
)

// tornPage is the page of the window torn off for 04-차트/02-spark.md, as
// the real platform asks Wails for it. frontend/src/lib/shell.test.ts reads
// the same string, so the two sides agree on it (#62).
const tornPage = "/?chrome=own&note=04-%EC%B0%A8%ED%8A%B8%2F02-spark.md&window=note"

// The options the real platform gives Wails for a torn-off note: the fake
// platform's tests never see them.
func TestATornNoteWindowAsWailsMakesIt(t *testing.T) {
	for _, goos := range []string{"darwin", "linux", "windows"} {
		o := webviewOptions(WindowOptions{
			Kind:      KindNote,
			Name:      "note:/p#04-차트/02-spark.md",
			Title:     "02-spark",
			URL:       NoteURL("04-차트/02-spark.md"),
			Bounds:    Rect{X: 540, Y: 260, Width: NoteWidth, Height: NoteHeight},
			Placed:    true,
			Frameless: true,
			Pinned:    true,
		}, goos)
		if o.URL != tornPage {
			t.Errorf("%s: URL %q, want %q", goos, o.URL, tornPage)
		}
		q, err := url.Parse(o.URL)
		if err != nil || q.Query().Get("window") != "note" || q.Query().Get("note") != "04-차트/02-spark.md" {
			t.Errorf("%s: the page cannot tell the note from %q", goos, o.URL)
		}
		if !o.Frameless || o.Hidden || !o.AlwaysOnTop {
			t.Errorf("%s: frameless %v, hidden %v, on top %v; want a shown, frameless window on top", goos, o.Frameless, o.Hidden, o.AlwaysOnTop)
		}
		if o.Width != NoteWidth || o.Height != NoteHeight || o.MinWidth != NoteMinWidth || o.MinHeight != NoteMinHeight {
			t.Errorf("%s: size %dx%d (min %dx%d)", goos, o.Width, o.Height, o.MinWidth, o.MinHeight)
		}
		if o.InitialPosition != application.WindowXY || o.X != 540 || o.Y != 260 {
			t.Errorf("%s: placed at %v %d,%d; want WindowXY 540,260", goos, o.InitialPosition, o.X, o.Y)
		}
	}
}

// A project window on macOS keeps the native traffic lights over the
// page's own title bar; elsewhere it draws its own.
func TestAProjectWindowAsWailsMakesIt(t *testing.T) {
	o := webviewOptions(WindowOptions{Kind: KindProject, URL: "/?window=project", Bounds: Rect{Width: 840, Height: 600}, Frameless: true}, "darwin")
	if o.Frameless || o.Mac.TitleBar != application.MacTitleBarHiddenInset || o.URL != "/?chrome=mac&window=project" {
		t.Errorf("darwin project window: frameless %v, url %q", o.Frameless, o.URL)
	}
	if o.InitialPosition != application.WindowCentered {
		t.Error("a window never placed opens centred")
	}
	o = webviewOptions(WindowOptions{Kind: KindProject, URL: "/?window=project", Frameless: true}, "linux")
	if !o.Frameless || o.URL != "/?chrome=own&window=project" {
		t.Errorf("linux project window: frameless %v, url %q", o.Frameless, o.URL)
	}
}
