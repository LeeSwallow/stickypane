//go:build !darwin && !windows

package env

// systemLocale has nothing to add on Linux and the BSDs: the locale
// variables are the system's answer.
func systemLocale() string { return "" }
