package layout

import (
	"reflect"
	"testing"
)

func TestGridDividesTheScreenIntoColumns(t *testing.T) {
	cases := []struct {
		name  string
		width int
		n     int
		spans []int
		sized []bool
		want  []Cell
	}{
		{
			"notes without a size take a column each", 120, 3,
			[]int{1, 1, 1, 1, 1, 1}, nil,
			[]Cell{{0, 0, 40}, {0, 40, 40}, {0, 80, 40}, {1, 0, 40}, {1, 40, 40}, {1, 80, 40}},
		},
		{
			"a row that is not full is shared equally", 120, 3,
			[]int{1, 1, 1, 1, 1}, nil,
			[]Cell{{0, 0, 40}, {0, 40, 40}, {0, 80, 40}, {1, 0, 60}, {1, 60, 60}},
		},
		{
			"a span counts only for a note with a size of its own", 120, 3,
			[]int{2, 3, 1}, []bool{false, false, false},
			[]Cell{{0, 0, 40}, {0, 40, 40}, {0, 80, 40}},
		},
		{
			"a sized note keeps its columns and the others share the rest", 120, 4,
			[]int{1, 1, 1}, []bool{true, false, false},
			[]Cell{{0, 0, 30}, {0, 30, 45}, {0, 75, 45}},
		},
		{
			"a note that does not fit starts the next row", 120, 3,
			[]int{1, 2, 3, 1}, []bool{false, true, true, false},
			[]Cell{{0, 0, 40}, {0, 40, 80}, {1, 0, 120}, {2, 0, 120}},
		},
		{
			"sized notes that do not fill a row leave the rest empty", 120, 3,
			[]int{2, 2}, []bool{true, true},
			[]Cell{{0, 0, 80}, {1, 0, 80}},
		},
		{
			"rounding is shared", 100, 3,
			[]int{1, 1, 1}, []bool{true, true, true},
			[]Cell{{0, 0, 34}, {0, 34, 33}, {0, 67, 33}},
		},
		{"one column", 80, 1, []int{1, 3}, []bool{false, true}, []Cell{{0, 0, 80}, {1, 0, 80}}},
		{"nothing to place", 80, 3, nil, nil, []Cell{}},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			if got := Grid(c.width, c.n, c.spans, c.sized); !reflect.DeepEqual(got, c.want) {
				t.Errorf("Grid = %v, want %v", got, c.want)
			}
		})
	}
}

func TestBesideIsTheNoteAboveOrBelow(t *testing.T) {
	grid := Grid(120, 3, []int{1, 1, 1, 1, 1}, nil) // a b c / d e
	cases := []struct{ i, delta, want int }{
		{1, 1, 4}, {2, 1, 4}, {0, 1, 3}, {4, -1, 1}, {3, -1, 0},
		{0, -1, -1}, {3, 1, -1}, {9, 1, -1},
	}
	for _, c := range cases {
		if got := Beside(grid, c.i, c.delta); got != c.want {
			t.Errorf("Beside(%d, %d) = %d, want %d", c.i, c.delta, got, c.want)
		}
	}
}
