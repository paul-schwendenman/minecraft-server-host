package maps

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/paul/minecraftctl/pkg/config"
	"github.com/paul/minecraftctl/pkg/lock"
	"github.com/rs/zerolog/log"
)

// tileFormats maps a tile file extension to the image format it holds.
// uNmINeD names tiles tiles/zoom.<z>/<x>/<z>/tile.<x>.<z>.<imageformat>.
var tileFormats = map[string]string{
	"jpeg": "jpeg",
	"jpg":  "jpeg",
	"png":  "png",
	"webp": "webp",
}

// normalizeFormat returns the canonical name of an image format, or "" if
// it isn't one uNmINeD renders tiles in.
func normalizeFormat(format string) string {
	return tileFormats[strings.ToLower(format)]
}

// removeStaleTiles removes image files under mapOutput/tiles whose format
// isn't keepFormat, and any directories under tiles/ that end up empty. Only
// files with a known tile image extension are touched; nothing outside
// tiles/ is. With dryRun, nothing is removed and the counts say what would be.
func removeStaleTiles(mapOutput, keepFormat string, dryRun bool) (files int, bytes int64, err error) {
	keep := normalizeFormat(keepFormat)
	if keep == "" {
		return 0, 0, fmt.Errorf("unknown image format %q", keepFormat)
	}

	tilesDir := filepath.Join(mapOutput, "tiles")
	if _, err := os.Lstat(tilesDir); os.IsNotExist(err) {
		return 0, 0, nil
	}

	var dirs []string
	err = filepath.WalkDir(tilesDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if path != tilesDir {
				dirs = append(dirs, path)
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		format := normalizeFormat(strings.TrimPrefix(filepath.Ext(path), "."))
		if format == "" || format == keep {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		if dryRun {
			return nil
		}
		return os.Remove(path)
	})
	if err != nil || dryRun {
		return files, bytes, err
	}

	// Deepest first, so a parent is empty by the time it's checked
	for i := len(dirs) - 1; i >= 0; i-- {
		if entries, err := os.ReadDir(dirs[i]); err == nil && len(entries) == 0 {
			_ = os.Remove(dirs[i])
		}
	}
	return files, bytes, nil
}

// CleanOptions control `map clean`
type CleanOptions struct {
	WorldName string
	MapName   string // If set, only clean this map and skip orphan directories
	DryRun    bool
	LockFile  string // Map build lock (empty = use config default)
}

// MapCleanResult is what was (or would be) removed from one map
type MapCleanResult struct {
	Map     string
	Dir     string
	Format  string // The map's rendered image format
	Skipped string // Why the map was skipped, if it was
	Files   int
	Bytes   int64
}

// OrphanDirResult is a directory in the world's maps dir that no map in
// map-config.yml renders to
type OrphanDirResult struct {
	Dir   string
	Files int
	Bytes int64
}

// CleanResult summarizes a `map clean` run
type CleanResult struct {
	Maps    []MapCleanResult
	Orphans []OrphanDirResult
	// OrphansSkipped says why orphan directories weren't looked for, if they
	// weren't although no map was given
	OrphansSkipped string
}

// TotalFiles is the number of files removed across maps and orphan dirs
func (r *CleanResult) TotalFiles() int {
	n := 0
	for _, m := range r.Maps {
		n += m.Files
	}
	for _, o := range r.Orphans {
		n += o.Files
	}
	return n
}

// TotalBytes is the size of everything removed across maps and orphan dirs
func (r *CleanResult) TotalBytes() int64 {
	var n int64
	for _, m := range r.Maps {
		n += m.Bytes
	}
	for _, o := range r.Orphans {
		n += o.Bytes
	}
	return n
}

// Clean removes tiles that aren't in each map's rendered image format and,
// unless a single map is given, map directories that map-config.yml no
// longer lists. Files directly in the world's maps dir (manifest.json,
// preview.png) are never touched.
//
// It holds the map build lock throughout (also for --dry-run): a range render
// in a running build rewrites the properties file, so reading it mid-build
// could pick the wrong format and delete tiles of a conversion in progress.
// If a build holds the lock, Clean fails rather than waiting for it.
func (b *Builder) Clean(opts CleanOptions) (*CleanResult, error) {
	lockPath := opts.LockFile
	if lockPath == "" {
		lockPath = config.Get().LockFile
	}
	fileLock := lock.NewFileLock(lockPath)
	if err := fileLock.LockWithOptions(lock.LockOptions{NonBlocking: true}); err != nil {
		return nil, fmt.Errorf("a map build is running (lock %s is held), try again when it finishes: %w", lockPath, err)
	}
	defer func() {
		if err := fileLock.Unlock(); err != nil {
			log.Error().Err(err).Msg("failed to release lock")
		}
	}()
	return b.clean(opts)
}

// clean is Clean without the lock
func (b *Builder) clean(opts CleanOptions) (*CleanResult, error) {
	mapConfig, err := config.LoadMapConfig(filepath.Join(b.worldsDir, opts.WorldName))
	if err != nil {
		return nil, fmt.Errorf("failed to load map config: %w", err)
	}
	worldMapsDir := filepath.Join(b.mapsDir, opts.WorldName)

	result := &CleanResult{}
	found := false
	for _, mapDef := range mapConfig.Maps {
		if opts.MapName != "" && mapDef.Name != opts.MapName {
			continue
		}
		found = true

		mapOutput := filepath.Join(worldMapsDir, mapOutputSubdir(mapDef))
		res := MapCleanResult{Map: mapDef.Name, Dir: mapOutput}
		res.Format = renderedImageFormat(mapOutput)
		switch {
		case res.Format == "":
			res.Skipped = "rendered image format unknown (no unmined.map.properties.js)"
		case normalizeFormat(res.Format) == "":
			res.Skipped = fmt.Sprintf("unrecognized rendered image format %q", res.Format)
		default:
			res.Files, res.Bytes, err = removeStaleTiles(mapOutput, res.Format, opts.DryRun)
			if err != nil {
				return result, fmt.Errorf("map %s: %w", mapDef.Name, err)
			}
		}
		result.Maps = append(result.Maps, res)
	}
	if opts.MapName != "" {
		if !found {
			return nil, fmt.Errorf("map '%s' not found in map-config.yml", opts.MapName)
		}
		return result, nil
	}

	if reason := unsafeOrphanCheck(worldMapsDir, mapConfig.Maps); reason != "" {
		result.OrphansSkipped = reason
		log.Warn().Str("world", opts.WorldName).Str("reason", reason).Msg("not looking for orphan map directories")
		return result, nil
	}
	orphans, err := orphanMapDirs(worldMapsDir, mapConfig.Maps)
	if err != nil {
		return result, err
	}
	for _, dir := range orphans {
		files, bytes, err := dirUsage(dir)
		if err != nil {
			return result, err
		}
		if !opts.DryRun {
			if err := os.RemoveAll(dir); err != nil {
				return result, fmt.Errorf("failed to remove %s: %w", dir, err)
			}
		}
		result.Orphans = append(result.Orphans, OrphanDirResult{Dir: dir, Files: files, Bytes: bytes})
	}
	return result, nil
}

// mapOutputSubdir is the directory under the world's maps dir a map renders to
func mapOutputSubdir(mapDef config.MapDefinition) string {
	if mapDef.OutputSubdir != "" {
		return mapDef.OutputSubdir
	}
	return mapDef.Name
}

// unsafeOrphanCheck says why orphan directories can't be told apart from map
// output, or "" if they can. A map rendering into the world's maps dir itself
// (output_subdir "." or "a/..") would make every directory there, its own
// tiles/ included, look like an orphan; one rendering outside it ("..")
// isn't covered by the check at all.
func unsafeOrphanCheck(worldMapsDir string, defs []config.MapDefinition) string {
	for _, d := range defs {
		rel, err := filepath.Rel(worldMapsDir, filepath.Join(worldMapsDir, mapOutputSubdir(d)))
		switch {
		case err != nil:
			return fmt.Sprintf("map %s: %v", d.Name, err)
		case rel == ".":
			return fmt.Sprintf("map %s renders into the world's maps dir itself (output_subdir %q)", d.Name, d.OutputSubdir)
		case rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)):
			return fmt.Sprintf("map %s renders outside the world's maps dir (output_subdir %q)", d.Name, d.OutputSubdir)
		}
	}
	return ""
}

// orphanMapDirs lists directories directly under worldMapsDir that no map
// renders into. A nested output_subdir keeps its top-level directory.
// Symlinks and plain files are left alone.
func orphanMapDirs(worldMapsDir string, defs []config.MapDefinition) ([]string, error) {
	entries, err := os.ReadDir(worldMapsDir)
	if os.IsNotExist(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	known := make(map[string]bool, len(defs))
	for _, d := range defs {
		rel, err := filepath.Rel(worldMapsDir, filepath.Join(worldMapsDir, mapOutputSubdir(d)))
		if err != nil {
			continue
		}
		top := strings.Split(filepath.ToSlash(rel), "/")[0]
		known[top] = true
	}

	var orphans []string
	for _, e := range entries {
		if !e.IsDir() || known[e.Name()] {
			continue
		}
		orphans = append(orphans, filepath.Join(worldMapsDir, e.Name()))
	}
	sort.Strings(orphans)
	return orphans, nil
}

// dirUsage counts the regular files under dir and their total size
func dirUsage(dir string) (files int, bytes int64, err error) {
	err = filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.Type().IsRegular() {
			return nil
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		files++
		bytes += info.Size()
		return nil
	})
	return files, bytes, err
}
