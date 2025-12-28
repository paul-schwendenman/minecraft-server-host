// gendocs generates man pages and shell completions for minecraftctl
package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	cmd "github.com/paul/minecraftctl/cmd/minecraftctl"
	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/internal/version"
	"github.com/paul/minecraftctl/pkg/jars"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

func main() {
	// Default output directories
	manDir := "man/man1"
	completionsDir := "completions"

	// Parse command line arguments
	if len(os.Args) > 1 {
		manDir = os.Args[1]
	}
	if len(os.Args) > 2 {
		completionsDir = os.Args[2]
	}

	// Register all subcommands
	rootCmd := root.GetRootCmd()
	rootCmd.AddCommand(cmd.WorldCmd)
	rootCmd.AddCommand(cmd.MapCmd)
	rootCmd.AddCommand(cmd.RconCmd)
	rootCmd.AddCommand(cmd.ConfigCmd)
	rootCmd.AddCommand(jars.JarCmd)

	// Disable auto-generation tag for cleaner output
	rootCmd.DisableAutoGenTag = true

	// Generate man pages
	if err := generateManPages(rootCmd, manDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating man pages: %v\n", err)
		os.Exit(1)
	}

	// Generate shell completions
	if err := generateCompletions(rootCmd, completionsDir); err != nil {
		fmt.Fprintf(os.Stderr, "Error generating completions: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("Documentation generated successfully:\n")
	fmt.Printf("  Man pages:    %s/\n", manDir)
	fmt.Printf("  Completions:  %s/\n", completionsDir)
}

func generateManPages(rootCmd *cobra.Command, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	now := time.Now()
	header := &doc.GenManHeader{
		Title:   "MINECRAFTCTL",
		Section: "1",
		Date:    &now,
		Source:  fmt.Sprintf("minecraftctl %s", version.Version),
		Manual:  "Minecraft Server Management",
	}

	if err := doc.GenManTree(rootCmd, header, outputDir); err != nil {
		return fmt.Errorf("failed to generate man pages: %w", err)
	}

	return nil
}

func generateCompletions(rootCmd *cobra.Command, outputDir string) error {
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		return fmt.Errorf("failed to create directory %s: %w", outputDir, err)
	}

	// Generate bash completion
	bashFile := filepath.Join(outputDir, "minecraftctl.bash")
	if err := rootCmd.GenBashCompletionFile(bashFile); err != nil {
		return fmt.Errorf("failed to generate bash completion: %w", err)
	}

	// Generate zsh completion
	zshFile := filepath.Join(outputDir, "minecraftctl.zsh")
	if err := rootCmd.GenZshCompletionFile(zshFile); err != nil {
		return fmt.Errorf("failed to generate zsh completion: %w", err)
	}

	// Generate fish completion
	fishFile := filepath.Join(outputDir, "minecraftctl.fish")
	if err := rootCmd.GenFishCompletionFile(fishFile, true); err != nil {
		return fmt.Errorf("failed to generate fish completion: %w", err)
	}

	return nil
}
