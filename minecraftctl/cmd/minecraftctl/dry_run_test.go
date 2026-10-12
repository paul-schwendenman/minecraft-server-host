package main

import (
	"bytes"
	"os"
	"strings"
	"testing"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/doctor"
	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/spf13/viper"
)

func TestMapBuildNowDryRun(t *testing.T) {
	worldsDir := t.TempDir()
	mapsDir := t.TempDir()
	viper.Reset()
	t.Cleanup(viper.Reset)
	viper.Set("worlds_dir", worldsDir)
	viper.Set("maps_dir", mapsDir)
	if err := config.Init(""); err != nil {
		t.Fatalf("config.Init() = %v", err)
	}
	createTestWorld(t, worldsDir, "alpha", true)

	if err := mapBuildNowCmd.Flags().Set("dry-run", "true"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = mapBuildNowCmd.Flags().Set("dry-run", "false") })

	var out bytes.Buffer
	mapBuildNowCmd.SetOut(&out)
	t.Cleanup(func() { mapBuildNowCmd.SetOut(nil) })
	if err := mapBuildNowCmd.RunE(mapBuildNowCmd, []string{"alpha"}); err != nil {
		t.Fatalf("map build now --dry-run = %v", err)
	}

	got := out.String()
	if !strings.Contains(got, "alpha:") || !strings.Contains(got, "overworld: full (no rendered imageformat found") {
		t.Errorf("output doesn't say overworld gets a full render:\n%s", got)
	}
	entries, err := os.ReadDir(mapsDir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 0 {
		t.Errorf("dry run wrote to the maps dir: %v", entries)
	}
}

func TestPrintCreatePlan(t *testing.T) {
	var out bytes.Buffer
	printCreatePlan(&out, &worlds.CreatePlan{
		World: "fresh",
		Steps: []worlds.CreateStep{{Desc: "create directory /srv/fresh"}, {Desc: "write /srv/fresh/eula.txt"}},
	})
	want := "Dry run: would create world 'fresh':\n  create directory /srv/fresh\n  write /srv/fresh/eula.txt\n"
	if out.String() != want {
		t.Errorf("printCreatePlan() = %q, want %q", out.String(), want)
	}
}

func TestReportDoctor(t *testing.T) {
	warn := []doctor.Result{
		{Status: doctor.Pass, Name: "config", Detail: "ok"},
		{Status: doctor.Warn, Name: "unmined", Detail: "missing"},
	}
	var out bytes.Buffer
	if err := reportDoctor(&out, warn, false, false); err != nil {
		t.Errorf("warnings without --strict = %v, want nil", err)
	}
	if !strings.Contains(out.String(), "WARN  unmined  missing") {
		t.Errorf("output = %q", out.String())
	}
	if err := reportDoctor(&bytes.Buffer{}, warn, true, false); err == nil {
		t.Error("warnings with --strict = nil, want error")
	}

	fail := append(warn, doctor.Result{Status: doctor.Fail, Name: "rcon", Detail: "refused"})
	out.Reset()
	if err := reportDoctor(&out, fail, false, true); err == nil {
		t.Error("a failure = nil, want error")
	}
	if !strings.Contains(out.String(), `"status": "FAIL"`) {
		t.Errorf("JSON output = %q", out.String())
	}
}
