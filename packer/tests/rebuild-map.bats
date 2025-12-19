#!/usr/bin/env bats
# Tests for rebuild-map.sh

load test_helper

setup() {
    setup_test_fixtures

    # Script doesn't need wrapping - uses direct paths
    SCRIPT="${SCRIPTS_DIR}/minecraft/maps/rebuild-map.sh"

    # Create mock minecraftctl
    create_mock "minecraftctl" 0 "Map built successfully"
}

teardown() {
    teardown_test_fixtures
}

@test "rebuild-map: exits with error when world path empty" {
    run bash "$SCRIPT" ""

    [ "$status" -eq 1 ]
    [[ "$output" == *"World path not found"* ]]
}

@test "rebuild-map: exits with error when world path doesn't exist" {
    run bash "$SCRIPT" "/nonexistent/world"

    [ "$status" -eq 1 ]
    [[ "$output" == *"World path not found"* ]]
}

@test "rebuild-map: calls minecraftctl with world name" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/survival"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl map build survival"
}

@test "rebuild-map: passes --map filter to minecraftctl" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/survival" --map overworld

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl map build survival --map overworld"
}

@test "rebuild-map: passes --force flag to minecraftctl" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/survival" --force

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl map build survival --force"
}

@test "rebuild-map: passes --non-blocking flag to minecraftctl" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/survival" --non-blocking

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl map build survival --non-blocking"
}

@test "rebuild-map: combines multiple flags" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/survival" --map nether --force --non-blocking

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl map build survival --map nether --force --non-blocking"
}

@test "rebuild-map: rejects unknown arguments" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/survival" --unknown

    [ "$status" -eq 1 ]
    [[ "$output" == *"Unknown argument"* ]]
}

@test "rebuild-map: processes glob pattern with multiple worlds" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/survival"
    mkdir -p "${TEST_TEMP_DIR}/worlds/creative"
    mkdir -p "${TEST_TEMP_DIR}/worlds/hardcore"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/*"

    [ "$status" -eq 0 ]
    # Should call minecraftctl for each world
    assert_mock_called_with "minecraftctl map build survival"
    assert_mock_called_with "minecraftctl map build creative"
    assert_mock_called_with "minecraftctl map build hardcore"
}

@test "rebuild-map: glob mode passes flags to each world" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/world1"
    mkdir -p "${TEST_TEMP_DIR}/worlds/world2"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/*" --force

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl map build world1 --force"
    assert_mock_called_with "minecraftctl map build world2 --force"
}

@test "rebuild-map: glob mode continues on error" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/world1"
    mkdir -p "${TEST_TEMP_DIR}/worlds/world2"

    # Make minecraftctl fail
    create_mock "minecraftctl" 1 "Build failed"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/*"

    # Should still exit 0 because glob mode continues on error
    [ "$status" -eq 0 ]
}

@test "rebuild-map: glob skips non-directories" {
    mkdir -p "${TEST_TEMP_DIR}/worlds/valid_world"
    touch "${TEST_TEMP_DIR}/worlds/not_a_world.txt"

    run bash "$SCRIPT" "${TEST_TEMP_DIR}/worlds/*"

    [ "$status" -eq 0 ]
    # Should only call for the directory
    assert_mock_called_with "minecraftctl map build valid_world"
    # Should NOT call for the file
    ! assert_mock_called_with "not_a_world"
}
