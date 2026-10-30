package shell

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"
)

// BoardPeek is what a board looked like the last time this computer
// opened it: enough for the home's card (docs/desktop/08 §8.3). The shell
// writes it down from the bridge's board.load; the app never reads a
// board's files for it.
type BoardPeek struct {
	Notes   int      `json:"notes"`
	Stalled int      `json:"stalled,omitempty"`
	Agents  []string `json:"agents,omitempty"`
	Theme   string   `json:"theme,omitempty"`
	Dark    bool     `json:"dark,omitempty"`
	// Colors are the board's palette's six note colours.
	Colors []string `json:"colors,omitempty"`
}

// maxPeekAgents is how many names a card shows.
const maxPeekAgents = 4

// peekOf reads a board.load answer; ok is false when it is not a board.
func peekOf(raw json.RawMessage) (BoardPeek, bool) {
	type card struct {
		Who     []string `json:"who"`
		Stalled bool     `json:"stalled"`
	}
	type note struct {
		Data struct {
			Columns []struct {
				Cards []card `json:"cards"`
			} `json:"columns"`
		} `json:"data"`
		Pages []json.RawMessage `json:"pages"`
	}
	var b struct {
		Tabs *[]struct {
			Notes []note `json:"notes"`
		} `json:"tabs"`
		Settings struct {
			Theme string `json:"theme"`
		} `json:"settings"`
		Palette map[string]json.RawMessage `json:"palette"`
	}
	if json.Unmarshal(raw, &b) != nil || b.Tabs == nil {
		return BoardPeek{}, false
	}
	pk := BoardPeek{Theme: b.Settings.Theme}
	who := map[string]bool{}
	for _, t := range *b.Tabs {
		for _, n := range t.Notes {
			pk.Notes++
			for _, c := range n.Data.Columns {
				for _, cd := range c.Cards {
					if cd.Stalled {
						pk.Stalled++
					}
					for _, w := range cd.Who {
						if w = strings.TrimPrefix(strings.TrimSpace(w), "@"); w != "" {
							who[w] = true
						}
					}
				}
			}
		}
	}
	for w := range who {
		pk.Agents = append(pk.Agents, w)
	}
	sort.Strings(pk.Agents)
	if len(pk.Agents) > maxPeekAgents {
		pk.Agents = pk.Agents[:maxPeekAgents]
	}
	// the palette, as Go wrote it (Notes) or lower-cased (notes)
	field := func(k string) json.RawMessage {
		if v, ok := b.Palette[k]; ok {
			return v
		}
		return b.Palette[strings.ToUpper(k[:1])+k[1:]]
	}
	var colors []string
	if json.Unmarshal(field("notes"), &colors) == nil && len(colors) == 6 {
		pk.Colors = colors
	}
	_ = json.Unmarshal(field("dark"), &pk.Dark)
	return pk, true
}

// peek writes down what a board.load answered for a project, and saves
// when it changed.
func (s *Shell) peek(path string, result any) {
	raw, ok := result.(*json.RawMessage)
	if !ok || raw == nil {
		return
	}
	pk, ok := peekOf(*raw)
	if !ok {
		return
	}
	s.mu.Lock()
	if old, had := s.state.Boards[path]; had && reflect.DeepEqual(old, pk) {
		s.mu.Unlock()
		return
	}
	if s.state.Boards == nil {
		s.state.Boards = map[string]BoardPeek{}
	}
	s.state.Boards[path] = pk
	s.mu.Unlock()
	s.save()
}

// Peeks are what each board looked like when this computer last opened
// it, by project path.
func (s *Shell) Peeks() map[string]BoardPeek {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make(map[string]BoardPeek, len(s.state.Boards))
	for k, v := range s.state.Boards {
		out[k] = v
	}
	return out
}
