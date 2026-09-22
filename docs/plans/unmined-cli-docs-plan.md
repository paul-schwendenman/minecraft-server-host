# Plan: mirror unmined-cli in S3 and keep a versioned copy of its docs

Combines the unmined half of `s3-build-artifacts-plan.md` (S3 artifact mirror) with a checked-in copy of the unmined-cli help output. Both are done by the existing `unmined-update` workflow, from the same downloaded tarball.

## Context

unmined-cli is pinned by SHA256 in `packer/scripts/base/install_base_deps.sh:80` and downloaded from `unmined.net` at AMI build time. unmined.net only publishes a rolling `-dev` URL, so an old pin stops working as soon as a new build is published, and AMI builds fail until the hash PR is merged.

`.github/workflows/unmined-update.yml` runs weekly, downloads the tarball, and opens a PR (fixed branch `unmined-update`, via `peter-evans/create-pull-request@v7`) when the hash changes. The version number appears only in the PR title, not in any file.

minecraftctl relies on specific flags of `web render` and `image render` (`minecraftctl/pkg/maps/build.go`, `preview.go`). Nothing in the repo records what those flags are, so a bump can change them with nothing to review. The only copy is in `~/notes/unmined notes.md`.

Out of scope: Minecraft server JARs. The S3 plan recommended leaving those on Mojang's stable URLs, and that still holds.

## Answers to the questions

- **Does the archive hold docs?** Partly. `unmined-cli_<ver>_linux-x64/README.md` is in the tarball (usage, examples, `--area` syntax). The **per-verb option lists** (`image help render`, `web help render`, ...) are **not** in it. They only exist as `unmined-cli <module> help [<verb>]` output, so the binary has to be run. The rest of the archive is the binary, `.so` files, `config/`, `templates/`, `library/` and `LICENSE.txt`.
- **Can the binary run in CI?** It should. It is a Linux x64 build and `ubuntu-latest` is Linux x64. Not yet run; this machine is macOS, so it gets a local Docker check (see Verification).
- **Should CI commit the docs?** Yes, in the same PR. `create-pull-request` commits whatever changed in the working tree, so regenerating the docs before that step gives one PR with the pin bump plus the doc diff. A direct push to `master` would skip review, so no.
- **Where does the packer build get the tarball?** The builder EC2 instance has no IAM instance profile (`packer/base.pkr.hcl` sets none), so it cannot `aws s3 cp` from a private bucket. Decision: the GitHub runner (which has OIDC credentials) presigns a URL and hands it to the provisioner. See section 4. The alternative is adding an instance profile to the builder, which means more Terraform and a broader trust surface for an instance that runs arbitrary install scripts.

## Approach

### 1. Artifacts bucket (Terraform)

One bucket shared by test and prod, since these are build inputs and not environment data. Put it in `infra/global/` next to `github_actions_role`, not in `modules/s3_buckets` (which is instantiated per environment).

- `aws_s3_bucket.artifacts`, name `<prefix>-artifacts`, `force_destroy = false`, versioning on, SSE-S3, public access block. Same shape as the existing `backups` bucket.
- Add the bucket to the `s3_buckets` list passed to `module "github_actions_role"` in `infra/global/main.tf`. That already grants List/Get/Put/Delete via `S3DeployPolicy`, which covers both the update workflow (upload) and `packer-build.yml` (presign). No new policy.
- Layout:
  ```
  tools/unmined-cli/unmined-cli_<ver>_<sha12>_linux-x64.tar.gz
  ```
  The key includes the first 12 hex chars of the SHA256. unmined.net can republish the same version string with different bytes, and a version-only key would then overwrite the object an older commit's pin points at. With the hash in the key, every pin stays reproducible. Bucket versioning stays on as a second safety net.

### 2. One pin file instead of grep/sed on a shell script

