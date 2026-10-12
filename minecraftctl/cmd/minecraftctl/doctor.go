package main

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/doctor"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
)

var (
	doctorStrict    bool
	doctorJSON      bool
	doctorConfigErr error
)

// DoctorCmd runs read-only health checks
var DoctorCmd = &cobra.Command{
	Use:   "doctor",
	Short: "Check the server's setup",
	Long: `Run read-only checks and print PASS, WARN, FAIL or SKIP for each:

  - the config loads
  - the worlds dir exists and is owned by minecraft (uid 25565)
  - each world has world/level.dat
  - the AMI's units and each world's timers are installed and enabled, and
    the backup timers (and the running world's map-build timer) are active
  - RCON answers, if a world is running
  - restic's settings are present and the repository is reachable
  - the uNmINeD binary is present

Exits 1 if any check fails; warnings exit 0 unless --strict. Away from a
server (no systemctl), server-only checks warn or skip instead of failing.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	// A broken config is something to report, not a reason not to run
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		doctorConfigErr = root.InitConfig(cmd)
		return nil
	},
	RunE: func(cmd *cobra.Command, args []string) error {
		env := doctor.SystemEnv(config.Get().WorldsDir, viper.ConfigFileUsed(), doctorConfigErr)
		return reportDoctor(cmd.OutOrStdout(), doctor.Run(env), doctorStrict, doctorJSON)
	},
}

// reportDoctor prints results and returns an error if doctor should exit
// non-zero
func reportDoctor(w io.Writer, results []doctor.Result, strict, asJSON bool) error {
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(results); err != nil {
			return err
		}
	} else {
		width := 0
		for _, r := range results {
			width = max(width, len(r.Name))
		}
		for _, r := range results {
			fmt.Fprintf(w, "%-4s  %-*s  %s\n", r.Status, width, r.Name, r.Detail)
		}
	}

	counts := doctor.Counts(results)
	if doctor.Failed(results, strict) {
		return fmt.Errorf("doctor: %d failed, %d warnings", counts[doctor.Fail], counts[doctor.Warn])
	}
	if !asJSON {
		fmt.Fprintf(w, "\n%d passed, %d warnings, %d skipped\n", counts[doctor.Pass], counts[doctor.Warn], counts[doctor.Skip])
	}
	return nil
}

func init() {
	DoctorCmd.Flags().BoolVar(&doctorStrict, "strict", false, "Exit 1 on warnings too")
	DoctorCmd.Flags().BoolVar(&doctorJSON, "json", false, "Print results as JSON")
}
