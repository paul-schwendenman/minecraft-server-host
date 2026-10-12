package backup

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/paul/minecraftctl/pkg/envfile"
)

// PasswordEnvKey is the env file line RotatePassword rewrites
const PasswordEnvKey = "RESTIC_PASSWORD"

// Key is one entry of `restic key list --json`
type Key struct {
	ID       string `json:"id"`
	Current  bool   `json:"current"`
	UserName string `json:"userName"`
	HostName string `json:"hostName"`
	Created  string `json:"created"`
}

// GeneratePassword returns 32 random bytes from crypto/rand as unpadded
// base64url (43 characters of A-Z a-z 0-9 - _), which needs no quoting in
// an env file or a shell
func GeneratePassword() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("failed to generate a password: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(b), nil
}

// ListKeys returns the repository's keys; the one c.Password opened is
// marked Current
func (c *Config) ListKeys() ([]Key, error) {
	cmd := c.command(c.RunAs, "key", "list", "--json")
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		if msg := strings.TrimSpace(stderr.String()); msg != "" {
			return nil, fmt.Errorf("restic key list: %w: %s", err, msg)
		}
		return nil, fmt.Errorf("restic key list: %w", err)
	}
	var keys []Key
	if err := json.Unmarshal(stdout.Bytes(), &keys); err != nil {
		return nil, fmt.Errorf("restic key list: can't parse output: %w", err)
	}
	return keys, nil
}

// currentKey returns the ID of the one key marked current
func currentKey(keys []Key) (string, error) {
	id := ""
	for _, k := range keys {
		if !k.Current {
			continue
		}
		if id != "" {
			return "", fmt.Errorf("restic marks more than one key current")
		}
		id = k.ID
	}
	if id == "" {
		return "", fmt.Errorf("restic marks no key current")
	}
	return id, nil
}

// addedKey returns the one key in after that isn't in before
func addedKey(before, after []Key) (string, error) {
	seen := make(map[string]bool, len(before))
	for _, k := range before {
		seen[k.ID] = true
	}
	var added []string
	for _, k := range after {
		if !seen[k.ID] {
			added = append(added, k.ID)
		}
	}
	if len(added) != 1 {
		return "", fmt.Errorf("expected exactly one new key, found %d (%v)", len(added), added)
	}
	return added[0], nil
}

func hasKey(keys []Key, id string) bool {
	for _, k := range keys {
		if k.ID == id {
			return true
		}
	}
	return false
}

// AddKey adds a key for newPassword. restic reads it from a 0600 file that
// is removed again before AddKey returns, so it's never in argv or the
// environment.
func (c *Config) AddKey(newPassword string) error {
	f, err := os.CreateTemp("", "restic-new-password-*")
	if err != nil {
		return fmt.Errorf("failed to create password file: %w", err)
	}
	name := f.Name()
	defer os.Remove(name)
	if c.RunAs != nil {
		// restic runs as RunAs and has to read it
		if err := f.Chown(int(c.RunAs.Uid), int(c.RunAs.Gid)); err != nil {
			f.Close()
			return fmt.Errorf("failed to chown password file: %w", err)
		}
	}
	if _, err := f.WriteString(newPassword); err != nil {
		f.Close()
		return fmt.Errorf("failed to write password file: %w", err)
	}
	if err := f.Close(); err != nil {
		return fmt.Errorf("failed to write password file: %w", err)
	}
	return c.runRestic("key", "add", "--new-password-file", name)
}

// RemoveKey removes the key with id. restic refuses to remove the key c
// itself opened the repository with.
func (c *Config) RemoveKey(id string) error {
	return c.runRestic("key", "remove", id)
}

// RotateOptions are the inputs to RotatePassword
type RotateOptions struct {
	// NewPassword is the password to switch to (GeneratePassword)
	NewPassword string
	// EnvPath is the env file whose PasswordEnvKey line is rewritten
	EnvPath string
	// BackupPath gets a 0600 copy of EnvPath before it's rewritten
	BackupPath string
	// Out gets progress messages
	Out io.Writer
}

// RotateResult says how far RotatePassword got
type RotateResult struct {
	OldKeyID string
	NewKeyID string
	// EnvUpdated: EnvPath now holds NewPassword, so it must be recorded
	// (tfvars, password manager) even if RotatePassword returned an error
	EnvUpdated bool
	// OldKeyRemoved: only NewPassword opens the repository now
	OldKeyRemoved bool
}

// VerifyError is returned when the new key was added but didn't verify.
// Both keys are left in place and the env file isn't touched.
type VerifyError struct {
	Err      error
	NewKeyID string // "" if it couldn't be identified
	Recovery string
}

func (e *VerifyError) Error() string {
	return fmt.Sprintf("the new password didn't verify: %v\n\n%s", e.Err, e.Recovery)
}

func (e *VerifyError) Unwrap() error { return e.Err }