Today the workflow finds the hash with `grep 'unmined-cli' ... | grep -oP '[a-f0-9]{64}'` and rewrites it with `sed`. That breaks silently if a second matching line appears. Replace it with a pin file, mirroring how `minecraft_jars.auto.pkrvars.hcl` works:

`packer/unmined.auto.pkrvars.hcl`
```hcl
unmined_cli = {
  version = "0.20.9-dev"
  sha256  = "0f3ac69c9e73c1edc86db3f8ebaaecb13bc4b535ccd7a845f48e5936d5c2920c"
}
```

The S3 key is derived from `version` and `sha256`, so nothing else has to be kept in sync. The workflow regenerates the whole file (no `sed`), and reads the current hash from it. This also gives `docs/unmined-cli/VERSION` a clear source.

### 3. Docs layout: one file per help screen, for small diffs

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

Output is raw help text with no timestamps and no wrapper header, so a bump that doesn't touch a given screen leaves that file's diff empty. Version and SHA live only in the pin file.

### 4. Scripts (new top-level `scripts/`)

There is no top-level `scripts/` directory yet. These must work both locally and in CI, so they don't belong under `packer/`.

**`scripts/unmined-docs.sh <extracted-unmined-cli-dir> [out-dir]`** (default out-dir `docs/unmined-cli`)
- Writes `VERSION` from the directory name and copies `README.md`.
- For each module (`image`, `web`) runs `unmined-cli <module> help`, parses the verb list from that output, then runs `help <verb>` for each. New verbs are picked up without script changes.
- Clears the out-dir's generated files first, so a removed verb's file is deleted in the PR.

**`scripts/unmined-artifact.sh`** (shared key logic, so workflow, packer build and humans agree)
- `key <version> <sha256>` prints the S3 key.
- `upload <tarball>` computes hash and version, then does `aws s3 cp` to the key if it doesn't already exist (`aws s3api head-object` first, skip if present).
- `presign <version> <sha256>` prints a presigned URL (`aws s3 presign --expires-in 3600`).

### 5. Edit `.github/workflows/unmined-update.yml`

- Add `id-token: write` to the job permissions and a `aws-actions/configure-aws-credentials@v6` step (same `AWS_ROLE_ARN` secret and region as `packer-build.yml`).
- The download step deletes the tarball (`rm -rf "${TMPDIR}"`). Extract into `${{ runner.temp }}` instead and keep it for later steps.
- After "Check if update needed", guarded by `needs_update == 'true'` (or `workflow_dispatch`):
  1. `scripts/unmined-artifact.sh upload` the tarball.
  2. Rewrite `packer/unmined.auto.pkrvars.hcl` (replaces the "Update hash in script" `sed` step).
  3. Run `scripts/unmined-docs.sh`.
- The upload must come before "Create or update Pull Request", so a merged PR can never point at an object that isn't there. If a PR is closed unmerged, the orphaned object is small and harmless.
- PR body: mention the S3 key and `docs/unmined-cli/`.
- On `workflow_dispatch`, always run upload and docs (the action no-ops if nothing changed). This seeds the bucket and the initial docs for the currently pinned 0.20.9-dev without waiting for a new release.

### 6. Packer changes

- `packer/base.pkr.hcl`: add variable `unmined_cli` (object with `version`, `sha256`) and `unmined_cli_url` (string, default `""`). Pass `UNMINED_URL` and `UNMINED_SHA256` to the `install_base_deps.sh` provisioner through `environment_vars`, the way `MINECRAFT_JARS_JSON` is passed to the JAR provisioner.
- `packer/scripts/base/install_base_deps.sh`: replace the hardcoded `wget` URL and hash with `${UNMINED_URL}` and `${UNMINED_SHA256}`. Keep the `sha256sum -c` check, since it is what makes the mirror trustworthy. Fail with a clear message if either variable is empty.
- `.github/workflows/packer-build.yml`: before building the base AMI, run `scripts/unmined-artifact.sh presign` and pass the result as `-var unmined_cli_url=...`. Add `unmined\.auto\.pkrvars\.hcl` to the `SHARED` change-detection regex so a pin bump rebuilds the AMIs (today only a change under `packer/scripts/base/` triggers a base build, and the pin file would live outside it).
- Local builds: `packer build -var "unmined_cli_url=$(scripts/unmined-artifact.sh presign ...)"`, documented in `packer/readme.rst`.
- `packer/tests/*.bats`: update any test that greps the hardcoded hash or URL.

