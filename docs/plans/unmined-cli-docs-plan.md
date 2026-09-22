# Plan: versioned copy of unmined-cli docs, refreshed by the update workflow

## Context

unmined-cli is pinned by SHA256 in `packer/scripts/base/install_base_deps.sh:80`. `.github/workflows/unmined-update.yml` runs weekly, downloads the tarball, and opens a PR (fixed branch `unmined-update`, via `peter-evans/create-pull-request@v7`) when the hash changes. The version number appears only in the PR title, not in any file.

minecraftctl relies on specific flags of `web render` and `image render` (`minecraftctl/pkg/maps/build.go`, `preview.go`). Nothing in the repo records what those flags are, so a bump can change them with nothing to review. The only copy is in `~/notes/unmined notes.md`.

Out of scope here: uploading the checksummed, versioned tarball to S3 for the AMI build to pull (see `docs/plans/s3-build-artifacts-plan.md`). That is a separate change to the same job. The job keeps downloading from unmined.net and checksumming as it does today, and the docs step just uses that same download.

## Answers to the questions

- **Does the archive hold docs?** Partly. `unmined-cli_<ver>_linux-x64/README.md` is in the tarball (usage, examples, `--area` syntax). The **per-verb option lists** (`image help render`, `web help render`, ...) are **not** in it. They only exist as `unmined-cli <module> help [<verb>]` output, so the binary has to be run. The rest of the archive is the binary, `.so` files, `config/`, `templates/`, `library/` and `LICENSE.txt`.
- **Can the binary run in CI?** It should. It is a Linux x64 build and `ubuntu-latest` is Linux x64. Not yet run; this machine is macOS, so it gets a local Docker check (see Verification).
- **Should CI commit it?** Yes, in the same PR. `create-pull-request` commits whatever changed in the working tree, so regenerating the docs before that step gives one PR with the SHA bump plus the doc diff. A direct push to `master` would skip review, so no.

## Approach

### 1. Layout: one file per help screen, for small diffs

```
docs/unmined-cli/
  VERSION              # e.g. 0.20.9-dev (only this changes on a pure bump)
  README.md            # verbatim from the archive
  image.txt            # `unmined-cli image help`
  image-render.txt     # `unmined-cli image help render`
  web.txt              # `unmined-cli web help`
  web-render.txt       # `unmined-cli web help render`
  ...                  # one file per verb the help output lists
```

Output is raw help text with no timestamps and no wrapper header, so a bump that doesn't touch a given screen leaves that file's diff empty. Version and SHA live only in `VERSION` (the SHA is already in `install_base_deps.sh`, so it isn't repeated).

### 2. New script: `scripts/unmined-docs.sh`

The repo has no top-level `scripts/` directory yet. The script must work both locally and in CI, so it should not live under `packer/`.

Usage: `scripts/unmined-docs.sh <extracted-unmined-cli-dir> [out-dir]` (default out-dir `docs/unmined-cli`).
- Writes `VERSION` from the directory name and copies `README.md`.
- For each module (`image`, `web`) runs `unmined-cli <module> help`, parses the verb list from that output, then runs `help <verb>` for each. New verbs are picked up without script changes.
- Clears the out-dir's generated files first, so a removed verb's file is deleted in the PR.

### 3. Edit `.github/workflows/unmined-update.yml`

- The download step deletes the tarball (`rm -rf "${TMPDIR}"`). Extract into `${{ runner.temp }}` instead and keep it for later steps.
- Add a step after "Update hash in script" and before "Create or update Pull Request", guarded by `needs_update == 'true'`, that runs `scripts/unmined-docs.sh`.
- Add `docs/unmined-cli/` to the PR body text.
- On `workflow_dispatch`, always regenerate (the action no-ops if nothing changed). This also produces the initial docs for the currently pinned 0.20.9-dev without waiting for a new release.

### 4. Docs touch-ups (small)

- `docs/map-build-config.rst`: link to `docs/unmined-cli/` where `options` are described as mapped to unmined-cli arguments.
- `docs/github-actions.md`: add the unmined-update workflow to the table, noting that it also refreshes `docs/unmined-cli/`.
- `packer/readme.rst`: one line pointing at `docs/unmined-cli/` where it mentions unmined.

## Known limits

- `GITHUB_TOKEN`-created PRs do not trigger other workflows, so the bot PR gets no CI checks. That is already true today.
- The workflow only greps lines containing `unmined-cli` in `install_base_deps.sh` for the hash. Nothing in the plan adds a second such line.

## Verification

1. Run the generator locally against the extracted 0.20.9 tarball inside a Docker Ubuntu container (`docker run --platform linux/amd64 -v ...`). Confirm the out-dir has README, VERSION and a file per help screen, and that running it twice gives an identical tree.
2. Compare the container's `web help render` and `image help render` output with the copies in `~/notes/unmined notes.md`, to see how far apart the versions are.
3. Read the workflow diff (and `actionlint` if available). Then run `workflow_dispatch` on the branch and confirm the PR contains `docs/unmined-cli/**` and nothing else beyond the expected changes.
4. Cross-check the flags minecraftctl passes (`--world --dimension --output --imageformat --chunkprocessors --log-level --zoomout --zoomin --force --area --topY --bottomY --gndxray --night --shadows`) against the generated files. Any flag missing from the help output is a real problem worth flagging.

No commit is made until Paul asks for one.
