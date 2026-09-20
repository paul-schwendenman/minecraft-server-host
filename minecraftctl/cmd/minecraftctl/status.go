package main

import (
	"fmt"
	"time"

	"github.com/paul/minecraftctl/pkg/status"
	"github.com/spf13/cobra"
)

var (
	statusHost    string
	statusPort    int
	statusTimeout time.Duration
)

// StatusCmd shows the server status, mirroring what the web UI displays.
var StatusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show server status (version and active players)",
	Long: `Show the status of the local Minecraft server, like the web UI does.

Pings the server for its version and online players, and shows the instance's
public IP address when running on EC2.`,
	Args: cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		out := cmd.OutOrStdout()

		s, err := status.Ping(status.Addr(statusHost, statusPort), statusTimeout)
		if err != nil {
			fmt.Fprintln(out, "Server is not running.")
			fmt.Fprintf(out, "(%v)\n", err)
			return nil
		}

		fmt.Fprintln(out, "Server is running.")
		if ip, err := status.PublicIP(500 * time.Millisecond); err == nil && ip != "" {
			fmt.Fprintf(out, "IP address: %s\n", ip)
		}
		fmt.Fprint(out, s.Format())
		return nil
	},
}

func init() {
	StatusCmd.Flags().StringVar(&statusHost, "host", "127.0.0.1", "Minecraft server host")
	StatusCmd.Flags().IntVar(&statusPort, "port", 25565, "Minecraft server port")
	StatusCmd.Flags().DurationVar(&statusTimeout, "timeout", 5*time.Second, "Ping timeout")
}