### 7. Docs touch-ups (small)

- `docs/map-build-config.rst`: link to `docs/unmined-cli/` where `options` are described as mapped to unmined-cli arguments.
- `docs/github-actions.md`: add the unmined-update workflow to the table, noting that it mirrors the tarball to S3 and refreshes `docs/unmined-cli/`.
- `packer/readme.rst`: explain the pin file, the artifact bucket and the presign step, and point at `docs/unmined-cli/`.
- Mark the unmined parts of `s3-build-artifacts-plan.md` as superseded by this plan.

## Rollout order

One PR, not split. An earlier draft of this plan split "workflow writes the pin file" from "packer reads the pin file" into two merges, but in between nothing would keep `install_base_deps.sh`'s hardcoded hash current — the old `sed` step is gone from the workflow (replaced by the pin-file rewrite) before the new consumer exists, so a real upstream bump in that window would silently pin the AMI to a stale hash with no PR to fix it. Ship the scripts, pin file, Terraform, workflow, and packer changes together.

1. Terraform: create the bucket and extend the role (apply in `infra/global`), from the branch, before the rest is merged.
2. On the branch: scripts, pin file, workflow change, packer change (base AMI reads the presigned URL via the pin file), `packer-build.yml` change.
3. Run `workflow_dispatch` on `unmined-update` from the branch to seed the bucket and docs. Check the object exists and its SHA256 matches the pin.
4. Run a manual base AMI build from the branch to confirm the presigned URL flow works end to end.
5. Merge.

## Known limits

- `GITHUB_TOKEN`-created PRs do not trigger other workflows, so the bot PR gets no CI checks. That is already true today.
- Presigned URLs from OIDC session credentials expire with the session (1 hour by default). The URL is generated in the same job that runs packer, so this is fine, but a long queue between the presign and the provisioner would break it.
- The mirror only protects builds once the tarball is uploaded. A version that unmined.net publishes and the workflow hasn't seen yet is still unavailable, but that is now a "not yet updated" state and not a broken build.

## Verification

1. Run the docs generator locally against the extracted 0.20.9 tarball inside a Docker Ubuntu container (`docker run --platform linux/amd64 -v ...`). Confirm the out-dir has README, VERSION and a file per help screen, and that running it twice gives an identical tree.
2. Compare the container's `web help render` and `image help render` output with the copies in `~/notes/unmined notes.md`, to see how far apart the versions are.
3. `terraform plan` in `infra/global` shows only the new bucket resources and the extended role policy.
4. Read the workflow diff (`actionlint` if available). Run `workflow_dispatch` on the branch and confirm: the S3 object exists at the expected key, its SHA256 equals the pin, and the PR contains only `docs/unmined-cli/**`, `packer/unmined.auto.pkrvars.hcl` and nothing else unexpected.
5. Run `scripts/unmined-artifact.sh presign`, `curl` the URL, and check the hash. Then run a manual base AMI build and confirm `install_base_deps.sh` passes the `sha256sum -c` step.
6. Cross-check the flags minecraftctl passes (`--world --dimension --output --imageformat --chunkprocessors --log-level --zoomout --zoomin --force --area --topY --bottomY --gndxray --night --shadows`) against the generated files. Any flag missing from the help output is a real problem worth flagging.
7. `cd packer && bats tests/*.bats` and `packer fmt .` pass.

No commit is made until Paul asks for one.
