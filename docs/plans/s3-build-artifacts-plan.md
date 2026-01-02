# S3 Build Artifacts Proxy Plan

## Problem

Packer AMI builds depend on external URLs that can break unexpectedly:

1. **unmined-cli** - Downloaded from `unmined.net` with hardcoded SHA256. Only dev releases exist, and each new release breaks the build until the hash is manually updated.

2. **Minecraft server JARs** - Downloaded from Mojang's CDN (`launcher.mojang.com`, `piston-data.mojang.com`). While more stable, these are external dependencies that could change or become unavailable.

## Solution

Create an S3 bucket to host build artifacts, making builds:
- **Reproducible** - Same inputs produce same outputs
- **Reliable** - No external dependencies during builds
- **Controlled** - Explicit upgrade process for new versions

## Implementation

### Phase 1: Infrastructure

Create a new S3 bucket for build artifacts:

```hcl
# infra/modules/s3_buckets/main.tf

resource "aws_s3_bucket" "artifacts" {
  bucket        = "${local.prefix}-artifacts"
  force_destroy = false  # Preserve artifacts
}

resource "aws_s3_bucket_versioning" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  versioning_configuration { status = "Enabled" }
}

resource "aws_s3_bucket_server_side_encryption_configuration" "artifacts" {
  bucket = aws_s3_bucket.artifacts.id
  rule {
    apply_server_side_encryption_by_default { sse_algorithm = "AES256" }
  }
}
```

Bucket structure:
```
s3://minecraft-artifacts/
├── tools/
│   └── unmined-cli/
│       └── unmined-cli_0.19.54-dev_linux-x64.tar.gz
└── minecraft-jars/
    ├── minecraft_server_1.16.5.jar
    ├── minecraft_server_1.17.1.jar
    └── ...
```

### Phase 2: Upload Scripts

Create a script to upload/sync artifacts:

```bash
# scripts/upload-artifacts.sh

#!/usr/bin/env bash
set -euo pipefail

BUCKET="minecraft-artifacts"

# Upload unmined-cli
upload_unmined() {
  local file="$1"
  local version=$(basename "$file" | sed 's/unmined-cli_\(.*\)_linux-x64.tar.gz/\1/')
  aws s3 cp "$file" "s3://${BUCKET}/tools/unmined-cli/unmined-cli_${version}_linux-x64.tar.gz"
  echo "Uploaded unmined-cli ${version}"
}

# Upload minecraft jar
upload_jar() {
  local file="$1"
  local version=$(basename "$file" | sed 's/minecraft_server_\(.*\).jar/\1/')
  aws s3 cp "$file" "s3://${BUCKET}/minecraft-jars/minecraft_server_${version}.jar"
  echo "Uploaded minecraft_server_${version}.jar"
}
```

### Phase 3: Update Packer Scripts

#### unmined-cli (packer/scripts/base/install_base_deps.sh)

Before:
```bash
UNMINED_VERSION="dev"
wget -q -O "${TMPDIR}/unmined-cli.tgz" "https://unmined.net/download/unmined-cli-linux-x64-${UNMINED_VERSION}/"
echo "<hash>  ${TMPDIR}/unmined-cli.tgz" | sha256sum -c -
```

After:
```bash
UNMINED_VERSION="0.19.54-dev"
ARTIFACTS_BUCKET="${ARTIFACTS_BUCKET:-minecraft-artifacts}"
aws s3 cp "s3://${ARTIFACTS_BUCKET}/tools/unmined-cli/unmined-cli_${UNMINED_VERSION}_linux-x64.tar.gz" "${TMPDIR}/unmined-cli.tgz"
echo "<hash>  ${TMPDIR}/unmined-cli.tgz" | sha256sum -c -
```

#### Minecraft JARs (packer/minecraft_jars.auto.pkrvars.hcl)

Before:
```hcl
minecraft_jars = [
  {
    version = "1.21.4"
    url     = "https://piston-data.mojang.com/v1/objects/.../server.jar"
    sha256  = "..."
  },
]
```

After:
```hcl
minecraft_jars = [
  {
    version = "1.21.4"
    url     = "s3://minecraft-artifacts/minecraft-jars/minecraft_server_1.21.4.jar"
    sha256  = "..."
  },
]
```

Update `install_minecraft_jars.sh` to handle S3 URLs:
```bash
if [[ "$url" == s3://* ]]; then
  aws s3 cp "$url" "$JAR_FILE"
else
  curl -fsSL "$url" -o "$JAR_FILE"
fi
```

