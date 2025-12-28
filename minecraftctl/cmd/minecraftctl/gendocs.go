package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/paul/minecraftctl/cmd/minecraftctl/root"
	"github.com/paul/minecraftctl/internal/version"
	"github.com/spf13/cobra"
	"github.com/spf13/cobra/doc"
)

var gendocsCmd = &cobra.Command{
	Use:    "gendocs",
	Short:  "Generate documentation (man pages and shell completions)",
	Hidden: true,
	RunE: func(cmd *cobra.Command, args []string) error {
		manDir, _ := cmd.Flags().GetString("man-dir")
		completionsDir, _ := cmd.Flags().GetString("completions-dir")

		rootCmd := root.GetRootCmd()
		rootCmd.DisableAutoGenTag = true

		// Generate man pages
		if err := generateManPages(rootCmd, manDir); err != nil {
			return err
		}

		// Generate shell completions
		if err := generateCompletions(rootCmd, completionsDir); err != nil {
			return err
		}

		fmt.Printf("Documentation generated successfully:\n")
		fmt.Printf("  Man pages:    %s/\n", manDir)
		fmt.Printf("  Completions:  %s/\n", completionsDir)
		return nil
	},
}

func init() {
	gendocsCmd.Flags().String("man-dir", "man/man1", "Output directory for man pages")
	gendocsCmd.Flags().String("completions-dir", "completions", "Output directory for shell completions")
	root.GetRootCmd().AddCommand(gendocsCmd)
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
