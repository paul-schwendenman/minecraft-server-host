package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

// configFixture makes world "w" with the given server.properties and a
// minecraft.env, and returns the env file path.
func configFixture(t *testing.T, props, env string) (worldsDir, envPath string) {
	t.Helper()
	worldsDir = t.TempDir()
	setupWorldTestConfig(t, worldsDir)
	if err := os.MkdirAll(filepath.Join(worldsDir, "w"), 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(worldsDir, "w", "server.properties"), []byte(props), 0644); err != nil {
		t.Fatal(err)
	}
	envPath = filepath.Join(t.TempDir(), "minecraft.env")
	if err := os.WriteFile(envPath, []byte(env), 0600); err != nil {
		t.Fatal(err)
	}
	return worldsDir, envPath
}

// setFlag sets a flag on a shared command and resets it after the test.
func setFlag(t *testing.T, cmd *cobra.Command, name, value string) {
	t.Helper()
	f := cmd.Flags().Lookup(name)
	orig := f.Value.String()
	if err := f.Value.Set(value); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = f.Value.Set(orig); f.Changed = false })
}

const goodEnv = "RCON_PORT=25575\nRCON_PASSWORD=hunter2\n"

func TestConfigCheckCmd(t *testing.T) {
	tests := []struct {
		name    string
		props   string
		wantErr bool
		want    []string
	}{
		{
			name:  "all good",
			props: "enable-rcon=true\nrcon.port=25575\nrcon.password=hunter2\n",
			want:  []string{"enable-rcon:   true (OK)", "rcon.port:     25575 (OK)", "rcon.password: ******** (OK)", "correctly configured"},
		},
		{
			name:    "mismatches",
			props:   "enable-rcon=false\nrcon.port=25576\nrcon.password=other\n",
			wantErr: true,
			want:    []string{"false (MISMATCH", "25576 (MISMATCH - minecraft.env has 25575)", "does not match minecraft.env", "config sync w"},
		},
		{
			name:    "missing",
			props:   "motd=hi\n",
			wantErr: true,
			want:    []string{"enable-rcon:   (missing)", "rcon.port:     (missing or invalid)", "rcon.password: (missing)"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, envPath := configFixture(t, tt.props, goodEnv)
			setFlag(t, configCheckCmd, "env-file", envPath)
			out, err := captureWorldStdout(t, func() error { return configCheckCmd.RunE(configCheckCmd, []string{"w"}) })
			if (err != nil) != tt.wantErr {
				t.Errorf("err = %v, wantErr %v", err, tt.wantErr)
			}
			for _, w := range tt.want {
				if !strings.Contains(out, w) {
					t.Errorf("output missing %q:\n%s", w, out)
				}
			}
			if strings.Contains(out, "hunter2") || strings.Contains(out, "other") {
				t.Errorf("output leaks a password:\n%s", out)
			}
		})
	}
}

func TestConfigCheckCmdErrors(t *testing.T) {
	_, envPath := configFixture(t, "enable-rcon=true\n", goodEnv)
	setFlag(t, configCheckCmd, "env-file", envPath)
	if err := configCheckCmd.RunE(configCheckCmd, []string{"missing"}); err == nil || !strings.Contains(err.Error(), "server.properties") {
		t.Errorf("missing world: %v", err)
	}
	setFlag(t, configCheckCmd, "env-file", filepath.Join(t.TempDir(), "nope.env"))
	if err := configCheckCmd.RunE(configCheckCmd, []string{"w"}); err == nil || !strings.Contains(err.Error(), "minecraft.env") {
		t.Errorf("missing env file: %v", err)
	}
}

