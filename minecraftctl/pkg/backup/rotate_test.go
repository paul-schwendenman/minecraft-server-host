package backup

import (
	"bytes"
	"encoding/base64"
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// fakeKeyRestic is a restic that keeps its keys in a file, one "id password"
// per line, and logs "password: args" for every call. It authenticates with
// $RESTIC_PASSWORD like restic does and refuses to remove the current key.
// FAKE_CHECK_FAIL (a password) makes check fail when opened with it.
const fakeKeyRestic = `
state="$FAKE_STATE"
echo "$RESTIC_PASSWORD: $*" >> "$state/log"
cur=$(awk -v pw="$RESTIC_PASSWORD" '$2 == pw { print $1 }' "$state/keys")
if [ -z "$cur" ]; then echo "Fatal: wrong password or no key found" >&2; exit 12; fi
case "$1 $2" in
"key list")
	printf '['
	sep=''
	while read -r id pw; do
		c=false; [ "$id" = "$cur" ] && c=true
		printf '%s{"current":%s,"id":"%s","userName":"minecraft","hostName":"h","created":"now"}' "$sep" "$c" "$id"
		sep=','
	done < "$state/keys"
	printf ']\n'
	;;
"key add")
	[ "$3" = "--new-password-file" ] || exit 2
	[ -n "$FAKE_ADD_FAIL" ] && exit 1
	cp "$4" "$state/pwfile-copy"
	n=$(wc -l < "$state/keys" | tr -d ' ')
	echo "key$((n + 1)) $(cat "$4")" >> "$state/keys"
	echo "saved new key"
	;;
"key remove")
	if [ "$3" = "$cur" ]; then echo "refusing to remove current key" >&2; exit 1; fi
	[ -n "$FAKE_REMOVE_FAIL" ] && exit 1
	grep -v "^$3 " "$state/keys" > "$state/keys.new"; mv "$state/keys.new" "$state/keys"
	;;
"check ")
	if [ "$RESTIC_PASSWORD" = "$FAKE_CHECK_FAIL" ]; then echo "check failed" >&2; exit 1; fi
	;;
*) exit 3;;
esac
`

type rotateFixture struct {
	state, env, bak string
	cfg             *Config
}

const rotateEnv = "# written by user_data\nMC_WORLD_BUCKET=bucket\nRESTIC_PASSWORD=oldpw\n\n# rcon\nRCON_PASSWORD=x\n"

func newRotateFixture(t *testing.T) *rotateFixture {
	t.Helper()
	fakeRestic(t, fakeKeyRestic)
	f := &rotateFixture{state: t.TempDir()}
	t.Setenv("FAKE_STATE", f.state)
	t.Setenv("FAKE_CHECK_FAIL", "")
	t.Setenv("FAKE_ADD_FAIL", "")
	t.Setenv("FAKE_REMOVE_FAIL", "")
	if err := os.WriteFile(filepath.Join(f.state, "keys"), []byte("other otherpw\nold1 oldpw\n"), 0600); err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	f.env = filepath.Join(dir, "minecraft.env")
	f.bak = f.env + ".bak"
	if err := os.WriteFile(f.env, []byte(rotateEnv), 0640); err != nil {
		t.Fatal(err)
	}
	f.cfg = &Config{Repository: "s3:example/bucket", Password: "oldpw"}
	return f
}

func (f *rotateFixture) opts(out io.Writer) RotateOptions {
	return RotateOptions{NewPassword: "newpw", EnvPath: f.env, BackupPath: f.bak, Out: out}
}

func (f *rotateFixture) log(t *testing.T) []string {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(f.state, "log"))
	if err != nil {
		t.Fatal(err)
	}
	return strings.Split(strings.TrimSpace(string(data)), "\n")
}

func (f *rotateFixture) keys(t *testing.T) string {
	t.Helper()
	data, _ := os.ReadFile(filepath.Join(f.state, "keys"))
	return string(data)
}

