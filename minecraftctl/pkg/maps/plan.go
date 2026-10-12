package maps

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/paul/minecraftctl/pkg/config"
)

// BuildAction is what a build does with one map
type BuildAction string

const (
	// ActionSkip leaves the map alone
	ActionSkip BuildAction = "skip"
	// ActionFull renders every range, then the base map
	ActionFull BuildAction = "full"
	// ActionRanges renders some ranges and skips the base map
	ActionRanges BuildAction = "ranges"
)

// MapPlan is what a build will do with one map, and why
type MapPlan struct {
	Map    string
	Action BuildAction
	Reason string
	// Ranges lists the ranges that will render (all of them for a full build)
	Ranges []string
	// PrevFormat is the format the map was last rendered in ("" if unknown)
	PrevFormat string
	// FormatChanged is set when the map is re-rendered in a new image format,
	// so old-format tiles are removed after a successful full render
	FormatChanged bool
	// Force passes --force to uNmINeD for the base map and ranges
	Force bool
	// ForceRanges names the ranges a non-forced full build still renders with
	// --force, because their settings are new or changed since the last render
	ForceRanges []string

	mapDef    config.MapDefinition
	defaults  config.MapDefaults
	worldDir  string
	mapOutput string
	rangeIdx  []int  // indexes into mapDef.Ranges, in render order
	forced    []bool // per mapDef.Ranges index: render this range with --force
	hashes    []string
	keys      []string
	manifest  *Manifest
}

// planInput is everything decideMap looks at, gathered from disk by planMap
type planInput struct {
	mapDef         config.MapDefinition
	defaults       config.MapDefaults
	renderedFormat string    // from unmined.map.properties.js, "" if unknown
	manifest       *Manifest // nil if missing or unreadable
	worldMtime     int64
	force          bool
	rangesOnly     bool
}

// decideMap picks full/ranges/skip for one map. It does no IO.
//
// The image format is checked first: a format change also changes every range
// hash, and the viewer reads all tiles in one format, so it always needs a
// full render, never a ranges-only one.
func decideMap(in planInput) (MapPlan, error) {
	p := MapPlan{
		Map:        in.mapDef.Name,
		PrevFormat: in.renderedFormat,
		mapDef:     in.mapDef,
		manifest:   in.manifest,
	}
	p.keys = rangeKeys(in.mapDef.Ranges)
	p.hashes = make([]string, len(in.mapDef.Ranges))
	for i, r := range in.mapDef.Ranges {
		p.hashes[i] = rangeHash(r, in.mapDef, in.defaults)
	}
	all := func() {
		p.rangeIdx = nil
		p.Ranges = nil
		for i, r := range in.mapDef.Ranges {
			p.rangeIdx = append(p.rangeIdx, i)
			p.Ranges = append(p.Ranges, r.Name)
		}
	}

	formatChanged := in.renderedFormat != "" && !strings.EqualFold(in.renderedFormat, in.defaults.ImageFormat)

	if in.rangesOnly {
		if formatChanged {
			return p, fmt.Errorf("imageformat changed (%s→%s): a full rebuild is needed; run without --ranges-only",
				in.renderedFormat, in.defaults.ImageFormat)
		}
		if in.renderedFormat == "" {
			return p, fmt.Errorf("map has not been rendered yet (no imageFormat in unmined.map.properties.js): a full rebuild is needed; run without --ranges-only")
		}
		if len(in.mapDef.Ranges) == 0 {
			p.Action = ActionSkip
			p.Reason = "--ranges-only: no ranges configured"
			return p, nil
		}
		all()
		p.Action = ActionRanges
		p.Reason = "--ranges-only"
		p.Force = true
		return p, nil
	}

	switch {
	case formatChanged:
		p.Action = ActionFull
		p.Reason = fmt.Sprintf("imageformat changed (%s→%s)", in.renderedFormat, in.defaults.ImageFormat)
		p.FormatChanged = true
		p.Force = true
	case in.renderedFormat == "":
		p.Action = ActionFull
		p.Reason = "no rendered imageformat found (not rendered yet)"
		p.Force = in.force
	case in.force:
		p.Action = ActionFull
		p.Reason = "--force"
		p.Force = true
	case in.manifest == nil:
		p.Action = ActionFull
		p.Reason = "no manifest"
	case in.manifest.LastRenderedEpoch < in.worldMtime:
		p.Action = ActionFull
		p.Reason = "world changed since last render"
	}
	if p.Action == ActionFull {
		all()
		if !p.Force {
			p.forceChangedRanges(in.manifest)
		}
		return p, nil
	}

	// World unchanged: render only new or changed ranges
	var changed, added int
	for i, r := range in.mapDef.Ranges {
		old, ok := in.manifest.RangeHashes[p.keys[i]]
		if ok && old == p.hashes[i] {
			continue
		}
		if ok {
			changed++
		} else {
			added++
		}
		p.rangeIdx = append(p.rangeIdx, i)
		p.Ranges = append(p.Ranges, r.Name)
	}
	if len(p.rangeIdx) == 0 {
		p.Action = ActionSkip
		p.Reason = "up to date"
		return p, nil
	}
	p.Action = ActionRanges
	p.Reason = fmt.Sprintf("world unchanged; %d new, %d changed range(s)", added, changed)
	// A changed range may only differ in options or zoom, which uNmINeD's
	// incremental check doesn't notice, so ranges always render with --force
	p.Force = true
	return p, nil
}

