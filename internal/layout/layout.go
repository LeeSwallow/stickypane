// Package layout places open notes on the screen. It is pure arithmetic: it
// knows widths and heights, not what a note contains.
package layout

import (
	"sort"
	"strings"
)

// Rect is a note's place on the screen, in terminal cells.
type Rect struct{ X, Y, W, H int }

// Item is the size of a note to place.
type Item struct{ W, H int }

// Shelf places items in order, left to right, and starts a new row when the
// next item does not fit. A row is as tall as its tallest item. The order of
// the items is never changed, so a note stays where its name puts it.
func Shelf(width int, items []Item) []Rect {
	rects := make([]Rect, len(items))
	x, y, rowH := 0, 0, 0
	for i, it := range items {
		if x > 0 && x+it.W > width {
			x, y, rowH = 0, y+rowH, 0
		}
		rects[i] = Rect{x, y, it.W, it.H}
		x += it.W
		rowH = max(rowH, it.H)
	}
	return rects
}

// Compose paints each box at its rect and returns the screen as lines. Every
// line of a box must be exactly its rect's width; gaps are filled with spaces.
func Compose(rects []Rect, boxes []string) []string {
	type segment struct {
		x, w int
		text string
	}
	height := 0
	for _, r := range rects {
		height = max(height, r.Y+r.H)
	}
	rows := make([][]segment, height)
	for i, r := range rects {
		lines := strings.Split(boxes[i], "\n")
		for dy := 0; dy < r.H; dy++ {
			text := strings.Repeat(" ", r.W)
			if dy < len(lines) {
				text = lines[dy]
			}
			rows[r.Y+dy] = append(rows[r.Y+dy], segment{r.X, r.W, text})
		}
	}
	out := make([]string, height)
	for y, segs := range rows {
		sort.Slice(segs, func(a, b int) bool { return segs[a].x < segs[b].x })
		var sb strings.Builder
		x := 0
		for _, s := range segs {
			if s.x > x {
				sb.WriteString(strings.Repeat(" ", s.x-x))
			}
			sb.WriteString(s.text)
			x = s.x + s.w
		}
		out[y] = sb.String()
	}
	return out
}
