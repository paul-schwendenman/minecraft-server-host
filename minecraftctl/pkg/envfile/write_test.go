package envfile

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestSetKeyPreservesOtherLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "minecraft.env")
	in := "# managed by user_data\nMC_WORLD_BUCKET=bucket\n\n  # indented comment\nRESTIC_PASSWORD=old\nexport OTHER='x'\nRCON_PASSWORD=\"rc\"\n"
	if err := os.WriteFile(path, []byte(in), 0640); err != nil {
		t.Fatal(err)
	}

	if err := SetKey(path, "RESTIC_PASSWORD", "new-pw_1"); err != nil {
		t.Fatalf("SetKey() = %v", err)
	}

	got, _ := os.ReadFile(path)
	want := strings.Replace(in, "RESTIC_PASSWORD=old", "RESTIC_PASSWORD=new-pw_1", 1)
	if string(got) != want {
		t.Errorf("file =\n%q\nwant\n%q", got, want)
	}
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0640 {
		t.Errorf("mode = %v, want 0640", info.Mode().Perm())
	}
	ef, _ := Load(path)
	if v, _ := ef.Get("RESTIC_PASSWORD"); v != "new-pw_1" {
		t.Errorf("Load after SetKey: RESTIC_PASSWORD = %q", v)
	}
	// No temp files left behind
	entries, _ := os.ReadDir(filepath.Dir(path))
	if len(entries) != 1 {
		t.Errorf("dir has %d entries, want 1", len(entries))
	}
}

func TestSetKeyKeepsQuotingAndExport(t *testing.T) {
	cases := map[string]string{
		"RESTIC_PASSWORD=\"old\"\n":            "RESTIC_PASSWORD=\"new\"\n",
		"RESTIC_PASSWORD='old'\n":              "RESTIC_PASSWORD='new'\n",
		"export RESTIC_PASSWORD=old\n":         "export RESTIC_PASSWORD=new\n",
		"RESTIC_PASSWORD = old\r\nA=b":         "RESTIC_PASSWORD = new\r\nA=b",
		"RESTIC_PASSWORD=a\nRESTIC_PASSWORD=b": "RESTIC_PASSWORD=new\nRESTIC_PASSWORD=new",
		"RESTIC_PASSWORD=\n":                   "RESTIC_PASSWORD=new\n",
	}
	for in, want := range cases {
		path := filepath.Join(t.TempDir(), "env")
		if err := os.WriteFile(path, []byte(in), 0600); err != nil {
			t.Fatal(err)
		}
		if err := SetKey(path, "RESTIC_PASSWORD", "new"); err != nil {
			t.Fatalf("SetKey(%q) = %v", in, err)
		}
		got, _ := os.ReadFile(path)
		if string(got) != want {
			t.Errorf("SetKey(%q) wrote %q, want %q", in, got, want)
		}
	}
}

func TestSetKeyErrors(t *testing.T) {
	path := filepath.Join(t.TempDir(), "env")
	in := "# RESTIC_PASSWORD=commented\nRESTIC_PASSWORD_FILE=x\nQ='old'\n"
	if err := os.WriteFile(path, []byte(in), 0600); err != nil {
		t.Fatal(err)
	}
	if err := SetKey(path, "RESTIC_PASSWORD", "new"); err == nil {
		t.Error("SetKey() with no matching line = nil, want an error")
	}
	if err := SetKey(path, "Q", "it's"); err == nil {
		t.Error("SetKey() with the quote char in the value = nil, want an error")
	}
	if err := SetKey(path, "Q", "a\nB=c"); err == nil {
		t.Error("SetKey() with a newline in the value = nil, want an error")
	}
	if got, _ := os.ReadFile(path); string(got) != in {
		t.Errorf("failed SetKey changed the file: %q", got)
	}
	if err := SetKey(filepath.Join(t.TempDir(), "missing"), "Q", "x"); err == nil {
		t.Error("SetKey() on a missing file = nil, want an error")
	}
}

func TestBackup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "minecraft.env")
	bak := path + ".bak"
	if err := os.WriteFile(path, []byte("A=1\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// An old, world-readable backup is replaced and tightened
	if err := os.WriteFile(bak, []byte("old"), 0644); err != nil {
		t.Fatal(err)
	}

	if err := Backup(path, bak); err != nil {
		t.Fatalf("Backup() = %v", err)
	}
	got, _ := os.ReadFile(bak)
	if string(got) != "A=1\n" {
		t.Errorf("backup = %q", got)
	}
	info, _ := os.Stat(bak)
	if info.Mode().Perm() != 0600 {
		t.Errorf("backup mode = %v, want 0600", info.Mode().Perm())
	}
}
