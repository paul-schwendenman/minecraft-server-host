#!/usr/bin/env bats
# Tests for autoshutdown.sh

load test_helper

setup() {
    setup_test_fixtures

    # Create mock commands
    create_mock "logger" 0 ""
    create_mock "sudo" 0 ""

    # Create wrapped script
    SCRIPT=$(wrap_script "${SCRIPTS_DIR}/minecraft/autoshutdown/autoshutdown.sh")
}

teardown() {
    teardown_test_fixtures
}

@test "autoshutdown: exits with error when minecraft.env missing" {
    rm -f "${MINECRAFT_ENV}"

    run bash "$SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"minecraft.env not readable"* ]]
}

@test "autoshutdown: exits with error when minecraftctl not found" {
    # Don't create minecraftctl mock

    run bash "$SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"minecraftctl not found"* ]]
}

@test "autoshutdown: skips shutdown when SSH sessions exist" {
    create_mock "minecraftctl" 0 "There are 0 of a max of 20 players online"
    # Mock 'who' to show SSH session
    create_mock "who" 0 "paul     pts/0        Dec 18 10:00"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    # Should not create touch file or call poweroff
    [ ! -f "${MINECRAFT_HOME}/no_one_playing" ]
}

@test "autoshutdown: creates touch file on first zero-player check" {
    create_mock "minecraftctl" 0 "There are 0 of a max of 20 players online"
    create_mock "who" 0 ""  # No SSH sessions

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    [ -f "${MINECRAFT_HOME}/no_one_playing" ]
    # poweroff should NOT be called on first check
    ! assert_mock_called_with "poweroff"
}

@test "autoshutdown: shuts down on second consecutive zero-player check" {
    create_mock "minecraftctl" 0 "There are 0 of a max of 20 players online"
    create_mock "who" 0 ""

    # Create touch file to simulate first check already happened
    touch "${MINECRAFT_HOME}/no_one_playing"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    # Touch file should be removed
    [ ! -f "${MINECRAFT_HOME}/no_one_playing" ]
    # poweroff should be called
    assert_mock_called_with "poweroff"
}

@test "autoshutdown: removes touch file when players are online" {
    create_mock "minecraftctl" 0 "There are 5 of a max of 20 players online"
    create_mock "who" 0 ""

    # Create touch file from previous check
    touch "${MINECRAFT_HOME}/no_one_playing"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    # Touch file should be removed
    [ ! -f "${MINECRAFT_HOME}/no_one_playing" ]
    # poweroff should NOT be called
    ! assert_mock_called_with "poweroff"
}

@test "autoshutdown: handles RCON failure gracefully" {
    create_mock "minecraftctl" 1 ""  # RCON fails
    create_mock "who" 0 ""

    run bash "$SCRIPT"

    # Should treat as 0 players (COUNT defaults to 0)
    [ "$status" -eq 0 ]
}

@test "autoshutdown: parses player count correctly from various formats" {
    # Test with different player counts
    create_mock "minecraftctl" 0 "There are 12 of a max of 20 players online"
    create_mock "who" 0 ""

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    # Should NOT create touch file since players > 0
    [ ! -f "${MINECRAFT_HOME}/no_one_playing" ]
}

@test "autoshutdown: removes touch file when SSH session detected" {
    create_mock "minecraftctl" 0 "There are 0 of a max of 20 players online"
    create_mock "who" 0 "paul     pts/0        Dec 18 10:00"

    # Create touch file
    touch "${MINECRAFT_HOME}/no_one_playing"

    run bash "$SCRIPT"

    [ "$status" -eq 0 ]
    # Touch file should be removed due to SSH session
    [ ! -f "${MINECRAFT_HOME}/no_one_playing" ]
}
