#!/usr/bin/env bats
# Tests for minecraft-active.sh

load test_helper

setup() {
    setup_test_fixtures

    create_mock "logger" 0 ""
    create_mock "systemctl" 0 ""

    make_world default

    SCRIPT=$(wrap_script "${SCRIPTS_DIR}/minecraft/active-world/minecraft-active.sh")
}

teardown() {
    teardown_test_fixtures
}

make_world() {
    mkdir -p "${MINECRAFT_HOME}/$1/world"
    touch "${MINECRAFT_HOME}/$1/world/level.dat"
}

# Mock IMDS: the token request succeeds; the tag request returns $1, or 404s
# (curl -f exits 22) when $1 is empty, as IMDS does for a missing tag.
mock_imds_tag() {
    local tag="$1"
    cat > "${MOCK_BIN}/curl" << EOF
#!/usr/bin/env bash
echo "curl \$*" >> "${TEST_TEMP_DIR}/mock_calls.log"
if [[ "\$*" == *"/api/token"* ]]; then echo "token"; exit 0; fi
if [[ -z "${tag}" ]]; then exit 22; fi
echo -n "${tag}"
EOF
    chmod +x "${MOCK_BIN}/curl"
}

@test "minecraft-active: starts the world named by the tag" {
    make_world old
    mock_imds_tag "old"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "systemctl start minecraft@old.service"
}

@test "minecraft-active: accepts dotted world names" {
    make_world "world.bak-1.19.2"
    mock_imds_tag "world.bak-1.19.2"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "systemctl start minecraft@world.bak-1.19.2.service"
}

@test "minecraft-active: falls back to default when the tag is missing" {
    mock_imds_tag ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"No ActiveWorld tag"* ]]
    assert_mock_called_with "systemctl start minecraft@default.service"
}

@test "minecraft-active: falls back to default when IMDS is unreachable" {
    create_mock "curl" 7 ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "systemctl start minecraft@default.service"
}

@test "minecraft-active: falls back to default when the tagged world doesn't exist" {
    mock_imds_tag "nope"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"'nope' is not a world"* ]]
    assert_mock_called_with "systemctl start minecraft@default.service"
}

@test "minecraft-active: rejects path traversal in the tag" {
    mkdir -p "${TEST_TEMP_DIR}/srv/escape/world"
    touch "${TEST_TEMP_DIR}/srv/escape/world/level.dat"
    mock_imds_tag "../escape"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "systemctl start minecraft@default.service"
    ! assert_mock_called_with "minecraft@../escape"
}

@test "minecraft-active: uses MC_DEFAULT_WORLD from minecraft.env" {
    make_world main
    echo "MC_DEFAULT_WORLD=main" >> "${MINECRAFT_ENV}"
    mock_imds_tag ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "systemctl start minecraft@main.service"
}

@test "minecraft-active: fails without starting anything when the default world is missing" {
    rm -rf "${MINECRAFT_HOME}/default"
    mock_imds_tag ""

    run bash "$SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"not starting anything"* ]]
    ! assert_mock_called_with "systemctl start"
}
