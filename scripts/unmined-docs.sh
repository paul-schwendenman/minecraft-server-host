#!/usr/bin/env bash
# Regenerate docs/unmined-cli/ from an extracted unmined-cli release.
#
# Usage: scripts/unmined-docs.sh <extracted-unmined-cli-dir> [out-dir]
#
# <extracted-unmined-cli-dir> is the unmined-cli_<version>_linux-x64 directory
# produced by extracting the release tarball. It must contain a runnable
# unmined-cli binary (Linux x64) and README.md.
set -euo pipefail

usage() {
  echo "Usage: $0 <extracted-unmined-cli-dir> [out-dir]" >&2
  exit 1
}

[[ $# -ge 1 && $# -le 2 ]] || usage

SRC_DIR=$(cd "$1" && pwd)
OUT_DIR="${2:-docs/unmined-cli}"
BIN="${SRC_DIR}/unmined-cli"

[[ -x "${BIN}" ]] || {
  echo "error: ${BIN} not found or not executable" >&2
  exit 1
}
[[ -f "${SRC_DIR}/README.md" ]] || {
  echo "error: ${SRC_DIR}/README.md not found" >&2
  exit 1
}

# The extracted directory is named unmined-cli_<version>_linux-x64.
DIR_NAME=$(basename "${SRC_DIR}")
VERSION="${DIR_NAME#unmined-cli_}"
VERSION="${VERSION%_linux-x64}"
if [[ -z "${VERSION}" || "${VERSION}" == "${DIR_NAME}" ]]; then
  echo "error: could not parse version from directory name '${DIR_NAME}'" >&2
  exit 1
fi

# unmined-cli writes a 3-line startup banner (timestamped, so it would churn
# every diff) ahead of every help screen, on stdout, and exits non-zero for
# every "help" invocation (even a successful one). strip_output drops the
# banner and any leading blank line; $(...) capture drops trailing blank
# lines for us.
strip_output() {
  grep -vE '^\[[0-9]{2}:[0-9]{2}:[0-9]{2} INF\]' | sed -e '/./,$!d'
}

run_help() {
  (cd "${SRC_DIR}" && "${BIN}" "$@" 2>/dev/null) || true
}

rm -rf "${OUT_DIR}"
mkdir -p "${OUT_DIR}"

echo "${VERSION}" >"${OUT_DIR}/VERSION"
cp "${SRC_DIR}/README.md" "${OUT_DIR}/README.md"

top_help=$(run_help | strip_output)
modules=$(printf '%s\n' "${top_help}" | grep -E '^    [A-Za-z0-9_-]+' | awk '{print $1}')

if [[ -z "${modules}" ]]; then
  echo "error: could not parse any modules from top-level help output" >&2
  exit 1
fi

for module in ${modules}; do
  module_help=$(run_help "${module}" help | strip_output)
  printf '%s\n' "${module_help}" >"${OUT_DIR}/${module}.txt"

  verbs=$(printf '%s\n' "${module_help}" | grep -E '^  [A-Za-z0-9_-]+' | awk '{print $1}' | grep -v '^help$' || true)

  for verb in ${verbs}; do
    verb_help=$(run_help "${module}" help "${verb}" | strip_output)
    printf '%s\n' "${verb_help}" >"${OUT_DIR}/${module}-${verb}.txt"
  done
done

echo "Wrote ${OUT_DIR} for unmined-cli ${VERSION} (modules: ${modules})" >&2
