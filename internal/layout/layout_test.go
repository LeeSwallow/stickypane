package layout

import (
	"reflect"
	"testing"
)

func TestShelf(t *testing.T) {
	cases := []struct {
		name  string
		width int
		items []Item
		want  []Rect
	}{
		{
			"a page takes its own row", 80,
			[]Item{{80, 5}, {80, 3}},
			[]Rect{{0, 0, 80, 5}, {0, 5, 80, 3}},
		},
		{
			"two halves share a row as tall as the taller one", 80,
			[]Item{{40, 6}, {40, 3}, {40, 2}},
			[]Rect{{0, 0, 40, 6}, {40, 0, 40, 3}, {0, 6, 40, 2}},
		},
		{
			"cards fill a row left to right", 108,
			[]Item{{36, 3}, {36, 4}, {36, 3}, {36, 2}},
			[]Rect{{0, 0, 36, 3}, {36, 0, 36, 4}, {72, 0, 36, 3}, {0, 4, 36, 2}},
		},
		{
			"mixed sizes keep their order", 80,
			[]Item{{40, 4}, {80, 5}, {40, 2}, {40, 3}},
			[]Rect{{0, 0, 40, 4}, {0, 4, 80, 5}, {0, 9, 40, 2}, {40, 9, 40, 3}},
		},
		{
			"an item wider than the screen still gets a row", 30,
			[]Item{{36, 3}, {36, 2}},
			[]Rect{{0, 0, 36, 3}, {0, 3, 36, 2}},
		},
		{"nothing to place", 80, nil, []Rect{}},
	}
	for _, c := range cases {
		if got := Shelf(c.width, c.items); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}

func TestCompose(t *testing.T) {
	rects := []Rect{{0, 0, 10, 2}, {0, 2, 5, 1}, {5, 2, 5, 2}}
	boxes := []string{"aaaaaaaaaa\nbbbbbbbbbb", "ccccc", "ddddd\neeeee"}
	want := []string{"aaaaaaaaaa", "bbbbbbbbbb", "cccccddddd", "     eeeee"}
	if got := Compose(rects, boxes); !reflect.DeepEqual(got, want) {
		t.Errorf("Compose = %q, want %q", got, want)
	}
	if got := Compose(nil, nil); len(got) != 0 {
		t.Errorf("Compose of nothing = %q", got)
	}
}
