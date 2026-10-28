package shell

import (
	"path/filepath"
	"runtime"
	"strconv"
	"sync"

	"github.com/LeeSwallow/stickypane/overlay/internal/index"
	"github.com/wailsapp/wails/v3/pkg/application"
)

// Tray is what the tray shows: how many boards are moving now, and the
// most recent five (08 §8.1).
type Tray struct {
	Active int
	Recent []TrayItem
}

// TrayItem is one recent project.
type TrayItem struct {
	Title string
	Path  string
}

// trayOf takes the tray from the home list: active rows first, then the
// days, newest first.
func trayOf(g index.Groups) Tray {
	t := Tray{Active: len(g.Active)}
	add := func(r index.Row) {
		if len(t.Recent) < trayRecent {
			title := r.Title
			if title == "" {
				title = filepath.Base(r.Path)
			}
			t.Recent = append(t.Recent, TrayItem{Title: title, Path: r.Path})
		}
	}
	for _, r := range g.Active {
		add(r)
	}
	for _, d := range g.Days {
		for _, r := range d.Rows {
			add(r)
		}
	}
	return t
}

// trayWords are the tray menu's two words, in the home window's language.
func trayWords(lang string) (home, quit string) {
	if lang == "ko" {
		return "홈 열기", "stickypane 종료"
	}
	return "Open Home", "Quit stickypane"
}

// wailsTray is the tray of a Wails app.
type wailsTray struct {
	mu    sync.Mutex
	tray  *application.SystemTray
	shell *Shell
	last  *Tray
}

// TrayIcons are the tray's two pictures of the app icon's three notes:
// Template, black and alpha only, which macOS draws in the menu bar's own
// colour, and Colour, for the trays of Linux.
type TrayIcons struct {
	Template []byte
	Colour   []byte
}

// trayIcon picks the picture for goos and says whether it is a template
// image: on macOS the template, if there is one; elsewhere the colour icon.
func trayIcon(goos string, ic TrayIcons) (icon []byte, template bool) {
	if goos == "darwin" && len(ic.Template) > 0 {
		return ic.Template, true
	}
	return ic.Colour, false
}

// StartTray makes the tray icon; it is called before app.Run, so that
// SetTray never races with it, and Wails shows the icon once the app runs.
func (p *Wails) StartTray(s *Shell, icons TrayIcons) {
	t := p.App.SystemTray.New()
	if icon, template := trayIcon(runtime.GOOS, icons); template {
		t.SetTemplateIcon(icon)
	} else {
		t.SetIcon(icon)
	}
	t.SetTooltip("stickypane")
	p.tray = &wailsTray{tray: t, shell: s}
	p.tray.set(Tray{})
}

func (p *Wails) SetTray(t Tray) {
	if p.tray == nil {
		return
	}
	p.tray.set(t)
}

func (w *wailsTray) set(t Tray) {
	w.mu.Lock()
	defer w.mu.Unlock()
	if w.last != nil && sameTray(*w.last, t) {
		return
	}
	w.last = &t
	s := w.shell
	home, quit := trayWords(s.Prefs().Language)
	m := application.NewMenu()
	for _, it := range t.Recent {
		path := it.Path
		m.Add(it.Title).OnClick(func(*application.Context) { s.OpenProject(path) })
	}
	if len(t.Recent) > 0 {
		m.AddSeparator()
	}
	m.Add(home).OnClick(func(*application.Context) { s.ShowHome() })
	m.Add(quit).OnClick(func(*application.Context) { s.Quit() })
	w.tray.SetMenu(m)
	label := ""
	if t.Active > 0 {
		label = strconv.Itoa(t.Active)
	}
	w.tray.SetLabel(label)
}

func sameTray(a, b Tray) bool {
	if a.Active != b.Active || len(a.Recent) != len(b.Recent) {
		return false
	}
	for i := range a.Recent {
		if a.Recent[i] != b.Recent[i] {
			return false
		}
	}
	return true
}
