package systemd

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// UnitType represents the type of systemd unit
type UnitType string

const (
	UnitService UnitType = "service"
	UnitTimer   UnitType = "timer"
)

// LogOptions configures journalctl output
type LogOptions struct {
	Follow  bool   // -f flag
	Lines   int    // -n flag (default: 100)
	Since   string // --since flag
	Output  string // -o flag (short, json, etc.)
	NoPager bool   // --no-pager flag
}

// FormatUnitName constructs the full unit name from components
// e.g., FormatUnitName("minecraft", "survival", UnitService) -> "minecraft@survival.service"
func FormatUnitName(prefix, instance string, unitType UnitType) string {
	return fmt.Sprintf("%s@%s.%s", prefix, instance, unitType)
}

// ErrPermission is returned when systemctl refuses an operation because the
// caller lacks the privileges to manage system units (typically: not root).
var ErrPermission = errors.New("permission denied")

// execCommand is a seam for tests.
var execCommand = exec.Command

// isPermissionError reports whether systemctl stderr output indicates a
// privilege problem rather than an ordinary failure.
func isPermissionError(stderr string) bool {
	s := strings.ToLower(stderr)
	for _, marker := range []string{
		"interactive authentication required",
		"access denied",
		"permission denied",
	} {
		if strings.Contains(s, marker) {
			return true
		}
	}
	return false
}

// runSystemctl executes a systemctl command and passes through stdout/stderr.
// Permission failures are returned as ErrPermission (without echoing systemctl's
// raw message); other failures are wrapped with the command that failed.
func runSystemctl(args ...string) error {
	var stderr bytes.Buffer
	cmd := execCommand("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil && isPermissionError(stderr.String()) {
		return fmt.Errorf("systemctl %s: %w", strings.Join(args, " "), ErrPermission)
	}
	if stderr.Len() > 0 {
		_, _ = os.Stderr.Write(stderr.Bytes())
	}
	if err != nil {
		return fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return nil
}

// Status runs systemctl status for a unit
func Status(unit string) error {
	return runSystemctl("status", unit)
}

// Start runs systemctl start for a unit
func Start(unit string) error {
	return runSystemctl("start", unit)
}

// Stop runs systemctl stop for a unit
func Stop(unit string) error {
	return runSystemctl("stop", unit)
}

// Restart runs systemctl restart for a unit
func Restart(unit string) error {
	return runSystemctl("restart", unit)
}

// Enable runs systemctl enable for a unit
func Enable(unit string) error {
	return runSystemctl("enable", unit)
}

// EnableNow runs systemctl enable --now for a unit
func EnableNow(unit string) error {
	return runSystemctl("enable", "--now", unit)
}

// Disable runs systemctl disable for a unit
func Disable(unit string) error {
	return runSystemctl("disable", unit)
}

// DaemonReload runs systemctl daemon-reload to pick up new/changed unit files
func DaemonReload() error {
	return runSystemctl("daemon-reload")
}

// IsActive checks if a unit is active (returns true if active, false otherwise)
func IsActive(unit string) (bool, error) {
	cmd := exec.Command("systemctl", "is-active", "--quiet", unit)
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check service status: %w", err)
	}
	return true, nil
}

// IsEnabled checks if a unit is enabled (returns true if enabled, false otherwise)
func IsEnabled(unit string) (bool, error) {
	cmd := exec.Command("systemctl", "is-enabled", "--quiet", unit)
	err := cmd.Run()
	if err != nil {
		if _, ok := err.(*exec.ExitError); ok {
			return false, nil
		}
		return false, fmt.Errorf("failed to check enabled status: %w", err)
	}
	return true, nil
}

// Unit is a loaded unit as listed by `systemctl list-units`.
type Unit struct {
	Name   string // e.g. minecraft@default.service
	Active string // ACTIVE column: active, activating, deactivating, ...
	Sub    string // SUB column: running, auto-restart, stop-sigterm, ...
}

// ListUnits lists the loaded units matching pattern (a glob such as
// "minecraft@*.service") whose active state is one of states. With no states,
// every loaded unit matching pattern is listed.
func ListUnits(pattern string, states ...string) ([]Unit, error) {
	args := []string{"list-units", "--all", "--plain", "--no-legend", "--no-pager"}
	if len(states) > 0 {
		args = append(args, "--state="+strings.Join(states, ","))
	}
	args = append(args, pattern)

	out, err := execCommand("systemctl", args...).Output()
	if err != nil {
		return nil, fmt.Errorf("systemctl %s failed: %w", strings.Join(args, " "), err)
	}
	return parseListUnits(string(out)), nil
}

// parseListUnits parses `systemctl list-units --plain --no-legend` output:
// UNIT LOAD ACTIVE SUB DESCRIPTION, one unit per line.
func parseListUnits(out string) []Unit {
	var units []Unit
	for _, line := range strings.Split(out, "\n") {
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		units = append(units, Unit{Name: fields[0], Active: fields[2], Sub: fields[3]})
	}
	return units
}

// GetActiveState returns the active state of a unit (active, inactive, failed, etc.)
func GetActiveState(unit string) string {
	cmd := exec.Command("systemctl", "show", "-p", "ActiveState", "--value", unit)
	output, err := cmd.Output()
	if err != nil {
		return "unknown"
	}
	// Trim newline
	state := string(output)
	if len(state) > 0 && state[len(state)-1] == '\n' {
		state = state[:len(state)-1]
	}
	return state
}

// Logs runs journalctl for a unit with the given options
func Logs(unit string, opts LogOptions) error {
	args := []string{"-u", unit}

	if opts.Follow {
		args = append(args, "-f")
	}

	if opts.Lines > 0 {
		args = append(args, "-n", strconv.Itoa(opts.Lines))
	} else if !opts.Follow {
		// Default to 100 lines if not following
		args = append(args, "-n", "100")
	}

	if opts.Since != "" {
		args = append(args, "--since", opts.Since)
	}

	if opts.Output != "" {
		args = append(args, "-o", opts.Output)
	}

	if opts.NoPager {
		args = append(args, "--no-pager")
	}

	cmd := exec.Command("journalctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin
	return cmd.Run()
}
