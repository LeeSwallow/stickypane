package app

import "github.com/LeeSwallow/stickypane/internal/api"

// calls is the use-case layer the screen goes through to change the board,
// as the command line and the bridge do: a note made, tended, archived,
// trashed, moved or saved, a setting changed, a request sent. It runs on
// the screen's clock, so what it stamps is what the screen would. What the
// screen only reads (Load, Read, Settings) and how it arranges itself
// (sticky.json's views, order and tab) it still asks the store.
func (m *Model) calls() *api.API { return api.NewWithClock(m.store, m.reg, m.now) }