// forceChangedRanges marks the ranges whose stored hash is missing or
// different for --force. Without it a full build renders them incrementally,
// and uNmINeD's incremental check doesn't notice an option or zoom change, but
// the new hash is still recorded, so the range would keep stale tiles.
func (p *MapPlan) forceChangedRanges(manifest *Manifest) {
	p.forced = make([]bool, len(p.keys))
	for i, k := range p.keys {
		if manifest != nil {
			if old, ok := manifest.RangeHashes[k]; ok && old == p.hashes[i] {
				continue
			}
		}
		p.forced[i] = true
		p.ForceRanges = append(p.ForceRanges, p.mapDef.Ranges[i].Name)
	}
	if len(p.ForceRanges) > 0 {
		p.Reason += fmt.Sprintf("; %d new or changed range(s) render with --force", len(p.ForceRanges))
	}
}

// rangeForced reports whether range i renders with --force
func (p MapPlan) rangeForced(i int) bool {
	return p.Force || (i < len(p.forced) && p.forced[i])
}

// planMap reads a map's rendered format and manifest and decides what to do
func planMap(mapDef config.MapDefinition, defaults config.MapDefaults, worldDir, worldMapsDir string, worldMtime int64, opts BuildOptions) (MapPlan, error) {
	outputSubdir := mapDef.OutputSubdir
	if outputSubdir == "" {
		outputSubdir = mapDef.Name
	}
	mapOutput := filepath.Join(worldMapsDir, outputSubdir)

	var manifest *Manifest
	if m, err := readManifest(filepath.Join(mapOutput, "manifest.json")); err == nil {
		manifest = m
	}
	p, err := decideMap(planInput{
		mapDef:         mapDef,
		defaults:       defaults,
		renderedFormat: renderedImageFormat(mapOutput),
		manifest:       manifest,
		worldMtime:     worldMtime,
		force:          opts.Force,
		rangesOnly:     opts.RangesOnly,
	})
	p.defaults = defaults
	p.worldDir = worldDir
	p.mapOutput = mapOutput
	return p, err
}

