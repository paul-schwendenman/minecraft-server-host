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

func TestUnitFileState(t *testing.T) {
	stubSystemctl(t, `echo disabled; exit 1`)
	got, err := UnitFileState("x.timer")
	if err != nil || got != "disabled" {
		t.Errorf("UnitFileState() = %q, %v; want disabled, nil", got, err)
	}

	stubSystemctl(t, `echo not-found; exit 4`)
	if got, _ := UnitFileState("x.timer"); got != "not-found" {
		t.Errorf("UnitFileState() = %q, want not-found", got)
	}

	stubSystemctl(t, `exit 1`)
	if _, err := UnitFileState("x.timer"); err == nil {
		t.Error("UnitFileState() with no output: want error")
	}
}

// recordCommands replaces execCommand with a stub that records each
// invocation ("name arg...") and runs script with the call's args as "$@".
func recordCommands(t *testing.T, script string) *[]string {
	t.Helper()
	orig := execCommand
	var calls []string
	execCommand = func(name string, args ...string) *exec.Cmd {
		calls = append(calls, strings.TrimSpace(name+" "+strings.Join(args, " ")))
		return exec.Command("sh", append([]string{"-c", script, name}, args...)...)
	}
	t.Cleanup(func() { execCommand = orig })
	return &calls
}

func TestUnitActions(t *testing.T) {
	calls := recordCommands(t, "exit 0")
	actions := []struct {
		fn   func(string) error
		want string
	}{
		{Status, "systemctl status u.service"},
		{Start, "systemctl start u.service"},
		{Stop, "systemctl stop u.service"},
		{Restart, "systemctl restart u.service"},
		{Enable, "systemctl enable u.service"},
		{EnableNow, "systemctl enable --now u.service"},
		{Disable, "systemctl disable u.service"},
	}
	for _, a := range actions {
		if err := a.fn("u.service"); err != nil {
			t.Errorf("%s: %v", a.want, err)
		}
	}
	if err := DaemonReload(); err != nil {
		t.Errorf("DaemonReload: %v", err)
	}
	want := make([]string, 0, len(actions)+1)
	for _, a := range actions {
		want = append(want, a.want)
	}
	want = append(want, "systemctl daemon-reload")
	if strings.Join(*calls, "\n") != strings.Join(want, "\n") {
		t.Errorf("calls = %q, want %q", *calls, want)
	}
}

func TestIsActiveAndIsEnabled(t *testing.T) {
	calls := recordCommands(t, "exit 0")
	if ok, err := IsActive("u.service"); !ok || err != nil {
		t.Errorf("IsActive() = %v, %v; want true", ok, err)
	}
	if ok, err := IsEnabled("u.timer"); !ok || err != nil {
		t.Errorf("IsEnabled() = %v, %v; want true", ok, err)
	}
	want := "systemctl is-active --quiet u.service\nsystemctl is-enabled --quiet u.timer"
	if got := strings.Join(*calls, "\n"); got != want {
		t.Errorf("calls = %q, want %q", got, want)
	}

	// A non-zero exit just means "no"
	recordCommands(t, "exit 3")
	if ok, err := IsActive("u.service"); ok || err != nil {
		t.Errorf("IsActive(inactive) = %v, %v; want false, nil", ok, err)
	}
	if ok, err := IsEnabled("u.timer"); ok || err != nil {
		t.Errorf("IsEnabled(disabled) = %v, %v; want false, nil", ok, err)
	}

	// systemctl missing altogether is an error
	orig := execCommand
	execCommand = func(string, ...string) *exec.Cmd { return exec.Command("/nonexistent/systemctl") }
	t.Cleanup(func() { execCommand = orig })
	if _, err := IsActive("u.service"); err == nil {
		t.Error("IsActive() without systemctl: want error")
	}
	if _, err := IsEnabled("u.timer"); err == nil {
		t.Error("IsEnabled() without systemctl: want error")
	}
}

func TestGetActiveState(t *testing.T) {
	calls := recordCommands(t, `echo failed`)
	if got := GetActiveState("u.service"); got != "failed" {
		t.Errorf("GetActiveState() = %q, want failed", got)
	}
	if want := "systemctl show -p ActiveState --value u.service"; (*calls)[0] != want {
		t.Errorf("call = %q, want %q", (*calls)[0], want)
	}

	recordCommands(t, `exit 1`)
	if got := GetActiveState("u.service"); got != "unknown" {
		t.Errorf("GetActiveState() on error = %q, want unknown", got)
	}
}

func TestLogsArgs(t *testing.T) {
	tests := []struct {
		opts LogOptions
		want string
	}{
		{LogOptions{}, "journalctl -u u.service -n 100"},
		{LogOptions{Follow: true}, "journalctl -u u.service -f"},
		{LogOptions{Follow: true, Lines: 5}, "journalctl -u u.service -f -n 5"},
		{
			LogOptions{Lines: 20, Since: "1 hour ago", Output: "json", NoPager: true},
			"journalctl -u u.service -n 20 --since 1 hour ago -o json --no-pager",
		},
	}
	for _, tt := range tests {
		calls := recordCommands(t, "exit 0")
		if err := Logs("u.service", tt.opts); err != nil {
			t.Errorf("Logs(%+v) = %v", tt.opts, err)
		}
		if (*calls)[0] != tt.want {
			t.Errorf("Logs(%+v) ran %q, want %q", tt.opts, (*calls)[0], tt.want)
		}
	}

	recordCommands(t, "exit 1")
	if err := Logs("u.service", LogOptions{}); err == nil {
		t.Error("Logs() with journalctl failing: want error")
	}
}
