# minecraftctl Development Plan

## Phase 0 — Foundations
**Goal:** Decide stack + repo hygiene so you don’t churn later.

- **Tech choice:** Go + Cobra + Viper + Zerolog  
  - NBT: `github.com/Tnze/go-mc/nbt`  
  - RCON: `github.com/gorcon/rcon`  
- **Repo layout:**
  ```
  minecraftctl/
  ├── cmd/minecraftctl/        # cobra root + wiring
  ├── pkg/
  │   ├── config/              # viper, map-config loader, merging
  │   ├── maps/                # unmined wrapper, preview, manifest
  │   ├── worlds/              # list/info/create/delete
  │   ├── nbt/                 # read level.dat
  │   ├── rcon/                # rcon client
  │   └── util/                # exec/logging/paths
  ├── internal/version/        # ldflags injection
  ├── testdata/                # sample worlds + map-config.yml
  ├── Makefile                 # build/test/release
  └── .github/workflows/ci.yml # lint + tests + release
  ```
- **Build targets:** linux/amd64 static binary; version via `-ldflags "-X internal/version.Version=$(git describe ...)"`.
- **Definition of Done:** repo boots, `minecraftctl --help` prints.

---

## Phase 1 — Config & Plumbing
**Goal:** Parse global config, env, CLI flags, and per-world `map-config.yml`.

1. **Root command & global flags**
   - `--config`, `--worlds-dir`, `--maps-dir`, `--rcon-*`, `--verbose`
   - Env bindings: `MINECRAFT_WORLDS_DIR`, `MINECRAFT_MAPS_DIR`, `MINECRAFT_RCON_HOST|PORT|PASS`
2. **Viper integration**
   - Load order: CLI > ENV > config file (`/etc/minecraftctl.yml`, `~/.config/minecraftctl.yml`) > defaults.
   - `${ENV_VAR}` substitution in YAML.
3. **Per-world map config**
   - Loader for `<world>/map-config.yml`
   - Merge with global `maps.defaults`
4. **`config show/get`**
   - `minecraftctl config show` (effective config JSON/YAML)
   - `minecraftctl map config show <world>`

---

## Phase 2 — Read-only Introspection
**Goal:** Non-destructive commands to verify pathing + NBT parsing.

- `world list`
- `world info <world>`
- `rcon status`

---

## Phase 3 — Map Build
**Goal:** Feature parity with `rebuild-map.sh`.

- Build maps per-dimension with uNmINeD CLI.
- Flags: `--dimension`, `--zoomout`, `--zoomin`, `--threads`, `--clean`.
- Optional preview generation.

---

## Phase 4 — Manifest & Preview
**Goal:** Feature parity with `build-map-manifests.sh`.

- Generate manifest.json + preview.png.
- Validate against existing HTML viewers.

---

## Phase 5 — RCON Commands
- `rcon send "<cmd>"`
- `rcon exec <file>`
- `rcon status`

---

## Phase 6 — Systemd Integration
- Replace existing services with calls to CLI:
  ```
  ExecStart=/usr/local/bin/minecraftctl map build %i
  ```

---

## Phase 7 — UX & Guardrails
- `--dry-run` and `doctor` command.
- JSON logs via Zerolog.

---

## Phase 8 — Packaging & Releases
- Makefile, CI build matrix, GitHub releases.
- Static binary copied in AMI build.

---

## Phase 9 — Tests & Fixtures
- Unit tests for config merging, NBT parsing, manifest generation.
- Golden tests for output validation.

---

## Phase 10 — Nice-to-Have
- `world backup`
- `map clean`
- `serve` API
- Autocomplete generation

---

## Milestone Checklist

- [ ] Root CLI + config merge (Viper)
- [ ] `world list`, `world info`
- [ ] `rcon status`
- [ ] `map build`
- [ ] `map manifest build`
- [ ] `map preview`
- [ ] Systemd integration
- [ ] `doctor`, `--dry-run`
- [ ] CI/CD + release pipeline
- [ ] Remove legacy scripts after burn-in
