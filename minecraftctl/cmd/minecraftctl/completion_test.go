package main

import (
	"slices"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/pkg/config"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

// fileCompletionAllowlist lists commands that intentionally fall back to
// the shell's file completion, with the reason
var fileCompletionAllowlist = map[string]string{
	"minecraftctl rcon exec": "takes a path to a file of RCON commands",
}

// setupCompletionTestConfig points worlds_dir and jars_dir at temp dirs
// holding two worlds and two JARs
func setupCompletionTestConfig(t *testing.T) {
	t.Helper()
	worldsDir := t.TempDir()
	jarsDir := t.TempDir()
	viper.Reset()
	viper.Set("worlds_dir", worldsDir)
	viper.Set("jars_dir", jarsDir)
	if err := config.Init(""); err != nil {
		t.Fatalf("config.Init() failed: %v", err)
	}
	createTestWorld(t, worldsDir, "alpha", false)
	createTestWorld(t, worldsDir, "beta", false)
	createTestJarFile(t, jarsDir, "1.21.1", "a")
	createTestJarFile(t, jarsDir, "1.20.4", "b")
}

func runnableCommands(cmd *cobra.Command) []*cobra.Command {
	var out []*cobra.Command
	if cmd.Runnable() {
		out = append(out, cmd)
	}
	for _, c := range cmd.Commands() {
		out = append(out, runnableCommands(c)...)
	}
	return out
}

// takesNoArgs reports whether the command's usage line declares no
// positional arguments
func takesNoArgs(cmd *cobra.Command) bool {
	return !strings.ContainsAny(cmd.Use, "<[")
}

func TestCompletionNoFileFallback(t *testing.T) {
	setupCompletionTestConfig(t)

	cmds := runnableCommands(root.GetRootCmd())
	if len(cmds) < 20 {
		t.Fatalf("found only %d runnable commands; is the tree registered?", len(cmds))
	}

	for _, cmd := range cmds {
		path := cmd.CommandPath()
		if cmd == root.GetRootCmd() {
			continue
		}
		t.Run(path, func(t *testing.T) {
			if takesNoArgs(cmd) {
				if cmd.Args == nil || cmd.Args(cmd, []string{"extra"}) == nil {
					t.Errorf("%s takes no arguments but accepts one; set Args: cobra.NoArgs", path)
				}
			}

			if _, ok := fileCompletionAllowlist[path]; ok {
				return
			}
			if cmd.ValidArgsFunction == nil && len(cmd.ValidArgs) == 0 {
				t.Fatalf("%s has no ValidArgsFunction, so the shell falls back to file completion", path)
			}
			if cmd.ValidArgsFunction == nil {
				return
			}
			_, directive := cmd.ValidArgsFunction(cmd, nil, "")
			if directive&cobra.ShellCompDirectiveNoFileComp == 0 {
				t.Errorf("%s completion directive %d allows file completion", path, directive)
			}
		})
	}
}

func TestBackupListCompletion(t *testing.T) {
	setupCompletionTestConfig(t)

	got, _ := backupListCmd.ValidArgsFunction(backupListCmd, nil, "")
	for _, want := range []string{"all", "alpha", "beta"} {
		if !slices.Contains(got, want) {
			t.Errorf("backup list completions = %v, missing %q", got, want)
		}
	}

	got, _ = backupListCmd.ValidArgsFunction(backupListCmd, []string{"all"}, "")
	if len(got) != 0 {
		t.Errorf("backup list second-arg completions = %v, want none", got)
	}
}

func TestSingleArgCommandsStopCompleting(t *testing.T) {
	setupCompletionTestConfig(t)

	cmds := []*cobra.Command{
		backupCreateCmd, backupRestoreCmd, worldInfoCmd, worldStartCmd,
		worldSwitchCmd, mapConfigCatCmd, configCheckCmd, jarInfoCmd, jarVerifyCmd,
	}
	for _, cmd := range cmds {
		first, _ := cmd.ValidArgsFunction(cmd, nil, "")
		if len(first) == 0 {
			t.Errorf("%s: expected completions for the first argument", cmd.CommandPath())
		}
		got, directive := cmd.ValidArgsFunction(cmd, []string{first[0]}, "")
		if len(got) != 0 {
			t.Errorf("%s: second-arg completions = %v, want none", cmd.CommandPath(), got)
		}
		if directive&cobra.ShellCompDirectiveNoFileComp == 0 {
			t.Errorf("%s: second-arg completion allows files", cmd.CommandPath())
		}
	}
}

func TestMultiWorldCompletionSkipsGivenWorlds(t *testing.T) {
	setupCompletionTestConfig(t)

	for _, cmd := range []*cobra.Command{mapBuildNowCmd, mapManifestCmd} {
		got, _ := cmd.ValidArgsFunction(cmd, []string{"alpha"}, "")
		if !slices.Equal(got, []string{"beta"}) {
			t.Errorf("%s completions after alpha = %v, want [beta]", cmd.CommandPath(), got)
		}
	}
}

func TestJarVersionCompletion(t *testing.T) {
	setupCompletionTestConfig(t)

	got, _ := jarInfoCmd.ValidArgsFunction(jarInfoCmd, nil, "")
	slices.Sort(got)
	if !slices.Equal(got, []string{"1.20.4", "1.21.1"}) {
		t.Errorf("jar info completions = %v, want [1.20.4 1.21.1]", got)
	}
}

func TestMapCleanCompletion(t *testing.T) {
	setupCompletionTestConfig(t)
	createTestWorld(t, config.Get().WorldsDir, "gamma", true)

	got, _ := mapCleanCmd.ValidArgsFunction(mapCleanCmd, nil, "")
	if !slices.Contains(got, "gamma") {
		t.Errorf("map clean first-arg completions = %v, missing gamma", got)
	}

	got, directive := mapCleanCmd.ValidArgsFunction(mapCleanCmd, []string{"gamma"}, "")
	if !slices.Equal(got, []string{"overworld"}) || directive&cobra.ShellCompDirectiveNoFileComp == 0 {
		t.Errorf("map clean map completions = %v (directive %d), want [overworld] without files", got, directive)
	}

	got, _ = mapCleanCmd.ValidArgsFunction(mapCleanCmd, []string{"gamma", "overworld"}, "")
	if len(got) != 0 {
		t.Errorf("map clean third-arg completions = %v, want none", got)
	}
}
