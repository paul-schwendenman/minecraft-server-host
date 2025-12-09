# Migration Plan: Replace Packer Scripts with minecraftctl

## Overview

This document outlines the plan to migrate from bash scripts to using `minecraftctl` CLI tool for managing Minecraft servers in the packer image builds.

## Current State

### Scripts to Replace/Update

1. **Map Management Scripts**
   - `rebuild-map.sh` → Replace with `minecraftctl map build`
   - `build-map-manifests.sh` → Replace with `minecraftctl map manifest` and `minecraftctl map preview`
   - Map refresh services that use RCON → Use `minecraftctl rcon` commands

2. **Health Check Script**
   - `mc-healthcheck.sh` → Update to use `minecraftctl rcon status` instead of `mcrcon`

3. **Installation Scripts**
   - `install_deps.sh` → Add minecraftctl installation
   - `install_map_rebuild.sh` → Update to reference minecraftctl
   - `install_map_refresh.sh` → Update to use minecraftctl
   - `install_mc_healthcheck.sh` → Update health check script usage

### Systemd Services to Update

1. **minecraft-map-rebuild@.service**
   - Replace `/usr/local/bin/rebuild-map.sh` with `minecraftctl map build`
   - Replace `/usr/local/bin/build-map-manifests.sh` with `minecraftctl map manifest`

2. **minecraft-map-refresh@.service**
   - Replace RCON commands with drawer `minecraftctl rcon send`
   - Replace rebuild script with `minecraftctl map build`

3. **minecraft-override-rebuild.conf**
   - Update ExecStopPost commands to use minecraftctl

## Migration Steps

### Phase 1: Install minecraftctl in AMI

#### 1.1 Update `install_deps.sh`
- Add step to download/build minecraftctl binary
- Install to `/usr/local/bin/minecraftctl`
- Ensure proper permissions (755, root:root)
- Cross-compile for Linux amd64 in build process

**Implementation:**
```bash
# Add to install_deps.sh after automation tools section

# --- minecraftctl ---
if ! command -v minecraftctl >/dev/null 2>&1; then
  echo "[*] Installing minecraftctl"
  MINECRAFTCTL_VERSION="${MINECRAFTCTL_VERSION:-latest}"
  # Option 1: Download pre-built binary from releases (recommended)
  # Option 2: Build from source (if building AMI from source code)
  # For packer builds, copy from built binary
  sudo install -m 755 /tmp/minecraftctl /usr/local/bin/minecraftctl
fi
```

#### 1.2 Add minecraftctl to packer build
- Include minecraftctl binary in packer script directory
- Build minecraftctl during packer image creation:
  ```bash
  cd /tmp/scripts/minecraftctl
  make build-linux-amd64
  sudo install -m 755 minecraftctl-linux-amd64 /usr/local/bin/minecraftctl
  ```

#### 1.3 Create minecraftctl config file
- Install default config at `/etc/minecraftctl.yml`
- Configure default paths:
  ```yaml
  worlds_dir: /srv/minecraft-server
  maps_dir: /srv/minecraft-server/maps
  rcon:
    host: 127.0.0.1
    port: 25575
    password: ${RCON_PASSWORD}
  ```

### Phase 2: Replace Map Build Scripts

