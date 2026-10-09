package main

import (
	"strings"
	"testing"
)

func TestNewRootCmdRegistersEverySubcommand(t *testing.T) {
	root := newRootCmd()

	if root.Use != "ocd" {
		t.Errorf("Use = %q, want ocd", root.Use)
	}
	if !strings.Contains(root.Version, "dev") {
		t.Errorf("Version = %q, want it to include the build version", root.Version)
	}

	want := map[string]bool{
		"interact": false, "extract": false, "diff": false,
		"clean": false, "stat": false, "check": false, "origin": false,
	}
	for _, c := range root.Commands() {
		if _, ok := want[c.Name()]; ok {
			want[c.Name()] = true
		}
	}
	for name, found := range want {
		if !found {
			t.Errorf("subcommand %q is not registered", name)
		}
	}
}

func TestRootCmdVersionFlag(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"--version"})
	root.SetOut(&strings.Builder{})

	if err := root.Execute(); err != nil {
		t.Fatalf("Execute: %v", err)
	}
}

func TestRootCmdUnknownSubcommand(t *testing.T) {
	root := newRootCmd()
	root.SetArgs([]string{"nonsense"})
	root.SetOut(&strings.Builder{})
	root.SetErr(&strings.Builder{})

	if err := root.Execute(); err == nil {
		t.Error("expected an error for an unknown subcommand")
	}
}
