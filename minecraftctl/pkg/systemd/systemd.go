package systemd

import (
	"fmt"
	"os"
	"os/exec"
	"strconv"
)

// UnitType represents the type of systemd unit
type UnitType string

const (
	UnitService UnitType = "service"
	UnitTimer   UnitType = "timer"
)

// LogOptions configures journalctl output
type LogOptions struct {
	Follow   bool   // -f flag
	Lines    int    // -n flag (default: 100)
	Since    string // --since flag
	Output   string // -o flag (short, json, etc.)
	NoPager  bool   // --no-pager flag
}

// FormatUnitName constructs the full unit name from components
// e.g., FormatUnitName("minecraft", "survival", UnitService) -> "minecraft@survival.service"
func FormatUnitName(prefix, instance string, unitType UnitType) string {
	return fmt.Sprintf("%s@%s.%s", prefix, instance, unitType)
}

// runSystemctl executes a systemctl command and passes through stdout/stderr
func runSystemctl(args ...string) error {
	cmd := exec.Command("systemctl", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
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
