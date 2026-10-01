package layout

import (
	"reflect"
	"testing"
)

func TestColumns(t *testing.T) {
	cases := []struct{ width, n, cw int }{
		{0, 1, 0}, {20, 1, 20}, {40, 1, 40}, {71, 1, 71},
		{72, 2, 36}, {80, 2, 40}, {100, 2, 50}, {120, 3, 40},
	}
	for _, c := range cases {
		if n, cw := Columns(c.width); n != c.n || cw != c.cw {
			t.Errorf("Columns(%d) = %d, %d; want %d, %d", c.width, n, cw, c.n, c.cw)
		}
	}
}

func TestFlow(t *testing.T) {
	cases := []struct {
		name  string
		width int
		items []Item
		want  []Rect
	}{
		{
			"one column stacks everything", 40,
			[]Item{{Height: 3}, {Height: 4}, {Height: 5, FullRow: true}},
			[]Rect{{0, 0, 40, 3}, {0, 3, 40, 4}, {0, 7, 40, 5}},
		},
		{
			"two columns fill the shorter one", 80,
			[]Item{{Height: 4, FullRow: true}, {Height: 3}, {Height: 5}, {Height: 3}, {Height: 2}},
			[]Rect{{0, 0, 80, 4}, {0, 4, 40, 3}, {40, 4, 40, 5}, {0, 7, 40, 3}, {40, 9, 40, 2}},
		},
		{
			"three columns", 120,
			[]Item{{Height: 3}, {Height: 3}, {Height: 3}, {Height: 2}},
			[]Rect{{0, 0, 40, 3}, {40, 0, 40, 3}, {80, 0, 40, 3}, {0, 3, 40, 2}},
		},
		{
			"a full row goes below every column", 80,
			[]Item{{Height: 2}, {Height: 6}, {Height: 3, FullRow: true}, {Height: 1}},
			[]Rect{{0, 0, 40, 2}, {40, 0, 40, 6}, {0, 6, 80, 3}, {0, 9, 40, 1}},
		},
		{"nothing to place", 80, nil, []Rect{}},
	}
	for _, c := range cases {
		if got := Flow(c.width, c.items); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s:\n got %v\nwant %v", c.name, got, c.want)
		}
	}
}

func TestNeighbor(t *testing.T) {
	// A is a full row; B and D share the left column, C and E the right one.
	rects := []Rect{{0, 0, 80, 4}, {0, 4, 40, 3}, {40, 4, 40, 5}, {0, 7, 40, 3}, {40, 9, 40, 2}}
	const a, b, c, d, e = 0, 1, 2, 3, 4
	cases := []struct {
		cur  int
		dir  Dir
		want int
	}{
		{a, Down, b}, {a, Up, a}, {a, Left, a},
		{b, Right, c}, {b, Down, d}, {b, Up, a}, {b, Left, b},
		{c, Left, b}, {c, Down, e}, {c, Right, c},
		{d, Right, e}, {d, Down, d}, {d, Up, b},
		{e, Up, c}, {e, Left, d},
	}
	for _, tc := range cases {
		if got := Neighbor(rects, tc.cur, tc.dir); got != tc.want {
			t.Errorf("Neighbor(%d, %v) = %d, want %d", tc.cur, tc.dir, got, tc.want)
		}
	}
	if got := Neighbor(rects, 9, Down); got != 9 {
		t.Errorf("an index outside the list should come back unchanged, got %d", got)
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
