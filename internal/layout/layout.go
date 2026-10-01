// Package layout places notes on the board. It is pure arithmetic: it knows
// widths and heights, not what a note contains.
package layout

import (
	"sort"
	"strings"
)

// MinWidth is the narrowest a note is drawn when the pane allows it.
const MinWidth = 36

// Rect is a note's place on the board, in terminal cells.
type Rect struct{ X, Y, W, H int }

// Item is what Flow needs to know about a note.
type Item struct {
	Height  int
	FullRow bool // spans every column
}

// Columns returns how many columns fit and how wide each one is.
func Columns(width int) (n, colWidth int) {
	n = max(width/MinWidth, 1)
	return n, width / n
}

// Flow places items in order. A regular item goes to the shortest column
// (the leftmost on a tie); a full-row item goes below every column.
func Flow(width int, items []Item) []Rect {
	n, cw := Columns(width)
	bottoms := make([]int, n)
	rects := make([]Rect, len(items))
	for i, it := range items {
		if it.FullRow {
			y := 0
			for _, b := range bottoms {
				y = max(y, b)
			}
			rects[i] = Rect{0, y, n * cw, it.Height}
			for c := range bottoms {
				bottoms[c] = y + it.Height
			}
			continue
		}
		col := 0
		for c, b := range bottoms {
			if b < bottoms[col] {
				col = c
			}
		}
		rects[i] = Rect{col * cw, bottoms[col], cw, it.Height}
		bottoms[col] += it.Height
	}
	return rects
}

// Dir is a direction for moving the focus.
type Dir int

// Directions.
const (
	Up Dir = iota
	Down
	Left
	Right
)

// overlap is the length shared by the ranges [a0, a1) and [b0, b1).
func overlap(a0, a1, b0, b1 int) int { return min(a1, b1) - max(a0, b0) }

func abs(n int) int {
	if n < 0 {
		return -n
	}
	return n
}

// Neighbor returns the index of the nearest rect in direction d that shares
// some extent with the current one, or cur when there is none.
func Neighbor(rects []Rect, cur int, d Dir) int {
	if cur < 0 || cur >= len(rects) {
		return cur
	}
	c := rects[cur]
	best, bestDist, bestTie := cur, 0, 0
	for i, r := range rects {
		if i == cur {
			continue
		}
		sameCols := overlap(c.X, c.X+c.W, r.X, r.X+r.W) > 0
		sameRows := overlap(c.Y, c.Y+c.H, r.Y, r.Y+r.H) > 0
		var dist, tie int
		switch {
		case d == Down && sameCols && r.Y >= c.Y+c.H:
			dist, tie = r.Y-(c.Y+c.H), r.X
		case d == Up && sameCols && r.Y+r.H <= c.Y:
			dist, tie = c.Y-(r.Y+r.H), r.X
		case d == Right && sameRows && r.X >= c.X+c.W:
			dist, tie = r.X-(c.X+c.W), abs(r.Y-c.Y)
		case d == Left && sameRows && r.X+r.W <= c.X:
			dist, tie = c.X-(r.X+r.W), abs(r.Y-c.Y)
		default:
			continue
		}
		if best == cur || dist < bestDist || (dist == bestDist && tie < bestTie) {
			best, bestDist, bestTie = i, dist, tie
		}
	}
	return best
}

// Compose paints each box at its rect and returns the board as lines. Every
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
