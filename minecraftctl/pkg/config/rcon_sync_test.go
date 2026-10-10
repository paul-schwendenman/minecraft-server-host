package config

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mapEnv(m map[string]string) LookupFunc {
	return func(key string) (string, bool) {
		v, ok := m[key]
		return v, ok
	}
}

func TestSyncRcon(t *testing.T) {
	fullEnv := map[string]string{"RCON_PORT": "25575", "RCON_PASSWORD": "secret"}

	tests := []struct {
		name        string
		props       string
		env         map[string]string
		dryRun      bool
		wantChanges []Change
		wantFile    string // expected file contents afterwards; "" means unchanged
	}{
		{
			name:     "in sync",
			props:    "enable-rcon=true\nrcon.port=25575\nrcon.password=secret\n",
			env:      fullEnv,
			wantFile: "",
		},
		{
			name:        "port differs",
			props:       "enable-rcon=true\nrcon.port=25576\nrcon.password=secret\n",
			env:         fullEnv,
			wantChanges: []Change{{"rcon.port", "25576", "25575"}},
			wantFile:    "enable-rcon=true\nrcon.port=25575\nrcon.password=secret\n",
		},
		{
			name:        "password differs",
			props:       "enable-rcon=true\nrcon.port=25575\nrcon.password=old\n",
			env:         fullEnv,
			wantChanges: []Change{{"rcon.password", "old", "secret"}},
			wantFile:    "enable-rcon=true\nrcon.port=25575\nrcon.password=secret\n",
		},
		{
			name:        "enable-rcon false",
			props:       "enable-rcon=false\nrcon.port=25575\nrcon.password=secret\n",
			env:         fullEnv,
			wantChanges: []Change{{"enable-rcon", "false", "true"}},
			wantFile:    "enable-rcon=true\nrcon.port=25575\nrcon.password=secret\n",
		},
		{
			name:        "enable-rcon missing is appended",
			props:       "rcon.port=25575\nrcon.password=secret\n",
			env:         fullEnv,
			wantChanges: []Change{{"enable-rcon", "", "true"}},
			wantFile:    "rcon.port=25575\nrcon.password=secret\nenable-rcon=true\n",
		},
		{
			name:     "env missing keys leaves port and password alone",
			props:    "enable-rcon=true\nrcon.port=1\nrcon.password=whatever\n",
			env:      map[string]string{},
			wantFile: "",
		},
		{
			name:  "all three differ in order",
			props: "enable-rcon=false\nrcon.port=1\nrcon.password=old\n",
			env:   fullEnv,
			wantChanges: []Change{
				{"enable-rcon", "false", "true"},
				{"rcon.port", "1", "25575"},
				{"rcon.password", "old", "secret"},
			},
			wantFile: "enable-rcon=true\nrcon.port=25575\nrcon.password=secret\n",
		},
		{
			name:   "dry-run reports but does not write",
			props:  "enable-rcon=false\nrcon.port=1\nrcon.password=old\n",
			env:    fullEnv,
			dryRun: true,
			wantChanges: []Change{
				{"enable-rcon", "false", "true"},
				{"rcon.port", "1", "25575"},
				{"rcon.password", "old", "secret"},
			},
			wantFile: "",
		},
		{
			name: "comments order and unrelated keys preserved",
			props: "#Minecraft server properties\n#Generated on test\n" +
				"motd=Hello\nrcon.password=old\n# port comment\nrcon.port=25575\nenable-rcon=true\nmax-players=20\n",
			env:         fullEnv,
			wantChanges: []Change{{"rcon.password", "old", "secret"}},
			wantFile: "#Minecraft server properties\n#Generated on test\n" +
				"motd=Hello\nrcon.password=secret\n# port comment\nrcon.port=25575\nenable-rcon=true\nmax-players=20\n",
		},
		{
			name:     "non-int RCON_PORT is skipped",
			props:    "enable-rcon=true\nrcon.port=1\nrcon.password=secret\n",
			env:      map[string]string{"RCON_PORT": "abc", "RCON_PASSWORD": "secret"},
			wantFile: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := t.TempDir()
			propsPath := filepath.Join(dir, "server.properties")
			if err := os.WriteFile(propsPath, []byte(tt.props), 0644); err != nil {
				t.Fatalf("failed to write test properties file: %v", err)
			}

			changes, err := SyncRcon(propsPath, mapEnv(tt.env), tt.dryRun)
			if err != nil {
				t.Fatalf("SyncRcon() failed: %v", err)
			}
			if !reflect.DeepEqual(changes, tt.wantChanges) {
				t.Errorf("SyncRcon() changes = %#v, want %#v", changes, tt.wantChanges)
			}

			got, err := os.ReadFile(propsPath)
			if err != nil {
				t.Fatalf("failed to read properties file: %v", err)
			}
			want := tt.wantFile
			if want == "" {
				want = tt.props
			}
			if string(got) != want {
				t.Errorf("file contents =\n%q\nwant\n%q", got, want)
			}
		})
	}
}

func TestSyncRconMissingFile(t *testing.T) {
	_, err := SyncRcon(filepath.Join(t.TempDir(), "server.properties"), mapEnv(nil), false)
	if err == nil {
		t.Fatal("SyncRcon() expected error for missing file")
	}
	if !strings.Contains(err.Error(), "failed to load server.properties") {
		t.Errorf("SyncRcon() error = %v, want it to mention loading server.properties", err)
	}
}

func TestRconEnv(t *testing.T) {
	t.Run("readable file is used", func(t *testing.T) {
		t.Setenv("RCON_PASSWORD", "from-process")
		envPath := filepath.Join(t.TempDir(), "minecraft.env")
		if err := os.WriteFile(envPath, []byte("RCON_PASSWORD=from-file\nRCON_PORT=25575\n"), 0600); err != nil {
			t.Fatalf("failed to write env file: %v", err)
		}

		env, source, err := RconEnv(envPath)
		if err != nil {
			t.Fatalf("RconEnv() failed: %v", err)
		}
		if source != envPath {
			t.Errorf("source = %q, want %q", source, envPath)
		}
		if v, _ := env("RCON_PASSWORD"); v != "from-file" {
			t.Errorf("RCON_PASSWORD = %q, want %q", v, "from-file")
		}
	})

	t.Run("missing file falls back to process env", func(t *testing.T) {
		t.Setenv("RCON_PASSWORD", "from-process")
		envPath := filepath.Join(t.TempDir(), "missing.env")

		env, source, err := RconEnv(envPath)
		if err != nil {
			t.Fatalf("RconEnv() failed: %v", err)
		}
		if !strings.HasPrefix(source, "environment (") || !strings.Contains(source, envPath) {
			t.Errorf("source = %q, want environment label mentioning %s", source, envPath)
		}
		if v, _ := env("RCON_PASSWORD"); v != "from-process" {
			t.Errorf("RCON_PASSWORD = %q, want %q", v, "from-process")
		}
	})

	t.Run("neither file nor env is an error", func(t *testing.T) {
		// t.Setenv registers restoration; Unsetenv then removes it for this test.
		t.Setenv("RCON_PASSWORD", "")
		os.Unsetenv("RCON_PASSWORD")
		envPath := filepath.Join(t.TempDir(), "missing.env")

		if _, _, err := RconEnv(envPath); err == nil {
			t.Fatal("RconEnv() expected error when file missing and RCON_PASSWORD unset")
		}
	})
}
