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

func TestRowsFillTheWidth(t *testing.T) {
	cases := []struct {
		name   string
		width  int
		widths []int
		want   []Cell
	}{
		{"a page takes its own row", 80, []int{80, 80}, []Cell{{0, 0, 80}, {1, 0, 80}}},
		{"two halves share a row", 80, []int{40, 40, 80}, []Cell{{0, 0, 40}, {0, 40, 40}, {1, 0, 80}}},
		{"a row with room left is stretched", 80, []int{40, 80}, []Cell{{0, 0, 80}, {1, 0, 80}}},
		{"the stretch is shared by width", 90, []int{20, 40}, []Cell{{0, 0, 30}, {0, 30, 60}}},
		{"the last item takes the rounding", 100, []int{33, 33, 33}, []Cell{{0, 0, 33}, {0, 33, 33}, {0, 66, 34}}},
		{"an item wider than the screen is narrowed", 30, []int{80}, []Cell{{0, 0, 30}}},
		{"nothing", 80, nil, []Cell{}},
	}
	for _, c := range cases {
		if got := Rows(c.width, c.widths); !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Rows(%d, %v) = %v, want %v", c.name, c.width, c.widths, got, c.want)
		}
	}
}

func TestStackFillsEveryScreen(t *testing.T) {
	cases := []struct {
		name   string
		height int
		need   []int
		floor  int
		want   []Slot
	}{
		{"rows that fit share what is left over", 20, []int{5, 5}, 6, []Slot{{0, 0, 10}, {0, 10, 10}}},
		{"a short row keeps its height and the long one takes the rest", 20, []int{30, 4}, 6, []Slot{{0, 0, 16}, {0, 16, 4}}},
		{"long rows share the screen equally", 20, []int{30, 30}, 6, []Slot{{0, 0, 10}, {0, 10, 10}}},
		{"rows that cannot have the floor go to the next screen", 20, []int{30, 30, 30, 30}, 6,
			[]Slot{{0, 0, 6}, {0, 6, 7}, {0, 13, 7}, {1, 0, 20}}},
		{"short rows pack more to a screen", 20, []int{4, 4, 4, 4, 30}, 6, []Slot{{0, 0, 5}, {0, 5, 5}, {0, 10, 5}, {0, 15, 5}, {1, 0, 20}}},
		{"a screen shorter than the floor holds one row", 4, []int{30, 30}, 6, []Slot{{0, 0, 4}, {1, 0, 4}}},
		{"no height at all", 0, []int{5}, 6, []Slot{{0, 0, 0}}},
		{"nothing", 20, nil, 6, []Slot{}},
	}
	for _, c := range cases {
		got := Stack(c.height, c.need, c.floor)
		if !reflect.DeepEqual(got, c.want) {
			t.Errorf("%s: Stack(%d, %v, %d) = %v, want %v", c.name, c.height, c.need, c.floor, got, c.want)
		}
		used := map[int]int{}
		for _, s := range got {
			used[s.Screen] += s.H
		}
		for screen, h := range used {
			if c.height > 0 && h != c.height {
				t.Errorf("%s: screen %d uses %d of %d lines", c.name, screen, h, c.height)
			}
		}
	}
}
