package main

import (
	"fmt"
	"runtime"
	"sync"

	"github.com/paul/minecraftctl/pkg/maps"
	"github.com/paul/minecraftctl/pkg/worlds"
	"github.com/rs/zerolog/log"
	"github.com/spf13/cobra"
)

var mapCmd = &cobra.Command{
	Use:   "map",
	Short: "Manage maps",
	Long:  "Commands for building and managing Minecraft world maps",
}

var mapBuildCmd = &cobra.Command{
	Use:   "build <world> [worlds...]",
	Short: "Build maps for one or more worlds",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		mapName, _ := cmd.Flags().GetString("map")
		force, _ := cmd.Flags().GetBool("force")
		lockFile, _ := cmd.Flags().GetString("lock-file")
		lockTimeout, _ := cmd.Flags().GetDuration("lock-timeout")
		noLock, _ := cmd.Flags().GetBool("no-lock")
		nonBlocking, _ := cmd.Flags().GetBool("non-blocking")
		parallel, _ := cmd.Flags().GetBool("parallel")
		maxWorkers, _ := cmd.Flags().GetInt("max-workers")

		// Expand all patterns to world names
		worldNames := make([]string, 0)
		for _, pattern := range args {
			expanded, err := worlds.ExpandWorldPattern(pattern)
			if err != nil {
				return fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
			}
			worldNames = append(worldNames, expanded...)
		}

		if len(worldNames) == 0 {
			return fmt.Errorf("no worlds found matching patterns")
		}

		if len(worldNames) == 1 && !parallel {
			// Single world, process directly
			builder := maps.NewBuilder()
			opts := maps.BuildOptions{
				WorldName:   worldNames[0],
				MapName:     mapName,
				Force:       force,
				LockFile:    lockFile,
				LockTimeout: lockTimeout,
				NoLock:      noLock,
				NonBlocking: nonBlocking,
			}
			return builder.Build(opts)
		}

		// Multiple worlds - batch processing
		return buildBatch(worldNames, maps.BuildOptions{
			MapName:     mapName,
			Force:       force,
			LockFile:    lockFile,
			LockTimeout: lockTimeout,
			NoLock:      noLock,
			NonBlocking: nonBlocking,
		}, parallel, maxWorkers)
	},
}

var mapPreviewCmd = &cobra.Command{
	Use:   "preview <world> <map>",
	Short: "Generate preview image for a map",
	Args:  cobra.ExactArgs(2),
	RunE: func(cmd *cobra.Command, args []string) error {
		worldName := args[0]
		mapName := args[1]

		builder := maps.NewBuilder()
		return builder.GeneratePreview(worldName, mapName)
	},
}

var mapManifestCmd = &cobra.Command{
	Use:   "manifest <world> [worlds...]",
	Short: "Build manifests for all maps in one or more worlds",
	Args:  cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		noPreview, _ := cmd.Flags().GetBool("no-preview")
		previewOnly, _ := cmd.Flags().GetBool("preview-only")
		parallel, _ := cmd.Flags().GetBool("parallel")
		maxWorkers, _ := cmd.Flags().GetInt("max-workers")

		// Expand all patterns to world names
		worldNames := make([]string, 0)
		for _, pattern := range args {
			expanded, err := worlds.ExpandWorldPattern(pattern)
			if err != nil {
				return fmt.Errorf("failed to expand pattern %s: %w", pattern, err)
			}
			worldNames = append(worldNames, expanded...)
		}

		if len(worldNames) == 0 {
			return fmt.Errorf("no worlds found matching patterns")
		}

		updateIndex, _ := cmd.Flags().GetBool("update-index")
		builder := maps.NewManifestBuilder()

		var err error
		if len(worldNames) == 1 && !parallel {
			// Single world, process directly
			opts := maps.ManifestOptions{
				WorldName:       worldNames[0],
				GeneratePreviews: !noPreview,
				PreviewOnly:      previewOnly,
			}
			err = builder.BuildManifests(worldNames[0], opts)
		} else {
			// Multiple worlds - batch processing
			err = manifestBatch(worldNames, maps.ManifestOptions{
				GeneratePreviews: !noPreview,
				PreviewOnly:      previewOnly,
			}, parallel, maxWorkers)
		}

		if err != nil {
			return err
		}

		// Update aggregate index if requested
		if updateIndex {
			return builder.BuildAggregateIndex()
		}

		return nil
	},
}

