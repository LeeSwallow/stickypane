package widget

import (
	"encoding/json"
	"os"
	"strconv"
	"testing"
)

// The table is shared with the desktop app's tests, so the terminal and
// the app give a note the same columns.
func TestASizeTakesItsShareOfTheColumns(t *testing.T) {
	t.Parallel()
	raw, err := os.ReadFile("testdata/columns.json")
	if err != nil {
		t.Fatal(err)
	}
	var table struct {
		Spans map[string][]int    `json:"spans"`
		Sizes map[string][]string `json:"sizes"`
	}
	if err := json.Unmarshal(raw, &table); err != nil {
		t.Fatal(err)
	}
	if len(table.Spans) != len(Sizes)+2 {
		t.Errorf("the table has a row for every size, a card and none: %d rows", len(table.Spans))
	}
	for size, spans := range table.Spans {
		for i, want := range spans {
			if got := ColumnSpan(size, i+1); got != want {
				t.Errorf("ColumnSpan(%q, %d) = %d, want %d", size, i+1, got, want)
			}
		}
	}
	for ns, sizes := range table.Sizes {
		n, _ := strconv.Atoi(ns)
		for k, want := range sizes {
			if got := ColumnSize(k+1, n); got != want {
				t.Errorf("ColumnSize(%d, %d) = %q, want %q", k+1, n, got, want)
			}
			if _, ok := NormalSize(want); !ok {
				t.Errorf("%q is not a size", want)
			}
			if ColumnSpan(want, n) != k+1 {
				t.Errorf("%q reads back as %d of %d columns", want, ColumnSpan(want, n), n)
			}
		}
	}
}

func TestAStepIsOneColumn(t *testing.T) {
	t.Parallel()
	cases := []struct {
		size     string
		n, delta int
		want     string
	}{
		{"1/3", 3, 1, "2/3"}, {"2/3", 3, 1, "1"}, {"1", 3, 1, "1"},
		{"1/3", 3, -1, "1/3"}, {"1", 4, -1, "3/4"}, {"card", 4, 1, "1/2"},
		{"1/2", 2, 1, "1"}, {"1", 1, -1, "1"},
	}
	for _, c := range cases {
		if got := StepColumns(c.size, c.n, c.delta); got != c.want {
			t.Errorf("StepColumns(%q, %d, %d) = %q, want %q", c.size, c.n, c.delta, got, c.want)
		}
	}
}

func TestADraggedBorderLandsOnTheNearestColumn(t *testing.T) {
	t.Parallel()
	cases := []struct {
		cells, width, n int
		want            string
	}{
		{40, 120, 3, "1/3"}, {59, 120, 3, "1/3"}, {61, 120, 3, "2/3"},
		{110, 120, 3, "1"}, {3, 120, 3, "1/3"}, {500, 120, 4, "1"},
	}
	for _, c := range cases {
		if got := SnapColumns(c.cells, c.width, c.n); got != c.want {
			t.Errorf("SnapColumns(%d, %d, %d) = %q, want %q", c.cells, c.width, c.n, got, c.want)
		}
	}
}

// Fewer columns only when a column would be too narrow to read.
func TestASmallScreenDrawsFewerColumns(t *testing.T) {
	t.Parallel()
	cases := []struct{ n, width, want int }{
		{3, 160, 3}, {4, 100, 4}, {4, 79, 3}, {3, 40, 2}, {3, 10, 1}, {1, 200, 1},
	}
	for _, c := range cases {
		if got := FitColumns(c.n, c.width); got != c.want {
			t.Errorf("FitColumns(%d, %d) = %d, want %d", c.n, c.width, got, c.want)
		}
	}
}
