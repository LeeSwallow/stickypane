package chart

import (
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/x/ansi"

	"github.com/LeeSwallow/stickypane/internal/doc"
	"github.com/LeeSwallow/stickypane/internal/widget"
)

func parseSrc(s string) widget.Widget { return Kind.Parse(doc.Parse([]byte(s)), widget.DefaultEnv()) }

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
	t.Parallel()
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
	t.Parallel()
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
	t.Parallel()
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

func TestSparkOnOneLineWhenThereIsNothingToPlot(t *testing.T) {
	t.Parallel()
	lines := draw(t, "---\ntype: chart\nview: spark\n---\nmon: 1\n", 60)
	if lines[0] != "▅" || len(lines) != 2 {
		t.Errorf("one value keeps the one-line form: %q", lines)
	}
	lines = draw(t, "---\nview: spark\n---\nmon: 1\ntue: 3\nwed: 2\nthu: 8\n", 20) // too narrow for a plot
	if lines[0] != "▁▃▂█" || !strings.Contains(strings.Join(lines[1:], " "), "mon") || !strings.Contains(strings.Join(lines[1:], " "), "peak 8") {
		t.Errorf("a narrow spark keeps the one-line form and names the range and the peak: %q", lines)
	}
}

const eightValues = "---\ntype: chart\nview: spark\n---\nmon: 10\ntue: 30\nwed: 20\nthu: 80\nfri: 50\nsat: 60\nsun: 40\nmon2: 95\n"

func TestSparkIsALineChartThatFillsTheWidth(t *testing.T) {
	t.Parallel()
	lines := draw(t, eightValues, 60)
	if len(lines) < 6+1 {
		t.Errorf("want at least 6 rows and a label line, got %d:\n%s", len(lines), strings.Join(lines, "\n"))
	}
	span := 0
	for _, l := range lines {
		span = max(span, widget.Width(strings.TrimRight(l, " ")))
	}
	if span < 50 {
		t.Errorf("the chart spans %d cells, want at least 50:\n%s", span, strings.Join(lines, "\n"))
	}
	text := strings.Join(lines, "\n")
	if !strings.HasPrefix(lines[0], "95┤") || !strings.Contains(lines[5], "10┤") {
		t.Errorf("the axis names the largest and the smallest value:\n%s", text)
	}
	if !strings.HasPrefix(strings.TrimSpace(lines[len(lines)-1]), "mon") || !strings.HasSuffix(lines[len(lines)-1], "mon2") {
		t.Errorf("the first and last labels sit under the chart:\n%s", text)
	}
	if strings.Count(text, "●") != 1 || !strings.HasSuffix(strings.TrimRight(strings.Split(text, "\n")[0], " "), "● 95") {
		t.Errorf("the latest value is marked and written at the end:\n%s", text)
	}
	for _, w := range []int{7, 1} {
		draw(t, eightValues, w)
	}
}

func TestSparkIsTallerWhenZoomed(t *testing.T) {
	t.Parallel()
	w := parseSrc(eightValues).(widget.Roomy)
	pane, _ := w.DrawIn(60, 0, false)
	zoom, _ := w.DrawIn(150, 30, true)
	paneRows, zoomRows := strings.Count(pane, "\n")+1, strings.Count(zoom, "\n")+1
	if zoomRows <= paneRows || zoomRows > 30 {
		t.Errorf("zoomed rows = %d, pane rows = %d", zoomRows, paneRows)
	}
	for _, l := range strings.Split(zoom, "\n") {
		if widget.Width(l) > 150 {
			t.Errorf("zoomed line is %d cells wide", widget.Width(l))
		}
	}
	if widest(zoom) <= widest(pane) {
		t.Error("a zoomed chart should be wider than the pane's")
	}
}

func widest(s string) int {
	n := 0
	for _, l := range strings.Split(ansi.Strip(s), "\n") {
		n = max(n, widget.Width(strings.TrimRight(l, " ")))
	}
	return n
}

func ninetyDays() string {
	src := "---\ntype: chart\nview: heat\n---\n"
	d := time.Date(2026, 7, 1, 0, 0, 0, 0, time.UTC)
	for i := 0; i < 90; i++ {
		src += d.AddDate(0, 0, i).Format("2006-01-02") + ": " + strconv.Itoa(i%9) + "\n"
	}
	return src
}

func TestHeatFillsTheWidthWithTouchingShades(t *testing.T) {
	t.Parallel()
	lines := draw(t, ninetyDays(), 80)
	if got := widest(strings.Join(lines, "\n")); got < 60 {
		t.Errorf("the calendar spans %d cells, want at least 60:\n%s", got, strings.Join(lines, "\n"))
	}
	if !strings.Contains(lines[0], "7") || !strings.Contains(lines[0], "8") || !strings.Contains(lines[0], "9") {
		t.Errorf("month labels are missing: %q", lines[0])
	}
	// Two days a week apart are next to each other, with no gap: day 0 is
	// a Wednesday (value 0), day 7 the next Wednesday (value 7, the full
	// shade by rank), so the row has two blocks back to back.
	row := lines[3]
	if !strings.Contains(row, "██") && !strings.Contains(row, "▓▓") {
		t.Errorf("weeks should be blocks that touch: %q", row)
	}
	if !strings.Contains(strings.Join(lines, "\n"), "less ░▒▓█ more") {
		t.Errorf("the scale is missing:\n%s", strings.Join(lines, "\n"))
	}
}

