#!/usr/bin/env bats
# Tests for minecraft-active.sh
#
# Whether a world can start and comes up is `minecraftctl world switch`'s job
# (tested in minecraftctl/pkg/worlds); these cover reading the tag and falling
# back to the default world.

load test_helper

setup() {
    setup_test_fixtures

    create_mock "logger" 0 ""
    mock_switch_failing ""

    SCRIPT=$(wrap_script "${SCRIPTS_DIR}/minecraft/active-world/minecraft-active.sh")
}

teardown() {
    teardown_test_fixtures
}

# Mock minecraftctl: `world switch <name>` succeeds unless <name> is one of
# the space-separated worlds in $1.
mock_switch_failing() {
    local failing="$1"
    cat > "${MOCK_BIN}/minecraftctl" << EOF
#!/usr/bin/env bash
echo "minecraftctl \$*" >> "${TEST_TEMP_DIR}/mock_calls.log"
for w in ${failing}; do
  if [[ "\$3" == "\$w" ]]; then exit 2; fi
done
exit 0
EOF
    chmod +x "${MOCK_BIN}/minecraftctl"
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

@test "minecraft-active: switches to the world named by the tag" {
    mock_imds_tag "old"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world switch old"
    ! assert_mock_called_with "minecraftctl world switch default"
}

@test "minecraft-active: passes dotted world names through" {
    mock_imds_tag "world.bak-1.19.2"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world switch world.bak-1.19.2"
}

@test "minecraft-active: uses default when the tag is missing" {
    mock_imds_tag ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"No ActiveWorld tag"* ]]
    assert_mock_called_with "minecraftctl world switch default"
}

@test "minecraft-active: uses default when IMDS is unreachable" {
    create_mock "curl" 7 ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world switch default"
}

@test "minecraft-active: falls back to default when the tagged world doesn't start" {
    mock_imds_tag "nope"
    mock_switch_failing "nope"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [[ "$output" == *"'nope' didn't start, using default"* ]]
    assert_mock_called_with "minecraftctl world switch nope"
    assert_mock_called_with "minecraftctl world switch default"
}

@test "minecraft-active: tries the default once when the tag names it" {
    mock_imds_tag "default"
    mock_switch_failing "default"

    run bash "$SCRIPT"

    [ "$status" -eq 1 ]
    [ "$(grep -c "world switch default" "${TEST_TEMP_DIR}/mock_calls.log")" -eq 1 ]
}

@test "minecraft-active: uses MC_DEFAULT_WORLD from minecraft.env" {
    echo "MC_DEFAULT_WORLD=main" >> "${MINECRAFT_ENV}"
    mock_imds_tag ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world switch main"
}

@test "minecraft-active: fails when the default world doesn't start either" {
    mock_imds_tag "old"
    mock_switch_failing "old default"

    run bash "$SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"'default' didn't start, nothing is running"* ]]
}