func TestConfigSyncCmd(t *testing.T) {
	t.Setenv("RCON_PASSWORD", "")
	os.Unsetenv("RCON_PASSWORD")
	worldsDir, envPath := configFixture(t, "# comment\nmotd=hi\nrcon.port=1\nrcon.password=old\n", goodEnv)
	propsPath := filepath.Join(worldsDir, "w", "server.properties")
	setFlag(t, configSyncCmd, "env-file", envPath)

	// Dry run: reports, changes nothing
	setFlag(t, configSyncCmd, "dry-run", "true")
	out, err := captureWorldStdout(t, func() error { return configSyncCmd.RunE(configSyncCmd, []string{"w"}) })
	if err != nil {
		t.Fatalf("dry run: %v", err)
	}
	for _, w := range []string{"dry-run mode", "enable-rcon:   (unset) -> true", "rcon.port:     1 -> 25575", "rcon.password: <old> -> <new>", "Dry-run complete"} {
		if !strings.Contains(out, w) {
			t.Errorf("dry-run output missing %q:\n%s", w, out)
		}
	}
	if strings.Contains(out, "hunter2") {
		t.Errorf("output leaks the password:\n%s", out)
	}
	if data, _ := os.ReadFile(propsPath); strings.Contains(string(data), "hunter2") {
		t.Error("dry run changed server.properties")
	}

	// For real
	setFlag(t, configSyncCmd, "dry-run", "false")
	out, err = captureWorldStdout(t, func() error { return configSyncCmd.RunE(configSyncCmd, []string{"w"}) })
	if err != nil || !strings.Contains(out, "Changes saved") {
		t.Fatalf("sync: %v\n%s", err, out)
	}
	data, _ := os.ReadFile(propsPath)
	for _, w := range []string{"# comment", "motd=hi", "enable-rcon=true", "rcon.port=25575", "rcon.password=hunter2"} {
		if !strings.Contains(string(data), w) {
			t.Errorf("server.properties missing %q:\n%s", w, data)
		}
	}

	// Again: nothing to do
	out, err = captureWorldStdout(t, func() error { return configSyncCmd.RunE(configSyncCmd, []string{"w"}) })
	if err != nil || !strings.Contains(out, "already in sync") {
		t.Errorf("second sync: %v\n%s", err, out)
	}

	// No env file and no RCON_PASSWORD in the environment
	setFlag(t, configSyncCmd, "env-file", filepath.Join(t.TempDir(), "nope.env"))
	if err := configSyncCmd.RunE(configSyncCmd, []string{"w"}); err == nil {
		t.Error("sync without an env file = nil error")
	}
	// Missing world
	setFlag(t, configSyncCmd, "env-file", envPath)
	if _, err := captureWorldStdout(t, func() error { return configSyncCmd.RunE(configSyncCmd, []string{"missing"}) }); err == nil {
		t.Error("sync of a missing world = nil error")
	}
}

func TestConfigGetSetListCmds(t *testing.T) {
	worldsDir, _ := configFixture(t, "motd=hi\nrcon.password=hunter2\n", goodEnv)

	out, err := captureWorldStdout(t, func() error { return configGetCmd.RunE(configGetCmd, []string{"w", "motd"}) })
	if err != nil || out != "hi\n" {
		t.Errorf("get motd = %q, %v", out, err)
	}
	if err := configGetCmd.RunE(configGetCmd, []string{"w", "nope"}); err == nil || !strings.Contains(err.Error(), "key not found") {
		t.Errorf("get missing key: %v", err)
	}
	if err := configGetCmd.RunE(configGetCmd, []string{"missing", "motd"}); err == nil {
		t.Error("get on a missing world = nil error")
	}

	for _, kv := range [][2]string{{"motd", "hello"}, {"difficulty", "hard"}} {
		out, err = captureWorldStdout(t, func() error { return configSetCmd.RunE(configSetCmd, []string{"w", kv[0], kv[1]}) })
		if err != nil || !strings.Contains(out, "Set "+kv[0]+"="+kv[1]) {
			t.Errorf("set %s = %q, %v", kv[0], out, err)
		}
	}
	if err := configSetCmd.RunE(configSetCmd, []string{"missing", "a", "b"}); err == nil {
		t.Error("set on a missing world = nil error")
	}
	data, _ := os.ReadFile(filepath.Join(worldsDir, "w", "server.properties"))
	if !strings.Contains(string(data), "motd=hello") || !strings.Contains(string(data), "difficulty=hard") {
		t.Errorf("server.properties after set:\n%s", data)
	}

	out, err = captureWorldStdout(t, func() error { return configListCmd.RunE(configListCmd, []string{"w"}) })
	if err != nil || out != "motd=hello\nrcon.password=********\ndifficulty=hard\n" {
		t.Errorf("list = %q, %v", out, err)
	}
	setFlag(t, configListCmd, "show-passwords", "true")
	out, _ = captureWorldStdout(t, func() error { return configListCmd.RunE(configListCmd, []string{"w"}) })
	if !strings.Contains(out, "rcon.password=hunter2") {
		t.Errorf("list --show-passwords = %q", out)
	}
	if err := configListCmd.RunE(configListCmd, []string{"missing"}); err == nil {
		t.Error("list on a missing world = nil error")
	}
}
