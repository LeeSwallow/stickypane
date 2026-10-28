package canvas

import (
	"encoding/json"
	"math"
	"reflect"
	"testing"
)

func TestModeIsCanvasUnlessGrid(t *testing.T) {
	for in, want := range map[string]string{"": ModeCanvas, "canvas": ModeCanvas, "grid": ModeGrid, "tiles": ModeCanvas} {
		if got := (Project{Mode: in}).ModeOf(); got != want {
			t.Errorf("ModeOf(%q) = %q, want %q", in, got, want)
		}
	}
	if !ValidMode("grid") || !ValidMode("canvas") || ValidMode("") || ValidMode("tiles") {
		t.Error("ValidMode accepts only canvas and grid")
	}
}

func TestCleanKeepsANoteDrawable(t *testing.T) {
	got := Clean(Place{X: -40, Y: -1, W: 10, H: 0, Tilt: 30, Z: -3})
	want := Place{X: 0, Y: 0, W: MinW, H: MinH, Tilt: MaxTilt, Z: 0}
	if got != want {
		t.Errorf("Clean = %+v, want %+v", got, want)
	}
	if got := Clean(Place{W: 300, H: 200, Tilt: math.NaN()}); got.Tilt != 0 {
		t.Errorf("a NaN tilt is %v, want 0", got.Tilt)
	}
	if got := Clean(Place{W: 300, H: 200, Tilt: -9}); got.Tilt != -MaxTilt {
		t.Errorf("tilt -9 is %v, want %v", got.Tilt, -MaxTilt)
	}
}

func TestSetPlacesDoesNotChangeTheOriginal(t *testing.T) {
	p := Project{Places: map[string]Place{"a.md": {X: 1, W: 200, H: 100}}}
	q := p.SetPlaces(map[string]Place{"b.md": {X: 5, Y: 6, W: 300, H: 200, Z: 2}, "": {W: 999}})
	if len(p.Places) != 1 {
		t.Errorf("the original changed: %+v", p.Places)
	}
	if q.Places["b.md"] != (Place{X: 5, Y: 6, W: 300, H: 200, Z: 2}) || q.Places["a.md"].X != 1 {
		t.Errorf("places = %+v", q.Places)
	}
	if _, ok := q.Places[""]; ok {
		t.Error("a place without a note name was kept")
	}
}

func TestTheStackIsCountedAgainWhenItClimbsTooHigh(t *testing.T) {
	p := Project{Places: map[string]Place{"a.md": {W: 200, H: 100, Z: 7}, "b.md": {W: 200, H: 100, Z: 3}}}
	q := p.SetPlaces(map[string]Place{"c.md": {W: 200, H: 100, Z: maxZ + 1}})
	got := map[string]int{}
	for n, pl := range q.Places {
		got[n] = pl.Z
	}
	if want := map[string]int{"b.md": 1, "a.md": 2, "c.md": 3}; !reflect.DeepEqual(got, want) {
		t.Errorf("z = %v, want %v", got, want)
	}
}

func TestTearAndPutBack(t *testing.T) {
	f := Frame{ScreenX: 10, ScreenY: 20, W: 360, H: 400, Pinned: true}
	p := Project{}.Tear("a.md", f)
	if p.Torn["a.md"] != f || !f.Placed() {
		t.Errorf("torn = %+v", p.Torn)
	}
	if (Frame{}).Placed() {
		t.Error("a zero frame says it was placed")
	}
	q := p.PutBack("a.md")
	if len(q.Torn) != 0 || len(p.Torn) != 1 {
		t.Errorf("put back: %+v (original %+v)", q.Torn, p.Torn)
	}
}

func TestKeepForgetsNotesThatAreGone(t *testing.T) {
	p := Project{
		Places: map[string]Place{"a.md": {W: 200}, "gone.md": {W: 200}, "book/1.md": {W: 200}},
		Torn:   map[string]Frame{"a.md": {W: 1}, "z.md": {W: 1}, "gone.md": {W: 1}},
	}
	q, gone := p.Keep([]string{"a.md", "book/", "book/1.md"})
	if !reflect.DeepEqual(gone, []string{"gone.md", "z.md"}) {
		t.Errorf("gone = %v", gone)
	}
	if len(q.Places) != 2 || len(q.Torn) != 1 {
		t.Errorf("kept %+v", q)
	}
	if len(p.Places) != 3 {
		t.Error("Keep changed the original")
	}
}

func TestRenameMovesThePlaceAndTheFrame(t *testing.T) {
	p := Project{Places: map[string]Place{"a.md": {X: 3, W: 200}}, Torn: map[string]Frame{"a.md": {W: 5}}}
	q := p.Rename("a.md", "b.md")
	if _, ok := q.Places["a.md"]; ok || q.Places["b.md"].X != 3 || q.Torn["b.md"].W != 5 {
		t.Errorf("renamed = %+v", q)
	}
	if r := p.Rename("a.md", ""); !reflect.DeepEqual(r, p) {
		t.Error("a rename to nothing changed something")
	}
}

func TestTheJSONIsTheSpecs(t *testing.T) {
	p := Project{Mode: "canvas", Places: map[string]Place{"a.md": {X: 1, Y: 2, W: 3, H: 4, Tilt: 1.5, Z: 6}}, Torn: map[string]Frame{"b.md": {ScreenX: 7, ScreenY: 8, W: 9, H: 10, Pinned: true}}}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"mode":"canvas","places":{"a.md":{"x":1,"y":2,"w":3,"h":4,"tilt":1.5,"z":6}},"torn":{"b.md":{"screenX":7,"screenY":8,"w":9,"h":10,"pinned":true}}}`
	if string(b) != want {
		t.Errorf("json = %s\nwant   %s", b, want)
	}
	if !(Project{}).Empty() || p.Empty() {
		t.Error("Empty is wrong")
	}
}
