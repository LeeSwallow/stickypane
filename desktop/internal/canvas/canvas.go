// Package canvas holds where a project's notes sit on this computer: their
// places on the board window's free canvas and the frames of the notes torn
// off into their own windows (docs/desktop/03 §3.6.1, ADR-0017). It is the
// domain: values and rules, no I/O. The shell keeps them in
// ~/.stickypane/overlay.json under projects.<path>; a note's contents and
// sticky.json never change because of them.
package canvas

import (
	"math"
	"sort"
)

// The board window's layouts.
const (
	ModeCanvas = "canvas" // notes anywhere, by x, y, w, h (the default)
	ModeGrid   = "grid"   // the terminal's 12-column order and fractions
)

// The smallest a note gets on the canvas, the most it leans, and how far
// the stacking order may climb before it is counted again from 1.
const (
	MinW    = 160
	MinH    = 96
	MaxTilt = 6.0
	maxZ    = 10000
)

// Place is a note on the canvas, in the canvas's own pixels (before its
// zoom): its corner, its size, how much it leans in degrees, and its place
// in the stack (the higher, the nearer the top).
type Place struct {
	X    int     `json:"x"`
	Y    int     `json:"y"`
	W    int     `json:"w"`
	H    int     `json:"h"`
	Tilt float64 `json:"tilt"`
	Z    int     `json:"z"`
}

// Frame is a torn-off note's window on the screen, and whether it stays on
// top of other windows.
type Frame struct {
	ScreenX int  `json:"screenX"`
	ScreenY int  `json:"screenY"`
	W       int  `json:"w"`
	H       int  `json:"h"`
	Pinned  bool `json:"pinned"`
}

// Placed says whether the frame has been on the screen: a frame never
// placed opens where the shell chooses.
func (f Frame) Placed() bool { return f.W > 0 && f.H > 0 }

// Project is one project's layout on this computer.
type Project struct {
	// Mode is canvas or grid; "" is the default, canvas.
	Mode string `json:"mode,omitempty"`
	// NoSnap lets notes land on any pixel; by default they land on an
	// 8px grid.
	NoSnap bool             `json:"noSnap,omitempty"`
	Places map[string]Place `json:"places,omitempty"`
	Torn   map[string]Frame `json:"torn,omitempty"`
}

// ModeOf is the project's mode, the default when it has none or one that is
// not known.
func (p Project) ModeOf() string {
	if p.Mode == ModeGrid {
		return ModeGrid
	}
	return ModeCanvas
}

// ValidMode says whether m is a mode.
func ValidMode(m string) bool { return m == ModeCanvas || m == ModeGrid }

// Clone copies the project so that it can be changed or saved apart.
func (p Project) Clone() Project {
	c := p
	c.Places = make(map[string]Place, len(p.Places))
	for k, v := range p.Places {
		c.Places[k] = v
	}
	c.Torn = make(map[string]Frame, len(p.Torn))
	for k, v := range p.Torn {
		c.Torn[k] = v
	}
	return c
}

// Empty says whether the project holds nothing worth keeping.
func (p Project) Empty() bool {
	return p.Mode == "" && !p.NoSnap && len(p.Places) == 0 && len(p.Torn) == 0
}

// Clean makes a place one the canvas can draw: not above or left of the
// canvas, not smaller than MinW x MinH, leaning no more than MaxTilt.
func Clean(pl Place) Place {
	pl.X = max(pl.X, 0)
	pl.Y = max(pl.Y, 0)
	pl.W = max(pl.W, MinW)
	pl.H = max(pl.H, MinH)
	if math.IsNaN(pl.Tilt) {
		pl.Tilt = 0
	}
	pl.Tilt = math.Max(-MaxTilt, math.Min(MaxTilt, pl.Tilt))
	pl.Z = max(pl.Z, 0)
	return pl
}

// SetPlaces keeps the places given, cleaned. When the stack has climbed
// past its limit it is counted again from 1, in the same order.
func (p Project) SetPlaces(places map[string]Place) Project {
	c := p.Clone()
	top := 0
	for name, pl := range places {
		if name == "" {
			continue
		}
		pl = Clean(pl)
		c.Places[name] = pl
		top = max(top, pl.Z)
	}
	if top > maxZ {
		c.Places = restack(c.Places)
	}
	return c
}

// restack numbers the stack 1, 2, … keeping its order (ties by name).
func restack(places map[string]Place) map[string]Place {
	names := make([]string, 0, len(places))
	for n := range places {
		names = append(names, n)
	}
	sort.Slice(names, func(i, j int) bool { return below(places, names[i], names[j]) })
	out := make(map[string]Place, len(places))
	for i, n := range names {
		pl := places[n]
		pl.Z = i + 1
		out[n] = pl
	}
	return out
}

func below(places map[string]Place, a, b string) bool {
	if places[a].Z != places[b].Z {
		return places[a].Z < places[b].Z
	}
	return a < b
}

// Tear records that a note is torn off into a window with frame f.
func (p Project) Tear(note string, f Frame) Project {
	c := p.Clone()
	c.Torn[note] = f
	return c
}

// PutBack records that a torn-off note is back on the board.
func (p Project) PutBack(note string) Project {
	c := p.Clone()
	delete(c.Torn, note)
	return c
}

// Keep forgets the places and frames of notes that are not among names
// (the notes the board has now, pages included), and says which torn-off
// notes went with them.
func (p Project) Keep(names []string) (Project, []string) {
	have := make(map[string]bool, len(names))
	for _, n := range names {
		have[n] = true
	}
	c := p.Clone()
	for n := range c.Places {
		if !have[n] {
			delete(c.Places, n)
		}
	}
	var gone []string
	for n := range c.Torn {
		if !have[n] {
			delete(c.Torn, n)
			gone = append(gone, n)
		}
	}
	sort.Strings(gone)
	return c, gone
}

// Rename moves a note's place and frame to its new name.
func (p Project) Rename(from, to string) Project {
	if from == to || from == "" || to == "" {
		return p
	}
	c := p.Clone()
	if pl, ok := c.Places[from]; ok {
		delete(c.Places, from)
		c.Places[to] = pl
	}
	if f, ok := c.Torn[from]; ok {
		delete(c.Torn, from)
		c.Torn[to] = f
	}
	return c
}
