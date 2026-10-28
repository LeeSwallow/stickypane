package shell

import (
	"bytes"
	"testing"
)

func TestTrayIconIsATemplateOnMacOS(t *testing.T) {
	ic := TrayIcons{Template: []byte("template"), Colour: []byte("colour")}
	icon, template := trayIcon("darwin", ic)
	if !template || !bytes.Equal(icon, ic.Template) {
		t.Fatalf("darwin: got %q template=%v, want the template image with the template flag", icon, template)
	}
}

func TestTrayIconIsInColourOnLinux(t *testing.T) {
	ic := TrayIcons{Template: []byte("template"), Colour: []byte("colour")}
	icon, template := trayIcon("linux", ic)
	if template || !bytes.Equal(icon, ic.Colour) {
		t.Fatalf("linux: got %q template=%v, want the colour icon without the template flag", icon, template)
	}
}

func TestTrayIconWithoutATemplateFallsBackToColour(t *testing.T) {
	ic := TrayIcons{Colour: []byte("colour")}
	icon, template := trayIcon("darwin", ic)
	if template || !bytes.Equal(icon, ic.Colour) {
		t.Fatalf("darwin without a template: got %q template=%v", icon, template)
	}
}
