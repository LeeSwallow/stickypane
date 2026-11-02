package prefs

import (
	"slices"
	"strings"
	"testing"
)

// A board has three columns unless set otherwise; auto, the width-based
// rows of before, is still read but not offered.
func TestColumnsAreANumberOrAuto(t *testing.T) {
	st := board(t)
	if Columns(st.Settings()) != 3 {
		t.Errorf("columns by default = %d, want 3", Columns(st.Settings()))
	}
	for _, c := range []struct {
		value string
		want  int
	}{{"1", 1}, {"4", 4}, {"AUTO", 0}, {"3", 3}} {
		if err := Set(st, "columns", c.value); err != nil {
			t.Fatal(err)
		}
		if got := Columns(st.Settings()); got != c.want {
			t.Errorf("columns %s = %d, want %d", c.value, got, c.want)
		}
	}
	if err := Set(st, "columns", "5"); err == nil || !strings.Contains(err.Error(), "3, 4, 1, 2") {
		t.Errorf("5 columns are refused with the values: %v", err)
	}
	if slices.Contains(Lookup("columns").Values(), ColumnsAuto) {
		t.Error("auto is not offered")
	}
}
