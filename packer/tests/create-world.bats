#!/usr/bin/env bats
# Tests for create-world.sh

load test_helper

setup() {
    setup_test_fixtures

    # Create JAR directory
    mkdir -p "${TEST_TEMP_DIR}/opt/minecraft/jars"

    # Create mock commands
    create_mock "minecraftctl" 0 "World created"
    create_mock "sudo" 0 ""

    # Create wrapped script
    SCRIPT=$(wrap_script "${SCRIPTS_DIR}/minecraft/create-world/create-world.sh")
}

teardown() {
    teardown_test_fixtures
}

@test "create-world: shows usage when no arguments" {
    run bash "$SCRIPT"

    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
    [[ "$output" == *"<world-name>"* ]]
    [[ "$output" == *"<jar-version>"* ]]
}

@test "create-world: shows usage when only one argument" {
    run bash "$SCRIPT" "myworld"

    [ "$status" -eq 1 ]
    [[ "$output" == *"Usage:"* ]]
}

@test "create-world: exits with error when JAR not found" {
    run bash "$SCRIPT" "myworld" "1.21.1"

    [ "$status" -eq 1 ]
    [[ "$output" == *"jar for version 1.21.1 not found"* ]]
}

@test "create-world: creates world with minecraftctl" {
    # Create the JAR file
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"

    run bash "$SCRIPT" "survival" "1.21.1"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world create survival --version 1.21.1"
}

@test "create-world: passes seed when provided" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"

    run bash "$SCRIPT" "survival" "1.21.1" "8675309"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world create survival --version 1.21.1 --seed 8675309"
}

@test "create-world: registers existing world instead of creating" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"

    # Create existing world directory
    mkdir -p "${MINECRAFT_HOME}/existing_world"

    run bash "$SCRIPT" "existing_world" "1.21.1"

    [ "$status" -eq 0 ]
    [[ "$output" == *"already exists on EBS"* ]]
    assert_mock_called_with "minecraftctl world register existing_world"
}

@test "create-world: sets ownership after creation" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.20.4.jar"

    run bash "$SCRIPT" "newworld" "1.20.4"

    [ "$status" -eq 0 ]
    # Check sudo chown was called
    assert_mock_called_with "chown -R minecraft:minecraft"
}

@test "create-world: handles version with multiple dots" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.20.4.jar"

    run bash "$SCRIPT" "myworld" "1.20.4"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world create myworld --version 1.20.4"
}

@test "create-world: handles world names with hyphens" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"

    run bash "$SCRIPT" "vanilla-121-hardcore" "1.21.1"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world create vanilla-121-hardcore --version 1.21.1"
}

@test "create-world: registers the other worlds on the volume" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"
    for w in default old world.bak-1.19.2; do
        mkdir -p "${MINECRAFT_HOME}/$w"
        touch "${MINECRAFT_HOME}/$w/server.properties"
    done
    mkdir -p "${MINECRAFT_HOME}/caddy"  # not a world: no server.properties

    run bash "$SCRIPT" "default" "1.21.1"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world register old"
    assert_mock_called_with "minecraftctl world register world.bak-1.19.2"
    ! assert_mock_called_with "minecraftctl world register caddy"
    [ "$(grep -c 'world register default' "${TEST_TEMP_DIR}/mock_calls.log")" -eq 1 ]
}

@test "create-world: registers existing worlds after creating a new one" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"
    mkdir -p "${MINECRAFT_HOME}/old"
    touch "${MINECRAFT_HOME}/old/server.properties"

    run bash "$SCRIPT" "survival" "1.21.1"

    [ "$status" -eq 0 ]
    assert_mock_called_with "minecraftctl world create survival --version 1.21.1"
    assert_mock_called_with "minecraftctl world register old"
}

@test "create-world: keeps going when another world can't be registered" {
    touch "${TEST_TEMP_DIR}/opt/minecraft/jars/minecraft_server_1.21.1.jar"
    mkdir -p "${MINECRAFT_HOME}/default" "${MINECRAFT_HOME}/broken"
    touch "${MINECRAFT_HOME}/default/server.properties" "${MINECRAFT_HOME}/broken/server.properties"
    cat > "${MOCK_BIN}/minecraftctl" << EOF
#!/usr/bin/env bash
echo "minecraftctl \$*" >> "${TEST_TEMP_DIR}/mock_calls.log"
[[ "\$*" == *broken* ]] && exit 1
exit 0
EOF
    chmod +x "${MOCK_BIN}/minecraftctl"

    run bash "$SCRIPT" "default" "1.21.1"

    [ "$status" -eq 0 ]
    [[ "$output" == *"Could not register 'broken'"* ]]
}
