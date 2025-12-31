package main

import (
	"testing"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
)

func TestWorldCmdExists(t *testing.T) {
	if WorldCmd == nil {
		t.Fatal("WorldCmd is nil")
	}
	if WorldCmd.Use != "world" {
		t.Errorf("WorldCmd.Use = %q, want world", WorldCmd.Use)
	}
	if WorldCmd.Short == "" {
		t.Error("WorldCmd should have Short description")
	}
}

func TestMapCmdExists(t *testing.T) {
	if MapCmd == nil {
		t.Fatal("MapCmd is nil")
	}
	if MapCmd.Use != "map" {
		t.Errorf("MapCmd.Use = %q, want map", MapCmd.Use)
	}
	if MapCmd.Short == "" {
		t.Error("MapCmd should have Short description")
	}
}

func TestRconCmdExists(t *testing.T) {
	if RconCmd == nil {
		t.Fatal("RconCmd is nil")
	}
	if RconCmd.Use != "rcon" {
		t.Errorf("RconCmd.Use = %q, want rcon", RconCmd.Use)
	}
	if RconCmd.Short == "" {
		t.Error("RconCmd should have Short description")
	}
}

func TestWorldSubcommands(t *testing.T) {
	subcommands := []string{
		"list", "info", "create", "register", "upgrade",
		"status", "start", "stop", "restart", "enable", "disable", "logs",
		"backup",
	}

	for _, name := range subcommands {
		found := false
		for _, cmd := range WorldCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("WorldCmd missing subcommand %q", name)
		}
	}
}

func TestWorldBackupSubcommands(t *testing.T) {
	// Find backup command first
	var backupCmd = worldBackupCmd
	if backupCmd == nil {
		t.Fatal("worldBackupCmd is nil")
	}

	subcommands := []string{"status", "start", "stop", "enable", "disable", "logs"}
	for _, name := range subcommands {
		found := false
		for _, cmd := range backupCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("world backup missing subcommand %q", name)
		}
	}
}

func TestMapSubcommands(t *testing.T) {
	subcommands := []string{
		"build", "preview", "manifest", "index", "config",
		"backup", "update",
	}

	for _, name := range subcommands {
		found := false
		for _, cmd := range MapCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("MapCmd missing subcommand %q", name)
		}
	}
}

func TestMapBackupSubcommands(t *testing.T) {
	if mapBackupCmd == nil {
		t.Fatal("mapBackupCmd is nil")
	}

	subcommands := []string{"status", "start", "stop", "enable", "disable", "logs"}
	for _, name := range subcommands {
		found := false
		for _, cmd := range mapBackupCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("map backup missing subcommand %q", name)
		}
	}
}

func TestMapUpdateSubcommands(t *testing.T) {
	if mapUpdateCmd == nil {
		t.Fatal("mapUpdateCmd is nil")
	}

	subcommands := []string{"status", "start", "stop", "enable", "disable", "logs"}
	for _, name := range subcommands {
		found := false
		for _, cmd := range mapUpdateCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("map update missing subcommand %q", name)
		}
	}
}

func TestRconSubcommands(t *testing.T) {
	subcommands := []string{"status", "send"}

	for _, name := range subcommands {
		found := false
		for _, cmd := range RconCmd.Commands() {
			if cmd.Use == name || cmd.Name() == name {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("RconCmd missing subcommand %q", name)
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
	if len(WorldCmd.Commands()) == 0 {
		t.Error("WorldCmd should have subcommands")
	}
}

func TestMapCmdHasSubcommands(t *testing.T) {
	if len(MapCmd.Commands()) == 0 {
		t.Error("MapCmd should have subcommands")
	}
}

func TestRconCmdHasSubcommands(t *testing.T) {
	if len(RconCmd.Commands()) == 0 {
		t.Error("RconCmd should have subcommands")
	}
}
