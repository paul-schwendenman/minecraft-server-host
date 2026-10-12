package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/pkg/envfile"
)

// rotateKeyRestic keeps "id password" lines in $FAKE_STATE/keys and logs each
// call to $FAKE_STATE/log (see pkg/backup's fake for the full version)
const rotateKeyRestic = `#!/bin/sh
state="$FAKE_STATE"
echo "$*" >> "$state/log"
cur=$(awk -v pw="$RESTIC_PASSWORD" '$2 == pw { print $1 }' "$state/keys")
[ -n "$cur" ] || exit 12
case "$1 $2" in
"key list")
	printf '['; sep=''
	while read -r id pw; do
		c=false; [ "$id" = "$cur" ] && c=true
		printf '%s{"current":%s,"id":"%s"}' "$sep" "$c" "$id"; sep=','
	done < "$state/keys"
	printf ']\n';;
"key add") echo "new $(cat "$4")" >> "$state/keys";;
"key remove") grep -v "^$3 " "$state/keys" > "$state/k"; mv "$state/k" "$state/keys";;
"check ") ;;
*) exit 3;;
esac
`

type rotateCmdFixture struct {
	state, env string
	out        bytes.Buffer
}

const rotateCmdEnv = "# user_data\nMC_WORLD_BUCKET=bucket\nRESTIC_PASSWORD=\"oldpw\"\n"

func setupRotateCmd(t *testing.T, euid int, tty bool) *rotateCmdFixture {
	t.Helper()
	f := &rotateCmdFixture{state: t.TempDir()}
	bin := t.TempDir()
	if err := os.WriteFile(filepath.Join(bin, "restic"), []byte(rotateKeyRestic), 0755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", bin+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("FAKE_STATE", f.state)
	if err := os.WriteFile(filepath.Join(f.state, "keys"), []byte("old oldpw\n"), 0600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("MC_WORLD_BUCKET", "bucket")
	t.Setenv("RESTIC_PASSWORD", "oldpw")
	t.Setenv("RESTIC_PASSWORD_FILE", "")
	t.Setenv("RESTIC_PASSWORD_COMMAND", "")
	t.Setenv("MC_RESTIC_RETRY_LOCK", "0")

	f.env = filepath.Join(t.TempDir(), "minecraft.env")
	if err := os.WriteFile(f.env, []byte(rotateCmdEnv), 0600); err != nil {
		t.Fatal(err)
	}
	rootFlags := root.GetRootCmd().PersistentFlags()
	if err := rootFlags.Set("env-file", f.env); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = rootFlags.Set("env-file", "") })

	origEuid, origTTY, origConfirm := rotateGeteuid, rotateStdinIsTerminal, rotateConfirm
	rotateGeteuid = func() int { return euid }
	rotateStdinIsTerminal = func() bool { return tty }
	rotateConfirm = func(string) (bool, error) {
		t.Error("unexpected confirmation prompt")
		return false, nil
	}
	t.Cleanup(func() { rotateGeteuid, rotateStdinIsTerminal, rotateConfirm = origEuid, origTTY, origConfirm })

	backupRotatePasswordCmd.SetOut(&f.out)
	t.Cleanup(func() {
		backupRotatePasswordCmd.SetOut(nil)
		_ = backupRotatePasswordCmd.Flags().Set("yes", "false")
		_ = backupRotatePasswordCmd.Flags().Set("dry-run", "false")
	})
	return f
}

func (f *rotateCmdFixture) run(t *testing.T, flags ...string) error {
	t.Helper()
	for _, name := range flags {
		if err := backupRotatePasswordCmd.Flags().Set(name, "true"); err != nil {
			t.Fatal(err)
		}
	}
	return backupRotatePasswordCmd.RunE(backupRotatePasswordCmd, nil)
}

