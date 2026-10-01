package widget

import (
	"reflect"
	"testing"
)

func TestWrapBreaksAtWidth(t *testing.T) {
	cases := []struct {
		text  string
		width int
		want  []string
	}{
		{"short", 20, []string{"short"}},
		{"hello wide world", 10, []string{"hello wide", "world"}},
		{"", 10, []string{""}},
		{"anything", 0, []string{"anything"}},
	}
	for _, c := range cases {
		if got := Wrap(c.text, c.width); !reflect.DeepEqual(got, c.want) {
			t.Errorf("Wrap(%q, %d) = %q, want %q", c.text, c.width, got, c.want)
		}
	}
	for _, line := range Wrap("프로젝트 개념 요약이 인용 하나의 revision 이 달라도 통째로 실패함", 14) {
		if w := Width(line); w > 14 {
			t.Errorf("line %q is %d cells wide, want at most 14", line, w)
		}
	}
}

func TestFitCutsWithoutEllipsis(t *testing.T) {
	if got := Fit("abcdef\n한글입니다\nok", 4); got != "abcd\n한글\nok" {
		t.Errorf("Fit = %q", got)
	}
	if got := Fit("한", 1); got != "" {
		t.Errorf("a wide character cannot fit in one cell, got %q", got)
	}
}

func TestKindHandles(t *testing.T) {
	k := Kind{Keys: "h l j k left right space n"}
	for _, key := range []string{"h", "left", "space", "n"} {
		if !k.Handles(key) {
			t.Errorf("Handles(%q) = false, want true", key)
		}
	}
	for _, key := range []string{"o", "", "N", "spa"} {
		if k.Handles(key) {
			t.Errorf("Handles(%q) = true, want false", key)
		}
	}
	if (Kind{}).Handles("j") {
		t.Error("a kind without keys handles nothing")
	}
}
