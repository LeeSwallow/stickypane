package httpfile

import (
	"testing"
	"time"
)

// A line of a WebSocket transcript says when to the millisecond.
func TestAFrameLineSaysTheMillisecond(t *testing.T) {
	at := time.Date(2026, 10, 3, 14, 2, 5, 42_000_000, time.UTC)
	for f, want := range map[Frame]string{
		{Out: true, Kind: "text", Data: "hi\nthere", At: at}: "→ 14:02:05.042  hi there",
		{Kind: "close", Code: 1000, Data: "bye", At: at}:     "← 14:02:05.042  close 1000 bye",
		{Kind: "binary", Data: "3 bytes", At: at}:            "← 14:02:05.042  binary 3 bytes",
		{Kind: "wait", Data: "1s", At: at}:                   "· wait 1s",
	} {
		if got := frameLine(f); got != want {
			t.Errorf("frameLine = %q, want %q", got, want)
		}
	}
}