func (f *rotateCmdFixture) assertUntouched(t *testing.T) {
	t.Helper()
	if _, err := os.Stat(filepath.Join(f.state, "log")); !os.IsNotExist(err) {
		log, _ := os.ReadFile(filepath.Join(f.state, "log"))
		t.Errorf("restic ran:\n%s", log)
	}
	if env, _ := os.ReadFile(f.env); string(env) != rotateCmdEnv {
		t.Errorf("env file changed: %q", env)
	}
	if _, err := os.Stat(f.env + ".bak"); !os.IsNotExist(err) {
		t.Errorf("backup env file written (err %v)", err)
	}
}

func TestRotatePasswordDryRun(t *testing.T) {
	// Not root, no terminal: a dry run needs neither
	f := setupRotateCmd(t, 1000, false)

	if err := f.run(t, "dry-run"); err != nil {
		t.Fatalf("rotate-password --dry-run = %v", err)
	}
	got := f.out.String()
	for _, want := range []string{"Dry run", "restic key add", "restic check", "restic key remove", f.env + ".bak"} {
		if !strings.Contains(got, want) {
			t.Errorf("dry-run output missing %q:\n%s", want, got)
		}
	}
	f.assertUntouched(t)
}

func TestRotatePasswordRequiresRoot(t *testing.T) {
	f := setupRotateCmd(t, 1000, true)

	err := f.run(t, "yes")
	if err == nil || !strings.Contains(err.Error(), "root") {
		t.Fatalf("rotate-password as non-root = %v, want a root error", err)
	}
	f.assertUntouched(t)
}

func TestRotatePasswordRefusesWithoutYesWhenNonInteractive(t *testing.T) {
	f := setupRotateCmd(t, 0, false)

	err := f.run(t)
	if err == nil || !strings.Contains(err.Error(), "--yes") {
		t.Fatalf("rotate-password without --yes and no tty = %v, want a --yes error", err)
	}
	f.assertUntouched(t)
}

func TestRotatePasswordDeclinedPrompt(t *testing.T) {
	f := setupRotateCmd(t, 0, true)
	asked := false
	rotateConfirm = func(string) (bool, error) { asked = true; return false, nil }

	if err := f.run(t); err != nil {
		t.Fatalf("declined rotate-password = %v", err)
	}
	if !asked {
		t.Error("no confirmation prompt")
	}
	f.assertUntouched(t)
}

func TestRotatePasswordRefusesMismatchedPassword(t *testing.T) {
	f := setupRotateCmd(t, 0, false)
	t.Setenv("RESTIC_PASSWORD", "something-else")

	if err := f.run(t, "yes"); err == nil || !strings.Contains(err.Error(), "differs") {
		t.Fatalf("rotate-password with env/file mismatch = %v", err)
	}
	f.assertUntouched(t)
}

func TestRotatePasswordWithYes(t *testing.T) {
	f := setupRotateCmd(t, 0, false)

	if err := f.run(t, "yes"); err != nil {
		t.Fatalf("rotate-password --yes = %v\n%s", err, f.out.String())
	}

	ef, err := envfile.Load(f.env)
	if err != nil {
		t.Fatal(err)
	}
	newPw, _ := ef.Get("RESTIC_PASSWORD")
	if newPw == "oldpw" || len(newPw) != 43 {
		t.Fatalf("env RESTIC_PASSWORD = %q, want a new 43-char password", newPw)
	}
	env, _ := os.ReadFile(f.env)
	if want := strings.Replace(rotateCmdEnv, `"oldpw"`, `"`+newPw+`"`, 1); string(env) != want {
		t.Errorf("env file = %q, want %q", env, want)
	}
	if bak, _ := os.ReadFile(f.env + ".bak"); string(bak) != rotateCmdEnv {
		t.Errorf("backup = %q", bak)
	}
	keys, _ := os.ReadFile(filepath.Join(f.state, "keys"))
	if string(keys) != "new "+newPw+"\n" {
		t.Errorf("keys = %q, want only the new key", keys)
	}
	got := f.out.String()
	if strings.Count(got, newPw) != 1 || !strings.Contains(got, "terraform.tfvars") {
		t.Errorf("output should show the password once with the tfvars reminder:\n%s", got)
	}
}