// Plan works out what a build of opts.WorldName would do with each map,
// without rendering anything or taking the build lock. If any map can't be
// planned (--ranges-only on a format change), it returns an error naming
// each one.
func (b *Builder) Plan(opts BuildOptions) ([]MapPlan, error) {
	worldPath := filepath.Join(b.worldsDir, opts.WorldName)
	mapConfig, err := config.LoadMapConfig(worldPath)
	if err != nil {
		return nil, fmt.Errorf("failed to load map config: %w", err)
	}
	worldDir := filepath.Join(worldPath, "world")
	worldStat, err := os.Stat(filepath.Join(worldDir, "level.dat"))
	if err != nil {
		return nil, fmt.Errorf("failed to stat level.dat: %w", err)
	}
	worldMapsDir := filepath.Join(b.mapsDir, opts.WorldName)

	var plans []MapPlan
	var errs []string
	for _, mapDef := range mapConfig.Maps {
		if opts.MapName != "" && mapDef.Name != opts.MapName {
			continue
		}
		p, err := planMap(mapDef, mapConfig.Defaults, worldDir, worldMapsDir, worldStat.ModTime().Unix(), opts)
		if err != nil {
			errs = append(errs, fmt.Sprintf("map %s: %v", mapDef.Name, err))
			continue
		}
		plans = append(plans, p)
	}
	if len(errs) > 0 {
		return plans, fmt.Errorf("%s", strings.Join(errs, "; "))
	}
	return plans, nil
}

// rangeHashInput is every setting that changes how a range renders. Field
// names are part of the hash, so renaming one re-renders every range once.
type rangeHashInput struct {
	V           int    `json:"v"`
	Dimension   string `json:"dimension"`
	Center      [2]int `json:"center"`
	Radius      int    `json:"radius"`
	Zoomout     int    `json:"zoomout"`
	Zoomin      int    `json:"zoomin"`
	ImageFormat string `json:"imageformat"`
	TopY        *int   `json:"topY"`
	BottomY     *int   `json:"bottomY"`
	GndXray     bool   `json:"gndxray"`
	Night       bool   `json:"night"`
	Shadows     string `json:"shadows"`
}

// rangeHash hashes the settings a range renders with: its own center, radius
// and zoom (falling back to the map's, then the defaults), the map's
// dimension and options, and the image format.
func rangeHash(r config.MapRange, mapDef config.MapDefinition, defaults config.MapDefaults) string {
	zoomout := defaults.Zoomout
	if mapDef.Zoomout != nil {
		zoomout = *mapDef.Zoomout
	}
	if r.Zoomout != nil {
		zoomout = *r.Zoomout
	}
	zoomin := defaults.Zoomin
	if mapDef.Zoomin != nil {
		zoomin = *mapDef.Zoomin
	}
	if r.Zoomin != nil {
		zoomin = *r.Zoomin
	}
	o := mapDef.Options
	in := rangeHashInput{
		V:           1,
		Dimension:   mapDef.Dimension,
		Center:      r.Center,
		Radius:      r.Radius,
		Zoomout:     zoomout,
		Zoomin:      zoomin,
		ImageFormat: strings.ToLower(defaults.ImageFormat),
		TopY:        o.TopY,
		BottomY:     o.BottomY,
		GndXray:     o.GndXray != nil && *o.GndXray,
		Night:       o.Night != nil && *o.Night,
		Shadows:     config.ShadowsToString(o.Shadows),
	}
	data, _ := json.Marshal(in) // a struct of plain fields can't fail
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])[:16]
}

// rangeKeys names each range in the manifest's range_hashes. Names are
// required but not unique: the first range with a name keys by the name, so
// reordering ranges doesn't re-render them, and each later duplicate gets the
// smallest "name#k" (k >= 2) that is neither another range's literal name nor
// an earlier key. Every key is distinct, and the same ranges in the same order
// always get the same keys.
func rangeKeys(ranges []config.MapRange) []string {
	literal := make(map[string]bool, len(ranges))
	for _, r := range ranges {
		literal[r.Name] = true
	}
	keys := make([]string, len(ranges))
	used := make(map[string]bool, len(ranges))
	for i, r := range ranges {
		key := r.Name
		if used[key] {
			for k := 2; ; k++ {
				key = fmt.Sprintf("%s#%d", r.Name, k)
				if !literal[key] && !used[key] {
					break
				}
			}
		}
		used[key] = true
		keys[i] = key
	}
	return keys
}
