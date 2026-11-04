package app

import (
	"strings"
	"unicode"
	"unicode/utf8"
)

// twoSet is the Korean 2-set (dubeolsik) layout: the jamo a key types,
// and the Latin key it is. With the Korean input on, the terminal sends ㅓ
// for j; the board reads it as j. Shift gives a double consonant on five
// keys and a second vowel on two (ㅃ is Q, ㅒ is O); on the others it gives
// the same jamo, which is read as the small letter.
var twoSet = map[rune]rune{
	'ㅂ': 'q', 'ㅈ': 'w', 'ㄷ': 'e', 'ㄱ': 'r', 'ㅅ': 't', 'ㅛ': 'y', 'ㅕ': 'u', 'ㅑ': 'i', 'ㅐ': 'o', 'ㅔ': 'p',
	'ㅁ': 'a', 'ㄴ': 's', 'ㅇ': 'd', 'ㄹ': 'f', 'ㅎ': 'g', 'ㅗ': 'h', 'ㅓ': 'j', 'ㅏ': 'k', 'ㅣ': 'l',
	'ㅋ': 'z', 'ㅌ': 'x', 'ㅊ': 'c', 'ㅍ': 'v', 'ㅠ': 'b', 'ㅜ': 'n', 'ㅡ': 'm',
	'ㅃ': 'Q', 'ㅉ': 'W', 'ㄸ': 'E', 'ㄲ': 'R', 'ㅆ': 'T', 'ㅒ': 'O', 'ㅖ': 'P',
}

// latinKey is key as the board matches it: a jamo of the 2-set layout is
// the Latin key that types it, modifiers kept ("ctrl+ㅓ" is "ctrl+j",
// "shift+ㅓ" is "J"). Any other key is itself. Only the board's keys go
// through it; what is typed into a prompt or the editor never does.
func latinKey(key string) string {
	base, mods := key, ""
	if len(key) > 1 {
		if i := strings.LastIndex(key[:len(key)-1], "+"); i >= 0 {
			base, mods = key[i+1:], key[:i+1]
		}
	}
	r, size := utf8.DecodeRuneInString(base)
	l, ok := twoSet[r]
	if !ok || size != len(base) {
		return key
	}
	if mods == "shift+" {
		return string(unicode.ToUpper(l))
	}
	return mods + string(l)
}
