# Test helper functions for bats tests
# Source this file in your tests: load test_helper

# Project root
PACKER_ROOT="$(cd "$(dirname "${BATS_TEST_DIRNAME}")" && pwd)"
SCRIPTS_DIR="${PACKER_ROOT}/scripts"

# Temp directory for test fixtures
setup_test_fixtures() {
    TEST_TEMP_DIR="$(mktemp -d)"
    export TEST_TEMP_DIR

    # Create mock minecraft home
    export MINECRAFT_HOME="${TEST_TEMP_DIR}/srv/minecraft-server"
    mkdir -p "${MINECRAFT_HOME}"

    # Create mock /etc/minecraft.env
    export MINECRAFT_ENV="${TEST_TEMP_DIR}/etc/minecraft.env"
    mkdir -p "$(dirname "${MINECRAFT_ENV}")"
    cat > "${MINECRAFT_ENV}" << 'EOF'
RCON_PASSWORD=testpass
RCON_HOST=localhost
RCON_PORT=25575
EOF

    # Create mock bin directory for command stubs
    export MOCK_BIN="${TEST_TEMP_DIR}/bin"
    mkdir -p "${MOCK_BIN}"
    export PATH="${MOCK_BIN}:${PATH}"
}

teardown_test_fixtures() {
    if [[ -n "${TEST_TEMP_DIR:-}" && -d "${TEST_TEMP_DIR}" ]]; then
        rm -rf "${TEST_TEMP_DIR}"
    fi
}

# Create a mock command that records calls and returns specified output
# Usage: create_mock <command> <exit_code> <stdout>
create_mock() {
    local cmd="$1"
    local exit_code="${2:-0}"
    local stdout="${3:-}"

    cat > "${MOCK_BIN}/${cmd}" << EOF
#!/usr/bin/env bash
echo "\$0 \$@" >> "${TEST_TEMP_DIR}/mock_calls.log"
echo "${stdout}"
exit ${exit_code}
EOF
    chmod +x "${MOCK_BIN}/${cmd}"
}

# Create mock that outputs from a file
create_mock_from_file() {
    local cmd="$1"
    local exit_code="${2:-0}"
    local output_file="$3"

    cat > "${MOCK_BIN}/${cmd}" << EOF
#!/usr/bin/env bash
echo "\$0 \$@" >> "${TEST_TEMP_DIR}/mock_calls.log"
cat "${output_file}"
exit ${exit_code}
EOF
    chmod +x "${MOCK_BIN}/${cmd}"
}

# Check if a mock was called with specific arguments
assert_mock_called_with() {
    local pattern="$1"
    grep -q "${pattern}" "${TEST_TEMP_DIR}/mock_calls.log"
}

# Get all mock calls
get_mock_calls() {
    cat "${TEST_TEMP_DIR}/mock_calls.log" 2>/dev/null || true
}

# Create a script wrapper that uses test fixtures
# This rewrites paths in the script to use test directories
wrap_script() {
    local script_path="$1"
    local wrapped_script="${TEST_TEMP_DIR}/wrapped_$(basename "$script_path")"

    # Copy script and replace paths
    sed \
        -e "s|/srv/minecraft-server|${MINECRAFT_HOME}|g" \
        -e "s|/etc/minecraft.env|${MINECRAFT_ENV}|g" \
        -e "s|/opt/minecraft|${TEST_TEMP_DIR}/opt/minecraft|g" \
        "$script_path" > "$wrapped_script"

    chmod +x "$wrapped_script"
    echo "$wrapped_script"
}