### Phase 4: IAM Permissions

Ensure the Packer build role has S3 read access:

```hcl
# Add to packer IAM policy
{
  "Effect": "Allow",
  "Action": ["s3:GetObject"],
  "Resource": "arn:aws:s3:::minecraft-artifacts/*"
}
```

## Upgrade Process

When a new version is released:

1. Download the new artifact locally
2. Verify the checksum
3. Upload to S3: `./scripts/upload-artifacts.sh <file>`
4. Update the version/hash in packer config
5. Commit and push

This is more explicit than the current approach but prevents unexpected build failures.

## Alternative A: Automated PR for Unmined Updates

A lighter-weight approach that keeps direct downloads but automates the hash update:

### GitHub Action: `.github/workflows/unmined-update.yml`

```yaml
name: Check for unmined-cli updates

on:
  schedule:
    - cron: '0 9 * * 1'  # Weekly on Monday at 9am UTC
  workflow_dispatch:      # Manual trigger

jobs:
  check-update:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4

      - name: Get current version
        id: current
        run: |
          HASH=$(grep -oP '(?<=echo ")[a-f0-9]{64}' packer/scripts/base/install_base_deps.sh)
          echo "hash=$HASH" >> $GITHUB_OUTPUT

      - name: Download latest unmined-cli
        id: latest
        run: |
          # Download the latest dev release
          wget -q -O unmined-cli.tgz "https://unmined.net/download/unmined-cli-linux-x64-dev/"

          # Calculate SHA256
          HASH=$(sha256sum unmined-cli.tgz | cut -d' ' -f1)
          echo "hash=$HASH" >> $GITHUB_OUTPUT

          # Extract version from tarball
          tar -tzf unmined-cli.tgz | head -1 | grep -oP 'unmined-cli_\K[^_]+' > version.txt
          VERSION=$(cat version.txt)
          echo "version=$VERSION" >> $GITHUB_OUTPUT

      - name: Check if update needed
        id: check
        run: |
          if [ "${{ steps.current.outputs.hash }}" != "${{ steps.latest.outputs.hash }}" ]; then
            echo "needs_update=true" >> $GITHUB_OUTPUT
          else
            echo "needs_update=false" >> $GITHUB_OUTPUT
          fi

      - name: Update hash in script
        if: steps.check.outputs.needs_update == 'true'
        run: |
          sed -i 's/${{ steps.current.outputs.hash }}/${{ steps.latest.outputs.hash }}/' \
            packer/scripts/base/install_base_deps.sh

      - name: Create Pull Request
        if: steps.check.outputs.needs_update == 'true'
        uses: peter-evans/create-pull-request@v5
        with:
          token: ${{ secrets.GITHUB_TOKEN }}
          commit-message: "chore(packer): update unmined-cli to ${{ steps.latest.outputs.version }}"
          title: "Update unmined-cli to ${{ steps.latest.outputs.version }}"
          body: |
            Automated update of unmined-cli SHA256 hash.

            **Version:** ${{ steps.latest.outputs.version }}
            **New SHA256:** `${{ steps.latest.outputs.hash }}`

            [View release notes](https://unmined.net/downloads/)

            ---
            *This PR was created automatically by the unmined-update workflow.*
          branch: unmined-update-${{ steps.latest.outputs.version }}
          labels: dependencies,automated
```

### Pros
- No infrastructure changes needed
- Keeps external download (one less thing to maintain)
- Human approval via PR review
- Links to release notes for visibility

### Cons
- Still depends on external URL during builds
- Builds fail between release and PR merge

## Alternative B: Keep External URLs as Fallback

For minecraft JARs specifically, Mojang's URLs are quite stable. An alternative is to keep the external URLs but add S3 as a fallback:

```bash
if ! aws s3 cp "s3://${BUCKET}/..." "$JAR_FILE" 2>/dev/null; then
  curl -fsSL "$url" -o "$JAR_FILE"
  # Optionally upload to S3 for next time
fi
```

## Recommendation

**For unmined-cli:** Use the GitHub Action (Alternative A). It's simpler, requires no infrastructure changes, and the weekly check should catch updates before they break builds.

**For minecraft JARs:** Keep current approach (direct Mojang downloads). They're stable and versioned - new JARs only get added manually when you want to support a new Minecraft version.

## Open Questions

1. Should the artifacts bucket be shared across test/prod environments? (Only relevant if going with S3 approach)
2. What schedule makes sense for the unmined check? (Weekly? Daily?)
3. Should the workflow also verify the download works before creating the PR?
