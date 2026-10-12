package main

import (
	"fmt"
	"io"
	"os"

	"github.com/AlecAivazis/survey/v2"
	"github.com/paul/minecraftctl/pkg/backup"
	"github.com/paul/minecraftctl/pkg/envfile"
	"github.com/spf13/cobra"
)

// Seams for tests
var (
	rotateGeteuid         = os.Geteuid
	rotateStdinIsTerminal = func() bool {
		info, err := os.Stdin.Stat()
		return err == nil && info.Mode()&os.ModeCharDevice != 0
	}
	rotateConfirm = func(message string) (bool, error) {
		ok := false
		err := survey.AskOne(&survey.Confirm{Message: message, Default: false}, &ok)
		return ok, err
	}
)

var backupRotatePasswordCmd = &cobra.Command{
	Use:   "rotate-password",
	Short: "Replace the restic repository password",
	Long: `Replace the restic repository password, on the server.

Steps:
  1. Generate a new password (32 random bytes, unpadded base64url)
  2. restic key add, reading the new password from a 0600 temp file
  3. Verify with the NEW password: restic key list (the new key must be the
     current one, the old key still listed) and restic check
  4. Copy the env file to <env-file>.bak (0600) and rewrite only its
     RESTIC_PASSWORD line (comments, order, quoting and mode are kept)
  5. restic key remove the old key, authenticated with the new password
  6. Print the new password once

If verification fails, both keys stay, the env file isn't touched, and the
error says how to remove the new key. The env file is updated before the old
key is removed, so it always holds a password that works.

Copy the printed password into infra/<env>/terraform.tfvars (restic_password)
and the password manager by hand: minecraftctl doesn't touch tfvars.

Runs as root (it rewrites the env file). Asks for confirmation unless --yes;
without a terminal, --yes is required. --dry-run prints the steps only.`,
	Args:              cobra.NoArgs,
	ValidArgsFunction: cobra.NoFileCompletions,
	RunE:              runBackupRotatePassword,
}

// rotateEnvPath is the env file rotate-password rewrites: --env-file, or
// /etc/minecraft.env
func rotateEnvPath(cmd *cobra.Command) string {
	if f := cmd.Flag("env-file"); f != nil && f.Value.String() != "" {
		return f.Value.String()
	}
	return envfile.DefaultMinecraftEnvPath
}

func printRotatePlan(w io.Writer, envPath, bakPath string) {
	fmt.Fprintf(w, `Dry run: nothing is changed. rotate-password would:
  1. generate a new password (32 random bytes, unpadded base64url)
  2. restic key list --json with the current password, noting the current key
  3. restic key add --new-password-file <0600 temp file, removed afterwards>
  4. restic key list --json with the current password, to find the new key's ID
  5. verify with the new password: restic key list (new key current, old key
     still listed) and restic check; stop here on failure, changing nothing else
  6. copy %[1]s to %[2]s (mode 0600)
  7. rewrite only the %[3]s line in %[1]s
  8. restic key remove <old key ID>, authenticated with the new password
  9. print the new password once, to copy into infra/<env>/terraform.tfvars
     and the password manager
`, envPath, bakPath, backup.PasswordEnvKey)
}

func runBackupRotatePassword(cmd *cobra.Command, args []string) error {
	out := cmd.OutOrStdout()
	envPath := rotateEnvPath(cmd)
	bakPath := envPath + ".bak"
	yes, _ := cmd.Flags().GetBool("yes")
	dryRun, _ := cmd.Flags().GetBool("dry-run")

	if dryRun {
		printRotatePlan(out, envPath, bakPath)
		return nil
	}

	if rotateGeteuid() != 0 {
		return fmt.Errorf("rotate-password rewrites %s, so it must run as root: sudo minecraftctl backup rotate-password", envPath)
	}
	if !backup.IsResticInstalled() {
		return fmt.Errorf("restic is not installed")
	}
	// restic prefers these over RESTIC_PASSWORD, so the "new password"
	// checks would really run with whatever they point at
	for _, name := range []string{"RESTIC_PASSWORD_FILE", "RESTIC_PASSWORD_COMMAND"} {
		if os.Getenv(name) != "" {
			return fmt.Errorf("%s is set; unset it so restic uses RESTIC_PASSWORD", name)
		}
	}

	cfg, err := backup.LoadConfig()
	if err != nil {
		return err
	}
	ef, err := envfile.Load(envPath)
	if err != nil {
		return err
	}
	filePassword, ok := ef.Get(backup.PasswordEnvKey)
	if !ok {
		return fmt.Errorf("%s has no %s line to update", envPath, backup.PasswordEnvKey)
	}
	if filePassword != cfg.Password {
		return fmt.Errorf("%s in the environment differs from the one in %s; unset it (or fix the file) so the old password is unambiguous", backup.PasswordEnvKey, envPath)
	}

	if !yes {
		if !rotateStdinIsTerminal() {
			return fmt.Errorf("refusing to rotate the password without --yes when stdin isn't a terminal")
		}
		ok, err := rotateConfirm(fmt.Sprintf("Rotate the restic password for %s and rewrite %s?", cfg.Repository, envPath))
		if err != nil {
			return err
		}
		if !ok {
			fmt.Fprintln(out, "Aborted; nothing changed.")
			return nil
		}
	}

	newPassword, err := backup.GeneratePassword()
	if err != nil {
		return err
	}
	res, err := cfg.RotatePassword(backup.RotateOptions{
		NewPassword: newPassword,
		EnvPath:     envPath,
		BackupPath:  bakPath,
		Out:         out,
	})
	if err == nil {
		fmt.Fprintf(out, "Done. Old key %s removed; only the new password opens the repository.\n", res.OldKeyID)
	}
	if res != nil && res.EnvUpdated {
		fmt.Fprintf(out, `
New restic password (shown once; %[1]s now holds it too):

  %[2]s

Copy it now into:
  - infra/<env>/terraform.tfvars (restic_password), before the next apply
    or instance replacement
  - the password manager
%[3]s holds the old env file (old password); delete it once you're done.
`, envPath, newPassword, bakPath)
	}
	return err
}

func init() {
	BackupCmd.AddCommand(backupRotatePasswordCmd)
	backupRotatePasswordCmd.Flags().BoolP("yes", "y", false, "Don't ask for confirmation")
	backupRotatePasswordCmd.Flags().Bool("dry-run", false, "Print the steps without running them")
}
