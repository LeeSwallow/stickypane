package checklist

import "regexp"

// itemRe splits a checkbox line into: prefix up to "[", the mark, "] ", text.
var itemRe = regexp.MustCompile(`^(\s*[-*] \[)([ xX])(\] ?)(.*)$`)