var mapIndexCmd = &cobra.Command{
	Use:   "index",
	Short: "Generate aggregate manifest and HTML index page",
	Long:  "Generates world_manifest.json and index.html in the maps directory",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		builder := maps.NewManifestBuilder()
		return builder.BuildAggregateIndex()
	},
}

func init() {
	mapBuildCmd.Flags().String("map", "", "Build only a specific map (by name)")
	mapBuildCmd.Flags().Bool("force", false, "Force rebuild even if map is up to date")
	mapBuildCmd.Flags().String("lock-file", "", "Path to lock file (default: from config)")
	mapBuildCmd.Flags().Duration("lock-timeout", 0, "Maximum time to wait for lock (0 = block forever)")
	mapBuildCmd.Flags().Bool("no-lock", false, "Disable file locking")
	mapBuildCmd.Flags().Bool("non-blocking", false, "Exit immediately if lock is held")
	mapBuildCmd.Flags().Bool("parallel", false, "Process multiple worlds in parallel")
	mapBuildCmd.Flags().Int("max-workers", runtime.NumCPU(), "Maximum number of parallel workers")

	mapManifestCmd.Flags().Bool("no-preview", false, "Skip preview generation")
	mapManifestCmd.Flags().Bool("preview-only", false, "Only generate previews, skip manifest")
	mapManifestCmd.Flags().Bool("parallel", false, "Process multiple worlds in parallel")
	mapManifestCmd.Flags().Int("max-workers", runtime.NumCPU(), "Maximum number of parallel workers")
	mapManifestCmd.Flags().Bool("update-index", false, "Update aggregate manifest and HTML index after manifest generation")

	mapCmd.AddCommand(mapBuildCmd)
	mapCmd.AddCommand(mapPreviewCmd)
	mapCmd.AddCommand(mapManifestCmd)
	mapCmd.AddCommand(mapIndexCmd)
}

// buildBatch processes multiple worlds in batch
func buildBatch(worldNames []string, baseOpts maps.BuildOptions, parallel bool, maxWorkers int) error {
	if !parallel {
		// Sequential processing
		var firstErr error
		for _, worldName := range worldNames {
			builder := maps.NewBuilder()
			opts := baseOpts
			opts.WorldName = worldName
			if err := builder.Build(opts); err != nil {
				log.Error().Err(err).Str("world", worldName).Msg("failed to build maps")
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr
	}

	// Parallel processing with worker pool
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers > len(worldNames) {
		maxWorkers = len(worldNames)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	errors := make([]error, len(worldNames))

	for i, worldName := range worldNames {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			builder := maps.NewBuilder()
			opts := baseOpts
			opts.WorldName = name
			errors[idx] = builder.Build(opts)
		}(i, worldName)
	}

	wg.Wait()

	// Return first error if any
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

// manifestBatch processes multiple worlds for manifest generation
func manifestBatch(worldNames []string, baseOpts maps.ManifestOptions, parallel bool, maxWorkers int) error {
	if !parallel {
		// Sequential processing
		var firstErr error
		for _, worldName := range worldNames {
			builder := maps.NewManifestBuilder()
			opts := baseOpts
			opts.WorldName = worldName
			if err := builder.BuildManifests(worldName, opts); err != nil {
				log.Error().Err(err).Str("world", worldName).Msg("failed to build manifests")
				if firstErr == nil {
					firstErr = err
				}
			}
		}
		return firstErr
	}

	// Parallel processing with worker pool
	if maxWorkers <= 0 {
		maxWorkers = runtime.NumCPU()
	}
	if maxWorkers > len(worldNames) {
		maxWorkers = len(worldNames)
	}

	var wg sync.WaitGroup
	semaphore := make(chan struct{}, maxWorkers)
	errors := make([]error, len(worldNames))

	for i, worldName := range worldNames {
		wg.Add(1)
		go func(idx int, name string) {
			defer wg.Done()
			semaphore <- struct{}{} // Acquire
			defer func() { <-semaphore }() // Release

			builder := maps.NewManifestBuilder()
			opts := baseOpts
			opts.WorldName = name
			errors[idx] = builder.BuildManifests(name, opts)
		}(i, worldName)
	}

	wg.Wait()

	// Return first error if any
	for _, err := range errors {
		if err != nil {
			return err
		}
	}
	return nil
}

