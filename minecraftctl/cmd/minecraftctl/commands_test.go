package main

import (
	"testing"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
)

func TestWorldCmdExists(t *testing.T) {
	if worldCmd == nil {
		t.Fatal("worldCmd is nil")
	}
	if worldCmd.Use != "world" {
		t.Errorf("worldCmd.Use = %q, want world", worldCmd.Use)
	}
	if worldCmd.Short == "" {
		t.Error("worldCmd should have Short description")
	}
}

func TestMapCmdExists(t *testing.T) {
	if mapCmd == nil {
		t.Fatal("mapCmd is nil")
	}
	if mapCmd.Use != "map" {
		t.Errorf("mapCmd.Use = %q, want map", mapCmd.Use)
	}
	if mapCmd.Short == "" {
		t.Error("mapCmd should have Short description")
	}
}

func TestRconCmdExists(t *testing.T) {
	if rconCmd == nil {
		t.Fatal("rconCmd is nil")
	}
	if rconCmd.Use != "rcon" {
		t.Errorf("rconCmd.Use = %q, want rcon", rconCmd.Use)
	}
	if rconCmd.Short == "" {
		t.Error("rconCmd should have Short description")
	}
}

func TestWorldSubcommands(t *testing.T) {
	subcommands := []string{"list", "info", "create", "register", "upgrade"}

	for _, name := range subcommands {
		found := false
		for _, cmd := range worldCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("worldCmd missing subcommand %q", name)
		}
	}
}

func TestMapSubcommands(t *testing.T) {
	subcommands := []string{"build", "preview", "manifest", "index"}

	for _, name := range subcommands {
		found := false
		for _, cmd := range mapCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("mapCmd missing subcommand %q", name)
		}
	}
}

func TestRconSubcommands(t *testing.T) {
	subcommands := []string{"status", "send"}

	for _, name := range subcommands {
		found := false
		for _, cmd := range rconCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("rconCmd missing subcommand %q", name)
		}
	}
}

func TestCommandsRegisteredWithRoot(t *testing.T) {
	rootCmd := root.GetRootCmd()

	// Check that main commands are registered
	expectedCommands := []string{"world", "map", "rcon", "jar"}

	for _, name := range expectedCommands {
		found := false
		for _, cmd := range rootCmd.Commands() {
			if cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("Root command missing subcommand %q", name)
		}
	}
}

func TestWorldCmdHasSubcommands(t *testing.T) {
	if len(worldCmd.Commands()) == 0 {
		t.Error("worldCmd should have subcommands")
	}
}

func TestMapCmdHasSubcommands(t *testing.T) {
	if len(mapCmd.Commands()) == 0 {
		t.Error("mapCmd should have subcommands")
	}
}

func TestRconCmdHasSubcommands(t *testing.T) {
	if len(rconCmd.Commands()) == 0 {
		t.Error("rconCmd should have subcommands")
	}
}