func TestHeatWeekWidthFollowsTheRoom(t *testing.T) {
	t.Parallel()
	// 2026-09-28 is a Monday: three weeks of data.
	src := "---\nview: heat\n---\n2026-09-28: 4\n2026-10-05: 4\n2026-10-12: 4\n"
	for width, cells := range map[int]int{80: 4, 40: 4, 16: 1} {
		lines := draw(t, src, width)
		if got := strings.Count(lines[1], "█"); got != 3*cells {
			t.Errorf("width %d: a week is %d cells wide, want %d: %q", width, got/3, cells, lines[1])
		}
	}
}

func TestHeatIsBiggerWhenZoomed(t *testing.T) {
	t.Parallel()
	w := parseSrc(ninetyDays())
	pane, _ := w.Draw(60, false)
	zoom, _ := w.(widget.Roomy).DrawIn(200, 40, true)
	if widest(zoom) <= widest(pane) {
		t.Errorf("zoomed %d, pane %d", widest(zoom), widest(pane))
	}
}

func TestSparkKeepsTheLatestValuesWhenNarrow(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
	if last := big[len(big)-2]; !strings.HasPrefix(last, "total 1,234,567.5 ") {
		t.Errorf("a large total should be grouped by thousands: %q", last)
	}
	if last := lines[len(lines)-1]; !strings.Contains(last, "10") || !strings.Contains(last, "2026-09-28") {
		t.Errorf("the last line should give the total and the peak day: %q", last)
	}
}

func TestHeatShadesByRankSoOneHugeDayDoesNotFlattenTheRest(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
	src := "---\nview: heat\n---\n2026-01-05: 1\n2026-10-05: 5\n"
	lines := draw(t, src, 16) // room for five week columns
	if !strings.Contains(lines[1], "█") {
		t.Errorf("the latest week should be visible: %q", lines)
	}
}

func TestHeatWithoutDatesFallsBackToBars(t *testing.T) {
	t.Parallel()
	lines := draw(t, "---\nview: heat\n---\napp: 4\nstore: 2\n", 20)
	if !strings.HasPrefix(lines[0], "app") || !strings.Contains(lines[0], "█") {
		t.Errorf("labels that are not dates cannot make a calendar: %q", lines)
	}
}

func TestChartWithoutDataShowsItsText(t *testing.T) {
	t.Parallel()
	lines := draw(t, "nothing to plot yet\nstill nothing\n", 50)
	text := strings.Join(lines, "\n")
	for _, want := range []string{`"label: number"`, "nothing to plot yet", "still nothing"} {
		if !strings.Contains(text, want) {
			t.Errorf("Draw should contain %q:\n%s", want, text)
		}
	}
}

func TestNarrowChartsDoNotBreak(t *testing.T) {
	t.Parallel()
	src := "아주 긴 한글 라벨입니다: 12\n또 다른 긴 라벨: 7\n"
	for _, view := range []string{"bar", "spark", "heat"} {
		for _, width := range []int{1, 4, 9, 20} {
			draw(t, "---\nview: "+view+"\n---\n"+src+"2026-10-01: 3\n", width)
		}
	}
}

func TestSparkSurvivesExtremeValues(t *testing.T) {
	t.Parallel()
	huge := "1" + strings.Repeat("0", 308)
	lines := draw(t, "---\nview: spark\n---\na: "+huge+"\nb: -"+huge+"\nc: 0\n", 40)
	if len(lines) < 2 {
		t.Errorf("spark = %q", lines)
	}
}

func TestTextThatOnlyLooksLikeDataStaysText(t *testing.T) {
	t.Parallel()
	lines := draw(t, "Updated at 12:30\nSee http://localhost:8080\napp: 4\n", 40)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "Updated at 12:30") || !strings.Contains(text, "http://localhost:8080") {
		t.Errorf("a time and a URL are text, not data:\n%s", text)
	}
	if got := parseSrc("Updated at 12:30\napp: 4\n").Summary(); got != "1 value" {
		t.Errorf("Summary = %q", got)
	}
}

func TestHeatShowsValuesThatAreNotDays(t *testing.T) {
	t.Parallel()
	lines := draw(t, "---\nview: heat\n---\n2026-09-28: 4\nbudget: 100\n", 40)
	text := strings.Join(lines, "\n")
	if !strings.Contains(text, "budget: 100") {
		t.Errorf("a value without a date must not vanish from a calendar:\n%s", text)
	}
	if got := parseSrc("---\nview: heat\n---\n2026-09-28: 4\nbudget: 100\n").Summary(); got != "1 value" {
		t.Errorf("Summary should count what is drawn: %q", got)
	}
}

func TestSummaryAndSync(t *testing.T) {
	t.Parallel()
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
	t.Parallel()
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