#### 2.1 Remove `rebuild-map.sh` wrapper - NOT NEEDED
- **Status**: ✅ **All features implemented in minecraftctl!**
- **Action**: Remove wrapper script entirely
- minecraftctl handles:
  - ✅ Lock file management (built-in)
  - ✅ Glob support (`world*` patterns)
  - ✅ Non-blocking mode (`--non-blocking` flag)
  - ✅ Path handling (uses world names directly)
  - ✅ All map building functionality
  
  ```bash
  #!/usr/bin/env bash
  # Wrapper for minecraftctl map build with locking support
  WORLD_PATH="${1:-}"
  MAP_FILTER=""
  FORCE=false
  NONBLOCK=false
  LOCK_FILE="/tmp/minecraft-map-build.lock"
  
  # Parse optional args
  shift || true
  while [[ $# -gt 0 ]]; do
    case "$1" in
      --map) MAP_FILTER="$2"; shift 2 ;;
      --force) FORCE=true; shift ;;
      --non-blocking) NONBLOCK=true; shift ;;
      *) shift ;;
    esac
  done
  
  # Handle globs
  if [[ "$WORLD_PATH" == *"*"* ]]; then
    for w in $WORLD_PATH; do
      [ -d "$w" ] && "$0" "$w" \
        ${MAP_FILTER:+--map "$MAP_FILTER"} \
        $([ "$FORCE" = "true" ] && echo "--force") \
        $([ "$NONBLOCK" = "true" ] && echo "--non-blocking")
    done
    exit 0
  fi
  
  # Acquire lock
  if [[ "${MAP_LOCK_HELD:-0}" != "1" ]]; then
    exec 200>"$LOCK_FILE"
    if $NONBLOCK; then
      if ! flock -n 200; then
        echo "⚠️  Another map rebuild is already running — skipping (non-blocking mode)." >&2
        exit 0
      fi
    else
      if ! flock -w 600 200; then
        echo "⚠️  Timeout waiting for map rebuild lock — skipping." >&2
        exit 0
      fi
    fi
    trap 'rm -f "$LOCK_FILE"' EXIT
    export MAP_LOCK_HELD=1
  fi
  
  # Convert world path to world name and call minecraftctl
  if [[ -z "$WORLD_PATH" || ! -d "$WORLD_PATH" ]]; then
    echo "❌ World path not found: $WORLD_PATH" >&2
    exit 1
  fi
  
  WORLD_NAME=$(basename "$WORLD_PATH")
  minecraftctl map build "$WORLD_NAME" \
    ${MAP_FILTER:+--map "$MAP_FILTER"} \
    $([ "$FORCE" = "true" ] && echo "--force")
  ```

#### 2.1a Create `build-map-manifests-wrapper.sh` (optional, for aggregate index)
**Status**: Preview generation is now automatic in `minecraftctl map manifest`!

Only needed if Enhancement 6 (aggregate index) not yet implemented:
```bash
#!/usr/bin/env bash
# Wrapper that generates previews and manifests using minecraftctl
set -euo pipefail

WORLD_PATH="${1:-}"
WORLD_NAME=$(basename "$WORLD_PATH")
CONFIG_PATH="$WORLD_PATH/map-config.yml"

if [[ ! -f "$CONFIG_PATH" ]]; then
  echo "⚠️  No config found for $WORLD_NAME, skipping" >&2
  exit 0
fi

# Build manifests (previews generated automatically)
minecraftctl map manifest "$WORLD_NAME"

# Update aggregate index (until Enhancement 6 is implemented)
# This generates world_manifest.json and index.html
/usr/local/bin/build-aggregate-index.sh || true
```

**Note**: This still requires `yq` for parsing the config. Alternative: enhance minecraftctl to generate previews during manifest step.

#### 2.2 Update `minecraft-map-rebuild@.service`
**Recommended**: Use minecraftctl directly (enhancements implemented):
```ini
[Unit]
Description=Rebuild Minecraft maps for %i
After=minecraft@%i.service

[Service]
Type=oneshot
User=minecraft
Group=minecraft
EnvironmentFile=-/etc/minecraft.env
# minecraftctl handles locking internally, use --non-blocking for timers
ExecStart=/usr/local/bin/minecraftctl map build %i --non-blocking
# Manifest command generates previews automatically, update index too
ExecStartPost=/usr/local/bin/minecraftctl map manifest %i --update-index
```


