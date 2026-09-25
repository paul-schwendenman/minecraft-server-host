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

# A world as `minecraftctl world create` leaves it: server.properties and a
# server.jar symlink into the jars dir, but no world/level.dat yet.
make_world() {
    local jars="${TEST_TEMP_DIR}/opt/minecraft/jars"
    mkdir -p "${MINECRAFT_HOME}/$1" "${jars}"
    touch "${MINECRAFT_HOME}/$1/server.properties" "${jars}/minecraft_server_1.21.jar"
    ln -sf "${jars}/minecraft_server_1.21.jar" "${MINECRAFT_HOME}/$1/server.jar"
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
    [[ "$output" == *"'nope' can't start"* ]]
    assert_mock_called_with "systemctl start minecraft@default.service"
}

@test "minecraft-active: rejects path traversal in the tag" {
    mkdir -p "${TEST_TEMP_DIR}/srv/escape"
    touch "${TEST_TEMP_DIR}/srv/escape/server.properties" "${TEST_TEMP_DIR}/srv/escape/server.jar"
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

@test "minecraft-active: starts a new world that has no level.dat yet" {
    make_world fresh
    mock_imds_tag "fresh"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [ ! -e "${MINECRAFT_HOME}/fresh/world/level.dat" ]
    assert_mock_called_with "systemctl start minecraft@fresh.service"
}

@test "minecraft-active: falls back to default when the world's jar is missing" {
    make_world old
    ln -sf "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.16.4.jar" \
        "${MINECRAFT_HOME}/old/server.jar"
    mock_imds_tag "old"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"points to a missing jar"* ]]
    assert_mock_called_with "systemctl start minecraft@default.service"
}

@test "minecraft-active: falls back to default when server.properties is missing" {
    make_world old
    rm "${MINECRAFT_HOME}/old/server.properties"
    mock_imds_tag "old"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"no "*"old/server.properties"* ]]
    assert_mock_called_with "systemctl start minecraft@default.service"
}
