package chart

import (
	"strings"
	"testing"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parseSrc(s string) widget.Widget { return Kind.Parse(doc.Parse([]byte(s))) }

func draw(t *testing.T, src string, width int) []string {
	t.Helper()
	out, at := parseSrc(src).Draw(width, true)
	if at.Ok() {
		t.Errorf("a chart has no cursor, got %+v", at)
	}
	for _, line := range strings.Split(out, "\n") {
		if w := widget.Width(line); w > width {
			t.Errorf("line %q is %d cells wide, want at most %d", ansi.Strip(line), w, width)
		}
	}
	return strings.Split(ansi.Strip(out), "\n")
}

func TestBarsAreScaledToTheLargestValue(t *testing.T) {
	lines := draw(t, "Tests per package\n\napp: 40\nboard: 20\nstore: 10\nempty: 0\n", 25)
	want := []string{
		"Tests per package",
		"",
		"app   ████████████████ 40",
		"board ████████         20",
		"store ████             10",
		"empty                   0",
	}
	if strings.Join(lines, "\n") != strings.Join(want, "\n") {
		t.Errorf("Draw:\n%s\nwant:\n%s", strings.Join(lines, "\n"), strings.Join(want, "\n"))
	}
}

func TestBarsUsePartialBlocks(t *testing.T) {
	lines := draw(t, "a: 8\nb: 3\n", 12) // 8 cells of bar: b is 3 cells exactly
	if lines[0] != "a ████████ 8" || lines[1] != "b ███      3" {
		t.Errorf("Draw = %q", lines)
	}
	lines = draw(t, "a: 16\nb: 3\n", 13) // 8 cells of bar: b is 1.5 cells
	if lines[1] != "b █▌        3" {
		t.Errorf("half a cell should draw as a half block: %q", lines[1])
	}
}

func TestBarsAcceptListItemsAndFormattedNumbers(t *testing.T) {
	lines := draw(t, "- 입력: 1,200\n* 출력: 300.5\n  캐시 읽기: 2_400\n", 30)
	text := strings.Join(lines, "\n")
	for _, want := range []string{"입력", "출력", "캐시 읽기", "1,200", "300.5", "2_400"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
	if strings.Count(lines[2], "█") <= strings.Count(lines[0], "█") {
		t.Errorf("2,400 should have the longest bar:\n%s", text)
	}
}

func TestSparkShowsTheTrend(t *testing.T) {
	lines := draw(t, "---\ntype: chart\nview: spark\n---\nmon: 1\ntue: 3\nwed: 2\nthu: 8\n", 40)
	if lines[0] != "▁▃▂█" {
		t.Errorf("spark = %q", lines[0])
	}
	if !strings.Contains(lines[1], "mon") || !strings.Contains(lines[1], "thu") || !strings.Contains(lines[1], "8") {
		t.Errorf("the line under the spark should name the range and the peak: %q", lines[1])
	}
}

func TestSparkKeepsTheLatestValuesWhenNarrow(t *testing.T) {
	src := "---\nview: spark\n---\n"
	for i := 1; i <= 30; i++ {
		src += "d" + strings.Repeat("x", i) + ": " + strings.Repeat("1", 1) + "\n"
	}
	src += "last: 9\n"
	lines := draw(t, src, 10)
	if widget.Width(lines[0]) != 10 || !strings.HasSuffix(lines[0], "█") {
		t.Errorf("a narrow spark should show the latest values: %q", lines[0])
	}
}

func TestHeatDrawsACalendar(t *testing.T) {
	// 2026-09-28 is a Monday.
	src := "---\ntype: chart\nview: heat\n---\n" +
		"2026-09-28: 4\n2026-09-30: 1\n2026-10-02: 2\n2026-10-05: 0\n2026-10-06: 3\n"
	lines := draw(t, src, 30)
	want := []string{
		"     9 10",
		"Mon  █ ·",
		"       ▓",
		"Wed  ░",
		"",
		"Fri  ▒",
		"",
		"",
	}
	for i, w := range want {
		if strings.TrimRight(lines[i], " ") != strings.TrimRight(w, " ") {
			t.Fatalf("line %d = %q, want %q\n%s", i, lines[i], w, strings.Join(lines, "\n"))
		}
	}
	big := draw(t, "---\nview: heat\n---\n2026-09-28: 1,200,000\n2026-09-29: 34567.5\n", 60)
	if last := big[len(big)-1]; !strings.HasPrefix(last, "total 1,234,567.5 ") {
		t.Errorf("a large total should be grouped by thousands: %q", last)
	}
	if last := lines[len(lines)-1]; !strings.Contains(last, "10") || !strings.Contains(last, "2026-09-28") {
		t.Errorf("the last line should give the total and the peak day: %q", last)
	}
}

func TestHeatShadesByRankSoOneHugeDayDoesNotFlattenTheRest(t *testing.T) {
	src := "---\nview: heat\n---\n2026-09-28: 1\n2026-09-29: 2\n2026-09-30: 3\n2026-10-01: 1000\n"
	lines := draw(t, src, 30)
	var got []string
	for _, l := range lines[1:5] {
		got = append(got, strings.TrimSpace(strings.TrimLeft(l, "MonWed")))
	}
	if strings.Join(got, "") != "░▒▓█" {
		t.Errorf("shades = %q, want one of each level:\n%s", got, strings.Join(lines, "\n"))
	}
}

func TestHeatShowsTheLatestWeeksWhenNarrow(t *testing.T) {
	src := "---\nview: heat\n---\n2026-01-05: 1\n2026-10-05: 5\n"
	lines := draw(t, src, 16) // room for five week columns
	if !strings.Contains(lines[1], "█") {
		t.Errorf("the latest week should be visible: %q", lines)
	}
}

func TestHeatWithoutDatesFallsBackToBars(t *testing.T) {
	lines := draw(t, "---\nview: heat\n---\napp: 4\nstore: 2\n", 20)
	if !strings.HasPrefix(lines[0], "app") || !strings.Contains(lines[0], "█") {
		t.Errorf("labels that are not dates cannot make a calendar: %q", lines)
	}
}

func TestChartWithoutDataShowsItsText(t *testing.T) {
	lines := draw(t, "nothing to plot yet\nstill nothing\n", 50)
	text := strings.Join(lines, "\n")
	for _, want := range []string{`"label: number"`, "nothing to plot yet", "still nothing"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
}

func TestNarrowChartsDoNotBreak(t *testing.T) {
	src := "아주 긴 한글 라벨입니다: 12\n또 다른 긴 라벨: 7\n"
	for _, view := range []string{"bar", "spark", "heat"} {
		for _, width := range []int{1, 4, 9, 20} {
			draw(t, "---\nview: "+view+"\n---\n"+src+"2026-10-01: 3\n", width)
		}
	}
}

func TestSummaryAndSync(t *testing.T) {
	w := parseSrc("a: 1\nb: 2\n")
	if got := w.Summary(); got != "2 values" {
		t.Errorf("Summary = %q", got)
	}
	if got := parseSrc("a: 1\n").Summary(); got != "1 value" {
		t.Errorf("Summary = %q", got)
	}
	w = w.Sync(doc.Parse([]byte("a: 1\nb: 2\nc: 3\n")))
	if got := w.Summary(); got != "3 values" {
		t.Errorf("Summary after Sync = %q", got)
	}
	if _, res := w.Update("j"); res.Op != nil || res.Prompt != nil {
		t.Error("a chart takes no keys")
	}
}

func TestKind(t *testing.T) {
	if Kind.Name != "chart" || Kind.Keys != "" || Kind.Size(doc.Document{}) != widget.SizeHalf {
		t.Errorf("Kind = %+v", Kind)
	}
	if Kind.Icon == "" || widget.Width(Kind.Icon) != 1 || Kind.Blurb == "" || Kind.Example == "" {
		t.Errorf("Kind needs a one-cell icon, a blurb and an example: %+v", Kind)
	}
	if got := string(Kind.Template("Tokens")); got != "---\ntype: chart\ntitle: Tokens\n---\n" {
		t.Errorf("Template = %q", got)
	}
}
