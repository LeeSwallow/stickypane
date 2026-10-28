package widget

import "testing"

func TestSizesAreFractionsAndTheOldNamesStayReadable(t *testing.T) {
	cases := []struct {
		in, want string
		ok       bool
	}{
		{"1/4", "1/4", true}, {"1/3", "1/3", true}, {"1/2", "1/2", true},
		{"2/3", "2/3", true}, {"3/4", "3/4", true}, {"1", "1", true},
		{" 1/3 ", "1/3", true},
		{"half", "1/2", true}, {"Page", "1", true}, {"card", "card", true}, {"CARD", "card", true},
		{"2/5", "", false}, {"huge", "", false}, {"", "", false}, {"1/0", "", false},
	}
	for _, c := range cases {
		if got, ok := NormalSize(c.in); got != c.want || ok != c.ok {
			t.Errorf("NormalSize(%q) = %q, %v; want %q, %v", c.in, got, ok, c.want, c.ok)
		}
	}
}

func TestASizeIsAShareOfTheScreen(t *testing.T) {
	cases := []struct {
		size  string
		width int
		want  int
	}{
		{"1/4", 80, 20}, {"1/3", 80, 26}, {"1/2", 80, 40}, {"2/3", 80, 53}, {"3/4", 80, 60}, {"1", 80, 80},
		{"half", 80, 40}, {"page", 80, 80},
		// card is the narrowest step a 36-cell card fits in.
		{"card", 80, 40}, {"card", 120, 40}, {"card", 160, 40}, {"card", 50, 37},
		{"nonsense", 80, 80},
		// A fraction narrower than MinWidth takes the next one that is not.
		{"1/4", 60, 20}, {"1/2", 36, 24}, {"1/4", 2, 2},
	}
	for _, c := range cases {
		if got := SizeCells(c.size, c.width); got != c.want {
			t.Errorf("SizeCells(%q, %d) = %d, want %d", c.size, c.width, got, c.want)
		}
	}
}

func TestPlusAndMinusStepThroughTheFractions(t *testing.T) {
	cases := []struct {
		size  string
		delta int
		want  string
	}{
		{"1/4", 1, "1/3"}, {"1/3", 1, "1/2"}, {"1/2", 1, "2/3"}, {"2/3", 1, "3/4"}, {"3/4", 1, "1"}, {"1", 1, "1"},
		{"1", -1, "3/4"}, {"1/4", -1, "1/4"},
		{"half", 1, "2/3"}, {"page", -1, "3/4"},
		{"card", 1, "2/3"}, {"card", -1, "1/3"}, // card is 1/2 on 80 cells
	}
	for _, c := range cases {
		if got := StepSize(c.size, 80, c.delta); got != c.want {
			t.Errorf("StepSize(%q, 80, %d) = %q, want %q", c.size, c.delta, got, c.want)
		}
	}
}

func TestADraggedWidthChangesOnlyPastTheMiddleOfTwoSteps(t *testing.T) {
	// On 80 cells the steps are 20 26 40 53 60 80.
	cases := []struct {
		from  string
		cells int
		want  string
	}{
		{"1/2", 40, "1/2"}, {"1/2", 39, "1/2"}, {"1/2", 41, "1/2"},
		{"1/2", 34, "1/2"}, {"1/2", 33, "1/2"}, {"1/2", 32, "1/3"}, // the middle of 26 and 40 is 33
		{"1/2", 47, "2/3"}, {"1/2", 46, "1/2"},
		{"1/2", 21, "1/4"}, {"1/2", 200, "1"}, {"1/2", 1, "1/4"},
		{"card", 41, "1/2"}, {"page", 79, "1"}, {"1", 70, "1"}, {"1", 69, "3/4"},
	}
	for _, c := range cases {
		if got := SnapSize(c.from, c.cells, 80); got != c.want {
			t.Errorf("SnapSize(%q, %d, 80) = %q, want %q", c.from, c.cells, got, c.want)
		}
	}
}
