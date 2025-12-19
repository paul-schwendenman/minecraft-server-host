package main

import (
	"fmt"
	"os"
	"text/tabwriter"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/jars"
	"github.com/spf13/cobra"
)

var jarCmd = &cobra.Command{
	Use:   "jar",
	Short: "Manage Minecraft server JARs",
	Long:  "Commands for downloading, listing, and verifying Minecraft server JAR files",
}

var jarListCmd = &cobra.Command{
	Use:   "list",
	Short: "List all installed JARs",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Get()
		jarList, err := jars.ListJars(cfg.JarsDir)
		if err != nil {
			return err
		}

		if len(jarList) == 0 {
			fmt.Println("No JARs found")
			return nil
		}

		w := tabwriter.NewWriter(os.Stdout, 0, 0, 2, ' ', 0)
		fmt.Fprintln(w, "VERSION\tSIZE\tCHECKSUM\tINSTALLED")
		for _, jar := range jarList {
			sizeStr := formatSize(jar.Size)
			checksumStr := jar.Checksum[:8] + "..."
			installedStr := jar.InstalledAt.Format("2006-01-02 15:04")
			fmt.Fprintf(w, "%s\t%s\t%s\t%s\n", jar.Version, sizeStr, checksumStr, installedStr)
		}
		w.Flush()

		return nil
	},
}

var (
	downloadURL    string
	downloadSHA256 string
)

var jarDownloadCmd = &cobra.Command{
	Use:   "download <version>",
	Short: "Download a Minecraft server JAR",
	Long:  "Download a Minecraft server JAR from a URL with optional checksum verification",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		cfg := config.Get()

		if downloadURL == "" {
			return fmt.Errorf("--url is required")
		}

		// If --sha256 not provided, try to load from checksums.txt
		expectedSHA256 := downloadSHA256
		if expectedSHA256 == "" {
			checksums, err := jars.LoadChecksums(cfg.JarsDir)
			if err == nil {
				jarFileName := fmt.Sprintf("minecraft_server_%s.jar", version)
				if checksum, ok := checksums[jarFileName]; ok {
					expectedSHA256 = checksum
					fmt.Printf("Using checksum from checksums.txt: %s\n", checksum[:16]+"...")
				}
			}
		}

		if err := jars.DownloadJar(version, downloadURL, cfg.JarsDir, expectedSHA256); err != nil {
			return err
		}

		fmt.Printf("JAR %s downloaded successfully\n", version)
		return nil
	},
}

var jarVerifyCmd = &cobra.Command{
	Use:   "verify <version>",
	Short: "Verify a JAR's checksum",
	Long:  "Verify a JAR's checksum against the checksums.txt file",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		cfg := config.Get()

		if err := jars.VerifyJar(version, cfg.JarsDir); err != nil {
			return err
		}

		fmt.Printf("JAR %s checksum verified\n", version)
		return nil
	},
}

var jarInfoCmd = &cobra.Command{
	Use:   "info <version>",
	Short: "Show detailed information about a JAR",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		version := args[0]
		cfg := config.Get()

		info, err := jars.GetJarInfo(version, cfg.JarsDir)
		if err != nil {
			return err
		}

		fmt.Printf("Version: %s\n", info.Version)
		fmt.Printf("Path: %s\n", info.Path)
		fmt.Printf("Size: %s\n", formatSize(info.Size))
		fmt.Printf("Checksum: %s\n", info.Checksum)
		fmt.Printf("Installed: %s\n", info.InstalledAt.Format("2006-01-02 15:04:05"))

		return nil
	},
}

func init() {
	jarCmd.AddCommand(jarListCmd)
	jarCmd.AddCommand(jarDownloadCmd)
	jarCmd.AddCommand(jarVerifyCmd)
	jarCmd.AddCommand(jarInfoCmd)

	jarDownloadCmd.Flags().StringVar(&downloadURL, "url", "", "URL to download the JAR from (required)")
	jarDownloadCmd.MarkFlagRequired("url")
	jarDownloadCmd.Flags().StringVar(&downloadSHA256, "sha256", "", "Expected SHA256 checksum (optional, will use checksums.txt if available)")
}

// formatSize formats a file size in human-readable format
func formatSize(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
