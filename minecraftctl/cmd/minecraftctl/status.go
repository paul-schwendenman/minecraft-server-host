package main

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/paul/minecraftctl/pkg/status"
	"github.com/paul/minecraftctl/pkg/systemd"
	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/spf13/cobra"
)

var (
	statusHost    string
	statusPort    int
	statusTimeout time.Duration
	statusJSON    bool
)

// StatusCmd shows the server status, mirroring what the web UI displays.
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status (version and active players)",
	Long: `Show the status of the local Minecraft server, like the web UI does.

Pings the server for its version and online players, and shows which world is
running (from systemd) and the instance's public IP address when running on
EC2.

With --json, prints a machine-readable document: "instance" and "dns_record"
have the same shape as the control API's /status response (dns_record values
are always null since DNS isn't visible from the server, and active_world is
the world actually running rather than the ActiveWorld tag), and "details" is the
raw server response, as returned by the details API (null if the server didn't
respond, in which case "error" says why).`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		s, err := status.Ping(status.Addr(statusHost, statusPort), statusTimeout)
		world := runningWorld()

		if statusJSON {
			ip, _ := status.PublicIP(500 * time.Millisecond)
			enc := json.NewEncoder(out)
			enc.SetIndent("", "  ")
			return enc.Encode(status.NewReport(s, err, ip, world))
		}

		if err != nil {
			if world != "" {
				// The unit is up but Minecraft isn't answering: still loading the
				// world, or saving it on the way down.
				fmt.Fprintf(out, "World %s is %s, not answering yet.\n", world, worldUnitState(world))
			} else {
				fmt.Fprintln(out, "Server is not running.")
			}
			fmt.Fprintf(out, "(%v)\n", err)
			return nil
		}

		fmt.Fprintln(out, "Server is running.")
		if world != "" {
			fmt.Fprintf(out, "World: %s\n", world)
		}
		if ip, err := status.PublicIP(500 * time.Millisecond); err == nil && ip != "" {
			fmt.Fprintf(out, "IP address: %s\n", ip)
		}
		fmt.Fprint(out, s.Format())
		return nil
	},
}

// runningWorld names the world(s) whose minecraft@ unit is up, or "" if none
// is (or systemd can't be asked, e.g. off the server).
func runningWorld() string {
	names, err := worlds.RunningWorlds()
	if err != nil {
		return ""
	}
	return strings.Join(names, ", ")
}

// worldUnitState describes a world's unit that isn't answering pings.
func worldUnitState(world string) string {
	unit := systemd.FormatUnitName("minecraft", strings.Split(world, ", ")[0], systemd.UnitService)
	if systemd.GetActiveState(unit) == "deactivating" {
		return "stopping"
	}
	return "starting"
}

func init() {
	StatusCmd.Flags().StringVar(&statusHost, "host", "127.0.0.1", "Minecraft server host")
	StatusCmd.Flags().IntVar(&statusPort, "port", 25565, "Minecraft server port")
	StatusCmd.Flags().BoolVar(&statusJSON, "json", false, "Output machine-readable JSON")
	StatusCmd.Flags().DurationVar(&statusTimeout, "timeout", 5*time.Second, "Ping timeout")
}
