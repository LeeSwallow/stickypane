package main

import (
	"bytes"
	"image"
	"image/png"
	"testing"
)

// The macOS tray icon is a template image: every pixel black, only its
// alpha says where the notes are, so the menu bar can colour it.
func TestTrayTemplateIsBlackAndAlpha(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(trayTemplate))
	if err != nil {
		t.Fatal(err)
	}
	if b := img.Bounds(); b.Dx() != 36 || b.Dy() != 36 {
		t.Fatalf("size %v, want 36x36 (18pt at @2x)", b.Size())
	}
	opaque := 0
	for y := 0; y < 36; y++ {
		for x := 0; x < 36; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if r != 0 || g != 0 || b != 0 {
				t.Fatalf("pixel (%d,%d) is not black: %d %d %d", x, y, r, g, b)
			}
			if a > 0x8000 {
				opaque++
			}
		}
	}
	if opaque < 36*36/5 {
		t.Fatalf("only %d of %d pixels are drawn; the icon is nearly empty", opaque, 36*36)
	}
}

// The Linux tray icon is in colour.
func TestTrayColourHasColour(t *testing.T) {
	img, err := png.Decode(bytes.NewReader(trayColour))
	if err != nil {
		t.Fatal(err)
	}
	if !hasColour(img) {
		t.Fatal("the Linux tray icon has no colour in it")
	}
}

// A copy run for a check has its own single instance.
func TestInstanceID(t *testing.T) {
	if got := instanceID(""); got != "io.github.leeswallow.stickypane" {
		t.Errorf("instanceID() = %q", got)
	}
	if got := instanceID("check"); got != "io.github.leeswallow.stickypane.check" {
		t.Errorf("instanceID(check) = %q", got)
	}
}

func hasColour(img image.Image) bool {
	b := img.Bounds()
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			r, g, bl, a := img.At(x, y).RGBA()
			if a > 0x8000 && (r != g || g != bl) {
				return true
			}
		}
	}
	return false
}
