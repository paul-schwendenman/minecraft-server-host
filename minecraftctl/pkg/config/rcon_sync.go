package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/paul/minecraftctl/pkg/envfile"
	"github.com/paul/minecraftctl/pkg/properties"
)

// Change is one server.properties key SyncRcon fixed (or would fix).
//
// Old and New hold the raw values, including rcon.password; callers that
// print changes must mask password values themselves.
type Change struct{ Key, Old, New string }

// LookupFunc matches os.LookupEnv and (*envfile.EnvFile).Get.
type LookupFunc func(key string) (string, bool)

// SyncRcon makes the RCON settings in the server.properties at propsPath
// match the values returned by env:
//
//   - enable-rcon is always set to "true"
//   - rcon.port is set to RCON_PORT, only if RCON_PORT is present and an integer
//   - rcon.password is set to RCON_PASSWORD, only if RCON_PASSWORD is present
//
// Changes are returned in that order. The file is written only when dryRun
// is false and at least one change is needed; comments, key order and
// unrelated keys are preserved.
func SyncRcon(propsPath string, env LookupFunc, dryRun bool) ([]Change, error) {
	props, err := properties.Load(propsPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load server.properties: %w", err)
	}

	var changes []Change

	if enableRcon, _ := props.Get("enable-rcon"); enableRcon != "true" {
		changes = append(changes, Change{Key: "enable-rcon", Old: enableRcon, New: "true"})
		props.Set("enable-rcon", "true")
	}

	if rawPort, ok := env("RCON_PORT"); ok {
		if envPort, err := strconv.Atoi(rawPort); err == nil {
			// A missing or non-integer rcon.port compares as 0, so it gets fixed.
			propsPort, _ := props.GetInt("rcon.port")
			if propsPort != envPort {
				oldPort, _ := props.Get("rcon.port")
				changes = append(changes, Change{Key: "rcon.port", Old: oldPort, New: strconv.Itoa(envPort)})
				props.SetInt("rcon.port", envPort)
			}
		}
	}

	if envPassword, ok := env("RCON_PASSWORD"); ok {
		if propsPassword, _ := props.Get("rcon.password"); propsPassword != envPassword {
			changes = append(changes, Change{Key: "rcon.password", Old: propsPassword, New: envPassword})
			props.Set("rcon.password", envPassword)
		}
	}

	if !dryRun && len(changes) > 0 {
		if err := props.Save(); err != nil {
			return changes, fmt.Errorf("failed to save server.properties: %w", err)
		}
	}

	return changes, nil
}

// RconEnv returns the lookup SyncRcon should read RCON_PORT/RCON_PASSWORD
// from, plus a human-readable label for where the values came from.
//
// The env file at envPath is preferred. If it can't be loaded (e.g.
// /etc/minecraft.env is root:root 0600 and we run as the minecraft user from
// ExecStartPre) but RCON_PASSWORD is set in the process environment (systemd's
// EnvironmentFile= reads the file as root and exports it to Exec* processes),
// the process environment is used instead. If neither is available, the load
// error is returned.
//
// Note the precedence differs from InitWithEnvFile, which calls
// ExportIfNotSet: there a non-empty process env var wins over the file. Here
// a readable file wins over the process environment entirely, since the file
// is the source of truth that server.properties is being synced to.
func RconEnv(envPath string) (LookupFunc, string, error) {
	ef, err := envfile.Load(envPath)
	if err == nil {
		return ef.Get, ef.Path(), nil
	}
	if _, ok := os.LookupEnv("RCON_PASSWORD"); ok {
		reason := err
		var pathErr *os.PathError
		if errors.As(err, &pathErr) {
			reason = pathErr.Err // e.g. "permission denied", without repeating the path
		}
		return os.LookupEnv, fmt.Sprintf("environment (%s unreadable: %v)", envPath, reason), nil
	}
	return nil, "", fmt.Errorf("failed to load minecraft.env: %w", err)
}
