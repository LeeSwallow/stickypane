package sshconfig

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func write(t *testing.T, path, text string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
		t.Fatal(err)
	}
}

func TestAliases(t *testing.T) {
	home := t.TempDir()
	ssh := filepath.Join(home, ".ssh")
	write(t, filepath.Join(ssh, "config"), `# mine
Include config.d/*
Include ~/extra/ssh.conf

Host dev box2   # two names on a line
  HostName 10.0.0.5
  User min

host=lower
HOST = upper "quoted name"

Host *.internal !bad prod-?
  User ops

Host *
  ServerAliveInterval 30

Match host bastion exec "true"
  User jump
`)
	write(t, filepath.Join(ssh, "config.d", "10-work"), "Host work-a work-b\n  HostName w\n")
	write(t, filepath.Join(ssh, "config.d", "20-loop"), "Include config.d/20-loop\nHost loop\n")
	write(t, filepath.Join(home, "extra", "ssh.conf"), "Host extra dev\n")

	got := Aliases(home)
	want := []string{"work-a", "work-b", "loop", "extra", "dev", "box2", "lower", "upper", "quoted name"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("Aliases =\n %q\nwant\n %q", got, want)
	}
}

func TestAliasesWithNoConfig(t *testing.T) {
	if got := Aliases(t.TempDir()); len(got) != 0 {
		t.Fatalf("Aliases = %q", got)
	}
}

func TestSplit(t *testing.T) {
	for line, want := range map[string][]string{
		"Host a b":          {"Host", "a", "b"},
		"Host=a":            {"Host", "a"},
		"  Host = a  # c":   {"Host", "a"},
		`Host "x y" z`:      {"Host", "x y", "z"},
		"# Host commented":  nil,
		"Include\tfoo/*.cf": {"Include", "foo/*.cf"},
	} {
		k, args := split(line)
		var got []string
		if k != "" {
			got = append([]string{k}, args...)
		}
		if !reflect.DeepEqual(got, want) {
			t.Errorf("split(%q) = %q, want %q", line, got, want)
		}
	}
}
