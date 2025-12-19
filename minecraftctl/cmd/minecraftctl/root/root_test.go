package root

import (
	"strings"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := NewRootCmd()
	if cmd == nil {
		t.Fatal("NewRootCmd returned nil")
	}

	if cmd.Use != "minecraftctl" {
		t.Errorf("Use = %q, want minecraftctl", cmd.Use)
	}

	if cmd.Short == "" {
		t.Error("Short description should not be empty")
	}

	if cmd.Long == "" {
		t.Error("Long description should not be empty")
	}
}

func TestGetRootCmd(t *testing.T) {
	cmd := GetRootCmd()
	if cmd == nil {
		t.Fatal("GetRootCmd returned nil")
	}
}

func TestRootCmdFlags(t *testing.T) {
	cmd := NewRootCmd()

	// Check persistent flags exist
	flags := []string{
		"config",
		"worlds-dir",
		"maps-dir",
		"jars-dir",
		"rcon-host",
		"rcon-port",
		"rcon-password",
		"verbose",
	}

	for _, flag := range flags {
		f := cmd.PersistentFlags().Lookup(flag)
		if f == nil {
			t.Errorf("Flag %q not found", flag)
		}
	}
}

func TestRootCmdVerboseShorthand(t *testing.T) {
	cmd := NewRootCmd()
	f := cmd.PersistentFlags().ShorthandLookup("v")
	if f == nil {
		t.Error("Verbose flag should have -v shorthand")
	}
	if f.Name != "verbose" {
		t.Errorf("Shorthand -v should be for verbose, got %q", f.Name)
	}
}

func TestRootCmdHelp(t *testing.T) {
	cmd := NewRootCmd()

	// Test that UsageString contains expected content
	usage := cmd.UsageString()

	expectedStrings := []string{
		"minecraftctl",
		"--config",
		"--verbose",
	}

	for _, s := range expectedStrings {
		if !strings.Contains(usage, s) {
			t.Errorf("Usage string missing %q", s)
		}
	}

	// Also check Short and Long descriptions
	if !strings.Contains(cmd.Short, "Minecraft") {
		t.Error("Short description should mention Minecraft")
	}
}

func TestRootCmdVersion(t *testing.T) {
	cmd := NewRootCmd()

	// Check version is set
	if cmd.Version == "" {
		// Version might be empty in tests if not built with ldflags
		t.Log("Version is empty (expected in unit tests without ldflags)")
	}
}

func TestRootCmdFlagDefaults(t *testing.T) {
	cmd := NewRootCmd()

	tests := []struct {
		flag     string
		defValue string
	}{
		{"config", ""},
		{"worlds-dir", ""},
		{"maps-dir", ""},
		{"jars-dir", ""},
		{"rcon-host", ""},
		{"rcon-port", "0"},
		{"rcon-password", ""},
		{"verbose", "false"},
	}

	for _, tt := range tests {
		f := cmd.PersistentFlags().Lookup(tt.flag)
		if f == nil {
			t.Errorf("Flag %q not found", tt.flag)
			continue
		}
		if f.DefValue != tt.defValue {
			t.Errorf("Flag %q default = %q, want %q", tt.flag, f.DefValue, tt.defValue)
		}
	}
}

func TestRootCmdFlagUsage(t *testing.T) {
	cmd := NewRootCmd()

	// Specific checks for important flags
	configFlag := cmd.PersistentFlags().Lookup("config")
	if configFlag != nil && configFlag.Usage == "" {
		t.Error("config flag should have usage description")
	}

	verboseFlag := cmd.PersistentFlags().Lookup("verbose")
	if verboseFlag != nil && verboseFlag.Usage == "" {
		t.Error("verbose flag should have usage description")
	}

	worldsDirFlag := cmd.PersistentFlags().Lookup("worlds-dir")
	if worldsDirFlag != nil && worldsDirFlag.Usage == "" {
		t.Error("worlds-dir flag should have usage description")
	}
}