#### 2.3 Update `minecraft-map-refresh@.service`
```ini
[Unit]
Description=Save world and rebuild uNmINeD map for %i
After=network.target minecraft@%i.service

[Service]
Type=oneshot
User=minecraft
Group=minecraft
EnvironmentFile=-/etc/minecraft.env
ExecStartPre=/usr/local/bin/minecraftctl rcon send "say Saving world before map rebuild..."
ExecStartPre=/usr/local/bin/minecraftctl rcon send "save-all"
ExecStart=/usr/local/bin/minecraftctl map build %i
ExecStartPost=/usr/local/bin/minecraftctl map manifest %i --update-index
```

#### 2.4 Update `minecraft-override-rebuild.conf`
**Final Configuration** - Use minecraftctl directly:
```ini
[Service]
User=minecraft
Group=minecraft
UMask=002
EnvironmentFile=-/etc/minecraft.env
ProtectHome=no

# Use --non-blocking to avoid blocking server shutdown
ExecStopPost=/usr/local/bin/minecraftctl map build %i --non-blocking
ExecStopPost=/usr/local/bin/minecraftctl map manifest %i --update-index
```

### Phase 3: Update Installation Scripts

#### 3.1 Update `install_map_rebuild.sh`
- Remove installation of bash scripts (if removing wrappers)
- Update systemd service files only
- Ensure minecraftctl is installed before this runs

#### 3.2 Update `install_map_refresh.sh`
- Ensure minecraftctl is available
- Update service file references

#### 3.3 Update `install_mc_healthcheck.sh`
- No changes needed (script calls remain the same)
- But update `mc-healthcheck.sh` itself (Phase 4)

### Phase 4: Update Health Check Script

#### 4.1 Update `mc-healthcheck.sh`
**Final Update** - Replace RCON check with minecraftctl:
```bash
# OLD:
if echo "list" | mcrcon -H 127.0.0.1 -P "${RCON_PORT:-25575}" -p "${RCON_PASSWORD:-}" >/dev/null; then
    echo "  ✔ RCON responsive"
else
    echo "  ✘ RCON query failed"
fi

# NEW:
if /usr/local/bin/minecraftctl rcon status >/dev/null 2>&1; then
    echo "  ✔ RCON responsive"
else
    echo "  ✘ RCON query failed"
fi
```

**Note**: Remove `mcrcon` dependency check from health check (minecraftctl replaces it)

Update CLI tools check (replace mcrcon with minecraftctl):
```bash
# Update CLI tools section
for bin in minecraftctl aws java caddy; do
  if command -v "$bin" >/dev/null 2>&1; then
    echo "  ✔ $bin installed"
  else
    echo "  ✘ $bin missing"
  fi
done
[[ -x /opt/unmined/unmined-cli ]] && echo "  ✔ uNmINeD installed" || echo "  ✘ uNmINeD missing"
```

### Phase 5: Remove Obsolete Scripts

**Action**: Remove scripts that are fully replaced by minecraftctl:

1. **Remove** (fully replaced):
   - ✅ `rebuild-map.sh` - All functionality in `minecraftctl map build`
   - ✅ `build-map-manifests.sh` - All functionality in `minecraftctl map manifest` + `map index`

2. **Keep** (still needed, not replaced):
   - `backup-maps.sh` (S3 sync, not map building logic)
   - Other backup/world scripts (outside minecraftctl scope)

**Note**: After removing scripts, update `install_map_rebuild.sh` and `install_map_refresh.sh` to no longer install the bash scripts.

## Implementation Checklist

### Pre-Migration
- [ ] Build minecraftctl for Linux amd64
- [ ] Test minecraftctl commands match script behavior
- [ ] Verify minecraftctl handles all map-config.yml options correctly
- [ ] Test RCON integration works correctly

### Installation Phase
- [ ] Add minecraftctl to `install_deps.sh`
- [ ] Include minecraftctl binary in packer build
- [ ] Create `/etc/minecraftctl.yml` config file
- [ ] Test installation in fresh AMI build

### Service Updates
- [ ] Update `minecraft-map-rebuild@.service`
- [ ] Update `minecraft-map-refresh@.service`
- [ ] Update `minecraft-override-rebuild.conf`
- [ ] Test services start correctly
- [ ] Verify map building works via systemd timers

