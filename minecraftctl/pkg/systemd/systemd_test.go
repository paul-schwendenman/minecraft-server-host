package systemd

import (
	"errors"
	"os/exec"
	"strings"
	"testing"
)

// stubSystemctl replaces the systemctl invocation with a shell script.
func stubSystemctl(t *testing.T, script string) {
	t.Helper()
	orig := execCommand
	execCommand = func(name string, args ...string) *exec.Cmd {
		return exec.Command("sh", "-c", script)
	}
	t.Cleanup(func() { execCommand = orig })
}

func TestRunSystemctlPermissionError(t *testing.T) {
	stubSystemctl(t, `echo "Failed to start x.service: Interactive authentication required." >&2; exit 1`)

	err := Start("x.service")
	if !errors.Is(err, ErrPermission) {
		t.Fatalf("Start() error = %v, want ErrPermission", err)
	}
	if !strings.Contains(err.Error(), "systemctl start x.service") {
		t.Errorf("error %q should name the failed command", err)
	}
}

func TestRunSystemctlOtherFailure(t *testing.T) {
	stubSystemctl(t, `echo "Unit x.service not found." >&2; exit 5`)

	err := Start("x.service")
	if err == nil {
		t.Fatal("Start() = nil, want error")
	}
	if errors.Is(err, ErrPermission) {
		t.Errorf("error %v should not be ErrPermission", err)
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("error %v should wrap *exec.ExitError", err)
	}
	if exitErr.ExitCode() != 5 {
		t.Errorf("exit code = %d, want 5", exitErr.ExitCode())
	}
	if !strings.Contains(err.Error(), "systemctl start x.service failed") {
		t.Errorf("error %q should name the failed command", err)
	}
}

func TestRunSystemctlSuccess(t *testing.T) {
	stubSystemctl(t, `exit 0`)

	if err := Start("x.service"); err != nil {
		t.Errorf("Start() = %v, want nil", err)
	}
}

func TestIsPermissionError(t *testing.T) {
	tests := []struct {
		stderr string
		want   bool
	}{
		{"Failed to start x.service: Interactive authentication required.", true},
		{"Failed to connect to bus: Access denied", true},
		{"Permission denied", true},
		{"Unit x.service not found.", false},
		{"", false},
	}
	for _, tt := range tests {
		if got := isPermissionError(tt.stderr); got != tt.want {
			t.Errorf("isPermissionError(%q) = %v, want %v", tt.stderr, got, tt.want)
		}
	}
}

func TestFormatUnitName(t *testing.T) {
	tests := []struct {
		prefix   string
		instance string
		unitType UnitType
		want     string
	}{
		{"minecraft", "survival", UnitService, "minecraft@survival.service"},
		{"minecraft", "creative", UnitService, "minecraft@creative.service"},
		{"minecraft-world-backup", "survival", UnitService, "minecraft-world-backup@survival.service"},
		{"minecraft-world-backup", "test", UnitTimer, "minecraft-world-backup@test.timer"},
		{"minecraft-map-backup", "world", UnitService, "minecraft-map-backup@world.service"},
		{"minecraft-map-build", "survival", UnitTimer, "minecraft-map-build@survival.timer"},
		{"minecraft-map-build", "creative", UnitService, "minecraft-map-build@creative.service"},
	}

	for _, tt := range tests {
		t.Run(tt.want, func(t *testing.T) {
			got := FormatUnitName(tt.prefix, tt.instance, tt.unitType)
			if got != tt.want {
				t.Errorf("FormatUnitName(%q, %q, %q) = %q, want %q",
					tt.prefix, tt.instance, tt.unitType, got, tt.want)
			}
		})
	}
}

func TestUnitTypeConstants(t *testing.T) {
	if UnitService != "service" {
		t.Errorf("UnitService = %q, want %q", UnitService, "service")
	}
	if UnitTimer != "timer" {
		t.Errorf("UnitTimer = %q, want %q", UnitTimer, "timer")
	}
}

func TestParseListUnits(t *testing.T) {
	out := "minecraft@default.service loaded active running Minecraft Server default\n" +
		"minecraft@old.service loaded activating auto-restart Minecraft Server old\n\n"
	want := []Unit{
		{Name: "minecraft@default.service", Active: "active", Sub: "running"},
		{Name: "minecraft@old.service", Active: "activating", Sub: "auto-restart"},
	}
	got := parseListUnits(out)
	if len(got) != len(want) {
		t.Fatalf("parseListUnits() = %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Errorf("unit %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if units := parseListUnits(""); len(units) != 0 {
		t.Errorf("parseListUnits(\"\") = %+v, want none", units)
	}
}

func TestListUnitsArgs(t *testing.T) {
	orig := execCommand
	var gotArgs []string
	execCommand = func(name string, args ...string) *exec.Cmd {
		gotArgs = args
		return exec.Command("sh", "-c", `echo "minecraft@default.service loaded active running Minecraft Server default"`)
	}
	t.Cleanup(func() { execCommand = orig })

	units, err := ListUnits("minecraft@*.service", "active", "activating")
	if err != nil {
		t.Fatal(err)
	}
	if len(units) != 1 || units[0].Name != "minecraft@default.service" {
		t.Errorf("ListUnits() = %+v", units)
	}
	joined := strings.Join(gotArgs, " ")
	if !strings.Contains(joined, "--state=active,activating") || !strings.HasSuffix(joined, "minecraft@*.service") {
		t.Errorf("systemctl args = %q", joined)
	}
}
