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

// Cell is an item's place across the screen: its row, where it starts and
// how wide it is.
type Cell struct{ Row, X, W int }

// Rows puts items of the given widths side by side, in order, starting a
// new row when the next one does not fit. Each row is then stretched to the
// full width, every item growing in proportion to its width, so no part of
// the screen is left empty.
func Rows(width int, widths []int) []Cell {
	cells := make([]Cell, 0, len(widths))
	stretch := func(from int) {
		row := cells[from:]
		sum := 0
		for _, c := range row {
			sum += c.W
		}
		if len(row) == 0 || sum <= 0 {
			return
		}
		x := 0
		for i := range row {
			row[i].X = x
			row[i].W = row[i].W * width / sum
			if i == len(row)-1 {
				row[i].W = width - x
			}
			x += row[i].W
		}
	}
	row, start, used := 0, 0, 0
	for _, w := range widths {
		w = max(min(w, width), 1)
		if used > 0 && used+w > width {
			stretch(start)
			row, start, used = row+1, len(cells), 0
		}
		cells = append(cells, Cell{Row: row, W: w})
		used += w
	}
	stretch(start)
	return cells
}

// Slot is a row's place down the screen: which screen it is on, where it
// starts and how tall it is.
type Slot struct{ Screen, Y, H int }

// Stack puts rows on screens of the given height, in order. need is the
// height each row asks for. A screen takes rows for as long as every one of
// them can have at least floor lines, or all it needs if that is less; the
// rest go to the next screen. A screen's height is then shared out: a row
// that needs less than an equal share gets what it needs and the others
// split the rest, and whatever is left when every row has what it needs is
// split equally. Every screen is filled exactly.
func Stack(height int, need []int, floor int) []Slot {
	slots := make([]Slot, len(need))
	if height <= 0 {
		return slots
	}
	share := func(from, to, screen int) {
		rows := need[from:to]
		got := make([]int, len(rows))
		left, open := height, len(rows)
		// Rows that need less than the equal share take what they need.
		for changed := true; changed && open > 0; {
			changed = false
			for i, n := range rows {
				if got[i] == 0 && n <= left/open {
					got[i] = max(n, 1)
					left -= got[i]
					open--
					changed = true
					if open == 0 {
						break
					}
				}
			}
		}
		// The others split the rest; with none, everyone splits the surplus.
		split := make([]int, 0, len(rows))
		for i := range rows {
			if got[i] == 0 || open == 0 {
				split = append(split, i)
			}
		}
		for k, i := range split {
			extra := left / (len(split) - k)
			got[i] += extra
			left -= extra
		}
		y := 0
		for i := range rows {
			slots[from+i] = Slot{Screen: screen, Y: y, H: got[i]}
			y += got[i]
		}
	}
	screen, start, used := 0, 0, 0
	for i, n := range need {
		least := max(min(n, floor), 1)
		if i > start && used+least > height {
			share(start, i, screen)
			screen, start, used = screen+1, i, 0
		}
		used += least
	}
	share(start, len(need), screen)
	return slots
}