### Script Updates
- [ ] Update `install_map_rebuild.sh`
- [ ] Update `install_map_refresh.sh`
- [ ] Update `mc-healthcheck.sh`
- [ ] Test all installation scripts

### Testing
- [ ] Build new AMI with minecraftctl
- [ ] Deploy to test environment
- [ ] Verify map rebuild timers work
- [ ] Verify map refresh timers work
- [ ] Verify health check works
- [ ] Verify maps build correctly
- [ ] Verify manifests are generated
- [ ] Test RCON commands via minecraftctl

### Migration
- [ ] Deploy to production
- [ ] Monitor for issues
- [ ] Remove old scripts (if applicable) after successful migration

## Compatibility Notes

### World Path vs World Name
- **Scripts**: Use full paths like `/srv/minecraft-server/world`
- **minecraftctl**: Uses world names like `world` (derived from directory basename)
- **Solution**: Update all service files to use world name (the `%i` parameter in systemd templates)

### Environment Variables
- minecraftctl uses `/etc/minecraftctl.yml` or environment variables
- Ensure `RCON_PASSWORD` is available for minecraftctl
- Map configs remain in world directories (`map-config.yml`)

### Lock File Behavior
- Old script used `/tmp/minecraft-map-build.lock` with flock
- minecraftctl should handle this internally or we add wrapper script
- **Action**: Check if minecraftctl has locking, or add wrapper if needed

### Non-blocking Mode
- Old script had `--non-blocking` flag for lock acquisition
- **Action**: Check if minecraftctl supports this, or keep thin wrapper for this behavior

## Missing Functionality in minecraftctl

### Current Gaps (Updated Status)
See "Status Update" section below for current implementation status.

### Status Update - ALL FEATURES IMPLEMENTED! ✅

**All enhancements completed!** See `docs/minecraftctl-enhancement-plan.md` for details.

**Implemented and Verified**:
- ✅ Lock file management (Enhancement 1) - Built into `map build` command
- ✅ Preview generation in manifest (Enhancement 3) - Automatic in `map manifest`
- ✅ Non-blocking mode (Enhancement 4) - `--non-blocking` flag
- ✅ Glob support bounds - `ExpandWorldPattern()` implemented
- ✅ Aggregate manifest & HTML index (Enhancement 6) - `map index` command available
- ✅ Batch operations (Enhancement 5) - Parallel processing with `--parallel` flag

**Commands Available**:
- `minecraftctl map build <world> [worlds...]` - Build maps with locking, globs, parallel support
- `minecraftctl map manifest <world> [worlds...]` - Generate manifests with automatic previews
- `minecraftctl map index` - Generate aggregate manifest and HTML index
- `minecraftctl map manifest --update-index` - Generate manifests and update index in one command
- `minecraftctl rcon status` - Check server status (replaces mcrcon)

**Recommendation**:
- **Use minecraftctl directly** - No wrapper scripts needed!
- All features from original scripts are fully implemented
- Systemd services can call minecraftctl directly

## Rollback Plan

If issues arise:

1. Keep old scripts in place initially (rename to `.bak`)
2. Update services to use old scripts if needed
3. Systemd service files can be quickly reverted
4. Test thoroughly before removing old scripts

## Timeline Estimate

- **Phase 1** (Installation): 1-2 days
- **Phase 2** (Service Updates): 1 day
- **Phase 3** (Install Scripts): 1 day
- **Phase 4** (Health Check): 0.5 days
- **Phase 5** (Testing): 2-3 days
- **Total**: ~1 week

## Success Criteria

1. ✅ All map builds work via minecraftctl
2. ✅ Systemd timers trigger correctly
3. ✅ Health check passes with minecraftctl
4. ✅ RCON commands work via minecraftctl
5. ✅ No functionality regression
6. ✅ Performance is equal or better
7. ✅ Easier to maintain and extend

