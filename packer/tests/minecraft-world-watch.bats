#!/usr/bin/env bats
# Tests for minecraft-world-watch.sh
#
# The switch itself is `minecraftctl world switch` (tested in
# minecraftctl/pkg/worlds); these cover when the watcher calls it and what it
# does with each exit code.

load test_helper

setup() {
    setup_test_fixtures

    create_mock "logger" 0 ""
    mock_switch_exit 0
    HANDLED="${TEST_TEMP_DIR}/run/minecraft-world-tag"

    SCRIPT=$(wrap_script "${SCRIPTS_DIR}/minecraft/active-world/minecraft-world-watch.sh")
}

teardown() {
    teardown_test_fixtures
}

# Mock minecraftctl: `world switch` exits with $1
mock_switch_exit() {
    cat > "${MOCK_BIN}/minecraftctl" << EOF
#!/usr/bin/env bash
echo "minecraftctl \$*" >> "${TEST_TEMP_DIR}/mock_calls.log"
exit $1
EOF
    chmod +x "${MOCK_BIN}/minecraftctl"
}

# Mock IMDS: the token request succeeds; the tag request returns $1, or 404s
# (curl -f exits 22) when $1 is empty, as IMDS does for a missing tag.
mock_imds_tag() {
    local tag="$1"
    cat > "${MOCK_BIN}/curl" << EOF
#!/usr/bin/env bash
if [[ "\$*" == *"/api/token"* ]]; then echo "token"; exit 0; fi
if [[ -z "${tag}" ]]; then exit 22; fi
echo -n "${tag}"
EOF
    chmod +x "${MOCK_BIN}/curl"
}

handled() {
    printf '%s' "$1" > "${HANDLED}"
}

@test "minecraft-world-watch: does nothing while the tag is unchanged" {
    handled "second"
    mock_imds_tag "second"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    ! assert_mock_called_with "minecraftctl"
}

@test "minecraft-world-watch: switches when the tag changes" {
    handled "default"
    mock_imds_tag "second"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world switch second"
    [ "$(cat "${HANDLED}")" = "second" ]
}

@test "minecraft-world-watch: never forces a switch" {
    handled "default"
    mock_imds_tag "second"

    run bash "$SCRIPT"

    ! assert_mock_called_with "--force"
}

@test "minecraft-world-watch: retries when players are online" {
    handled "default"
    mock_imds_tag "second"
    mock_switch_exit 3

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"will retry"* ]]
    [ "$(cat "${HANDLED}")" = "default" ]
}

@test "minecraft-world-watch: retries after an unexpected error" {
    handled "default"
    mock_imds_tag "second"
    mock_switch_exit 1

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [ "$(cat "${HANDLED}")" = "default" ]
}

@test "minecraft-world-watch: doesn't retry a world that can't start" {
    handled "default"
    mock_imds_tag "nope"
    mock_switch_exit 2

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [ "$(cat "${HANDLED}")" = "nope" ]
}

@test "minecraft-world-watch: doesn't retry a world that rolled back" {
    handled "default"
    mock_imds_tag "broken"
    mock_switch_exit 4

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [ "$(cat "${HANDLED}")" = "broken" ]
}

@test "minecraft-world-watch: logs an error when nothing is left running" {
    handled "default"
    mock_imds_tag "broken"
    mock_switch_exit 5

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "logger -p user.err"
    [ "$(cat "${HANDLED}")" = "broken" ]
}

@test "minecraft-world-watch: takes the current tag as handled when boot didn't record one" {
    mock_imds_tag "second"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    ! assert_mock_called_with "minecraftctl"
    [ "$(cat "${HANDLED}")" = "second" ]
}

@test "minecraft-world-watch: leaves the running world alone when the tag is removed" {
    handled "second"
    mock_imds_tag ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    ! assert_mock_called_with "minecraftctl"
    [ -z "$(cat "${HANDLED}")" ]
}

@test "minecraft-world-watch: catches up when boot couldn't read the tag" {
    # IMDS was down at boot, so boot recorded no tag and started the default
    handled ""
    mock_imds_tag "second"

    run bash "$SCRIPT"

    assert_mock_called_with "minecraftctl world switch second"
}

@test "minecraft-world-watch: does nothing when IMDS is unreachable" {
    handled "default"
    create_mock "curl" 7 ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    ! assert_mock_called_with "minecraftctl"
    [ "$(cat "${HANDLED}")" = "default" ]
}