// RotatePassword replaces the repository password c.Password with
// opts.NewPassword:
//
//  1. list the keys with the old password, noting the current one
//  2. add a key for the new password
//  3. list the keys again with the old password to find the new key's ID
//  4. verify with the new password: `key list` must mark the new key
//     current and still show the old one, and `check` must pass
//  5. copy EnvPath to BackupPath and rewrite its RESTIC_PASSWORD line
//  6. remove the old key, authenticated with the new password
//
// The old key is only removed after the new password verified and the env
// file holds it, so every failure leaves a password that works in the env
// file. The env file is written before the old key goes (the reverse of the
// manual runbook) so a timer backup starting in between still has a working
// password, and a failed removal leaves two working keys instead of an env
// file with a dead password.
func (c *Config) RotatePassword(opts RotateOptions) (*RotateResult, error) {
	out := opts.Out
	if out == nil {
		out = io.Discard
	}
	res := &RotateResult{}
	if opts.NewPassword == "" || opts.NewPassword == c.Password {
		return res, fmt.Errorf("the new password must be set and differ from the old one")
	}

	fmt.Fprintln(out, "Listing keys with the current password...")
	before, err := c.ListKeys()
	if err != nil {
		return res, err
	}
	res.OldKeyID, err = currentKey(before)
	if err != nil {
		return res, err
	}
	fmt.Fprintf(out, "Current key: %s (%d key(s) in the repository)\n", res.OldKeyID, len(before))

	fmt.Fprintln(out, "Adding a key for the new password...")
	if err := c.AddKey(opts.NewPassword); err != nil {
		return res, fmt.Errorf("restic key add: %w (if a key was added anyway, it is the one not listed before; old key %s is unchanged)", err, res.OldKeyID)
	}

	verifyFail := func(err error) (*RotateResult, error) {
		return res, &VerifyError{Err: err, NewKeyID: res.NewKeyID, Recovery: c.verifyRecovery(opts.EnvPath, res)}
	}

	after, err := c.ListKeys()
	if err != nil {
		return verifyFail(err)
	}
	res.NewKeyID, err = addedKey(before, after)
	if err != nil {
		return verifyFail(err)
	}
	fmt.Fprintf(out, "New key: %s\n", res.NewKeyID)

	fmt.Fprintln(out, "Verifying with the new password: key list...")
	next := *c
	next.Password = opts.NewPassword
	keys, err := next.ListKeys()
	if err != nil {
		return verifyFail(err)
	}
	cur, err := currentKey(keys)
	if err != nil {
		return verifyFail(err)
	}
	if cur != res.NewKeyID {
		return verifyFail(fmt.Errorf("the new password opened key %s, want the new key %s", cur, res.NewKeyID))
	}
	if !hasKey(keys, res.OldKeyID) {
		return verifyFail(fmt.Errorf("old key %s missing from the new password's key list", res.OldKeyID))
	}
	fmt.Fprintln(out, "Verifying with the new password: check...")
	if err := next.Check(); err != nil {
		return verifyFail(fmt.Errorf("restic check: %w", err))
	}

	fmt.Fprintf(out, "Copying %s to %s...\n", opts.EnvPath, opts.BackupPath)
	if err := envfile.Backup(opts.EnvPath, opts.BackupPath); err != nil {
		return res, fmt.Errorf("%w\n\n%s", err, c.envRecovery(opts.EnvPath, res))
	}
	fmt.Fprintf(out, "Updating %s in %s...\n", PasswordEnvKey, opts.EnvPath)
	if err := envfile.SetKey(opts.EnvPath, PasswordEnvKey, opts.NewPassword); err != nil {
		return res, fmt.Errorf("%w\n\n%s", err, c.envRecovery(opts.EnvPath, res))
	}
	res.EnvUpdated = true

	fmt.Fprintf(out, "Removing old key %s...\n", res.OldKeyID)
	if err := next.RemoveKey(res.OldKeyID); err != nil {
		return res, fmt.Errorf("restic key remove %s: %w\n\n"+
			"%s already holds the new password, and both keys still open the repository.\n"+
			"Remove the old key by hand once the new password is saved:\n%s",
			res.OldKeyID, err, opts.EnvPath, c.manualSteps(opts.EnvPath, "restic key remove "+res.OldKeyID))
	}
	res.OldKeyRemoved = true
	return res, nil
}

// manualSteps is a shell snippet that runs cmd against the repository with
// the password from envPath
func (c *Config) manualSteps(envPath, cmd string) string {
	return fmt.Sprintf("  export RESTIC_REPOSITORY=%s\n"+
		"  export RESTIC_PASSWORD=\"$(sudo sed -n 's/^%s=//p' %s | tr -d \"\\\"'\")\"\n"+
		"  restic key list\n"+
		"  %s\n", c.Repository, PasswordEnvKey, envPath, cmd)
}

func (c *Config) verifyRecovery(envPath string, res *RotateResult) string {
	remove := "restic key remove <the key not marked current that wasn't there before>"
	if res.NewKeyID != "" {
		remove = "restic key remove " + res.NewKeyID
	}
	return fmt.Sprintf("Nothing was removed and %s was not changed: the old password still works.\n"+
		"The new key was added, though. Remove it with the old password:\n%s",
		envPath, c.manualSteps(envPath, remove))
}

func (c *Config) envRecovery(envPath string, res *RotateResult) string {
	return fmt.Sprintf("The new password verified, but %s wasn't updated, so the old key %s was\n"+
		"left in place and the old password still works. Remove the unused new key with the old password:\n%s",
		envPath, res.OldKeyID, c.manualSteps(envPath, "restic key remove "+res.NewKeyID))
}
