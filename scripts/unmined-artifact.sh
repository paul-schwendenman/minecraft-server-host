#!/usr/bin/env bash
# Shared helper for mirroring unmined-cli release tarballs in S3, so the
# workflow that uploads a new tarball, the packer build that downloads it,
# and a human doing either by hand all agree on the S3 key.
#
# Usage:
#   scripts/unmined-artifact.sh key <version> <sha256>
#     Print the S3 key for a given version/hash pair.
#
#   scripts/unmined-artifact.sh upload <tarball> [version]
#     Compute the tarball's sha256 (and, if not given, read its version from
#     the unmined-cli_<version>_linux-x64/ directory name inside the tar),
#     then upload it to that key unless an object is already there.
#     Prints "version=<version>" and "sha256=<sha256>" on stdout.
#
#   scripts/unmined-artifact.sh presign <version> <sha256>
#     Print a presigned GetObject URL (1 hour) for that key. Requires AWS
#     credentials with s3:GetObject on the artifacts bucket.
#
# Env:
#   ARTIFACTS_BUCKET   S3 bucket name (default: minecraft-artifacts, the
#                      bucket created in infra/global)
set -euo pipefail

ARTIFACTS_BUCKET="${ARTIFACTS_BUCKET:-minecraft-artifacts}"

usage() {
  cat >&2 <<EOF
Usage:
  $0 key <version> <sha256>
  $0 upload <tarball> [version]
  $0 presign <version> <sha256>
EOF
  exit 1
}

artifact_key() {
  local version="$1" sha256="$2"
  local sha12="${sha256:0:12}"
  echo "tools/unmined-cli/unmined-cli_${version}_${sha12}_linux-x64.tar.gz"
}

cmd_key() {
  [[ $# -eq 2 ]] || usage
  artifact_key "$1" "$2"
}

cmd_upload() {
  [[ $# -eq 1 || $# -eq 2 ]] || usage
  local tarball="$1"
  [[ -f "${tarball}" ]] || {
    echo "error: ${tarball} not found" >&2
    exit 1
  }

  local version="${2:-}"
  if [[ -z "${version}" ]]; then
    # unmined-cli_<version>_linux-x64/...
    version=$(tar -tzf "${tarball}" | head -1 | sed -E 's|unmined-cli_([^/]+)_linux-x64/.*|\1|')
    [[ -n "${version}" ]] || {
      echo "error: could not read version from ${tarball}; pass it explicitly" >&2
      exit 1
    }
  fi

  local sha256
  sha256=$(sha256sum "${tarball}" | cut -d' ' -f1)

  local key
  key=$(artifact_key "${version}" "${sha256}")

  if aws s3api head-object --bucket "${ARTIFACTS_BUCKET}" --key "${key}" >/dev/null 2>&1; then
    echo "already present: s3://${ARTIFACTS_BUCKET}/${key}" >&2
  else
    aws s3 cp "${tarball}" "s3://${ARTIFACTS_BUCKET}/${key}" >&2
  fi

  echo "version=${version}"
  echo "sha256=${sha256}"
}

cmd_presign() {
  [[ $# -eq 2 ]] || usage
  local key
  key=$(artifact_key "$1" "$2")
  aws s3 presign "s3://${ARTIFACTS_BUCKET}/${key}" --expires-in 3600
}

[[ $# -ge 1 ]] || usage
subcommand="$1"
shift

case "${subcommand}" in
key) cmd_key "$@" ;;
upload) cmd_upload "$@" ;;
presign) cmd_presign "$@" ;;
*) usage ;;
esac