func TestRotatePasswordHappyPath(t *testing.T) {
	f := newRotateFixture(t)
	var out bytes.Buffer

	res, err := f.cfg.RotatePassword(f.opts(&out))
	if err != nil {
		t.Fatalf("RotatePassword() = %v\n%s", err, out.String())
	}
	if res.OldKeyID != "old1" || res.NewKeyID != "key3" || !res.EnvUpdated || !res.OldKeyRemoved {
		t.Errorf("result = %+v", res)
	}

	wantLog := []string{
		"oldpw: key list --json",
		"oldpw: key add --new-password-file *",
		"oldpw: key list --json",
		"newpw: key list --json",
		"newpw: check",
		"newpw: key remove old1",
	}
	got := f.log(t)
	if len(got) != len(wantLog) {
		t.Fatalf("restic calls =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(wantLog, "\n"))
	}
	for i, want := range wantLog {
		if got[i] != want && !(strings.HasSuffix(want, "*") && strings.HasPrefix(got[i], strings.TrimSuffix(want, "*"))) {
			t.Errorf("call %d = %q, want %q", i, got[i], want)
		}
	}

	if k := f.keys(t); k != "other otherpw\nkey3 newpw\n" {
		t.Errorf("keys after rotation = %q", k)
	}
	env, _ := os.ReadFile(f.env)
	if want := strings.Replace(rotateEnv, "RESTIC_PASSWORD=oldpw", "RESTIC_PASSWORD=newpw", 1); string(env) != want {
		t.Errorf("env file = %q, want %q", env, want)
	}
	if info, _ := os.Stat(f.env); info.Mode().Perm() != 0640 {
		t.Errorf("env mode = %v, want 0640", info.Mode().Perm())
	}
	bak, _ := os.ReadFile(f.bak)
	if string(bak) != rotateEnv {
		t.Errorf("backup = %q, want the original file", bak)
	}
	if info, _ := os.Stat(f.bak); info.Mode().Perm() != 0600 {
		t.Errorf("backup mode = %v, want 0600", info.Mode().Perm())
	}

	// The password file restic read held the new password and is gone
	if pw, _ := os.ReadFile(filepath.Join(f.state, "pwfile-copy")); string(pw) != "newpw" {
		t.Errorf("password file held %q", pw)
	}
	addArgs := strings.Fields(got[1])
	if _, err := os.Stat(addArgs[len(addArgs)-1]); !os.IsNotExist(err) {
		t.Errorf("password file %s still exists (err %v)", addArgs[len(addArgs)-1], err)
	}
	if strings.Contains(out.String(), "newpw") {
		t.Errorf("progress output leaks the password:\n%s", out.String())
	}
}

func TestRotatePasswordVerifyFailureKeepsOldKeyAndEnv(t *testing.T) {
	f := newRotateFixture(t)
	t.Setenv("FAKE_CHECK_FAIL", "newpw")
	var out bytes.Buffer

	res, err := f.cfg.RotatePassword(f.opts(&out))
	var verr *VerifyError
	if !errors.As(err, &verr) {
		t.Fatalf("RotatePassword() = %v, want a VerifyError", err)
	}
	if verr.NewKeyID != "key3" || !strings.Contains(err.Error(), "restic key remove key3") {
		t.Errorf("recovery doesn't name the new key:\n%v", err)
	}
	if res.EnvUpdated || res.OldKeyRemoved {
		t.Errorf("result = %+v, want nothing updated or removed", res)
	}
	for _, call := range f.log(t) {
		if strings.Contains(call, "key remove") {
			t.Errorf("a key was removed: %q", call)
		}
	}
	if k := f.keys(t); !strings.Contains(k, "old1 oldpw") {
		t.Errorf("old key gone: %q", k)
	}
	if env, _ := os.ReadFile(f.env); string(env) != rotateEnv {
		t.Errorf("env file changed: %q", env)
	}
	if _, err := os.Stat(f.bak); !os.IsNotExist(err) {
		t.Errorf("backup written on a failed verify (err %v)", err)
	}
}

func TestRotatePasswordAddFailure(t *testing.T) {
	f := newRotateFixture(t)
	t.Setenv("FAKE_ADD_FAIL", "1")

	res, err := f.cfg.RotatePassword(f.opts(nil))
	if err == nil || res.EnvUpdated {
		t.Fatalf("RotatePassword() = %+v, %v; want an error and no env change", res, err)
	}
	if env, _ := os.ReadFile(f.env); string(env) != rotateEnv {
		t.Errorf("env file changed: %q", env)
	}
}

func TestRotatePasswordRemoveFailureStillUpdatesEnv(t *testing.T) {
	f := newRotateFixture(t)
	t.Setenv("FAKE_REMOVE_FAIL", "1")

	res, err := f.cfg.RotatePassword(f.opts(nil))
	if err == nil {
		t.Fatal("RotatePassword() = nil, want the remove error")
	}
	if !res.EnvUpdated || res.OldKeyRemoved {
		t.Errorf("result = %+v, want env updated and old key kept", res)
	}
	if !strings.Contains(err.Error(), "restic key remove old1") {
		t.Errorf("error doesn't say how to remove the old key:\n%v", err)
	}
	if k := f.keys(t); !strings.Contains(k, "old1 oldpw") || !strings.Contains(k, "key3 newpw") {
		t.Errorf("keys = %q, want both", k)
	}
}

func TestRotatePasswordRejectsSamePassword(t *testing.T) {
	f := newRotateFixture(t)
	opts := f.opts(nil)
	opts.NewPassword = "oldpw"
	if _, err := f.cfg.RotatePassword(opts); err == nil {
		t.Fatal("RotatePassword() with the old password = nil, want an error")
	}
	if _, err := os.Stat(filepath.Join(f.state, "log")); !os.IsNotExist(err) {
		t.Error("restic ran")
	}
}

func TestCurrentKey(t *testing.T) {
	keys := []Key{{ID: "a"}, {ID: "b", Current: true}, {ID: "c"}}
	if id, err := currentKey(keys); err != nil || id != "b" {
		t.Errorf("currentKey() = %q, %v; want b", id, err)
	}
	if _, err := currentKey([]Key{{ID: "a"}}); err == nil {
		t.Error("currentKey() with none current = nil error")
	}
	if _, err := currentKey([]Key{{ID: "a", Current: true}, {ID: "b", Current: true}}); err == nil {
		t.Error("currentKey() with two current = nil error")
	}
}

func TestAddedKey(t *testing.T) {
	before := []Key{{ID: "a"}, {ID: "b"}}
	if id, err := addedKey(before, []Key{{ID: "b"}, {ID: "n"}, {ID: "a"}}); err != nil || id != "n" {
		t.Errorf("addedKey() = %q, %v; want n", id, err)
	}
	if _, err := addedKey(before, before); err == nil {
		t.Error("addedKey() with nothing added = nil error")
	}
	if _, err := addedKey(before, []Key{{ID: "a"}, {ID: "x"}, {ID: "y"}}); err == nil {
		t.Error("addedKey() with two added = nil error")
	}
}

func TestGeneratePassword(t *testing.T) {
	a, err := GeneratePassword()
	if err != nil {
		t.Fatal(err)
	}
	b, _ := GeneratePassword()
	if a == b {
		t.Error("two passwords are equal")
	}
	raw, err := base64.RawURLEncoding.DecodeString(a)
	if err != nil || len(raw) != 32 {
		t.Errorf("password %q isn't 32 bytes of unpadded base64url (err %v)", a, err)
	}
}
