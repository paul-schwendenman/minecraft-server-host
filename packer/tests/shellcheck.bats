#!/usr/bin/env bats
# Shellcheck validation for all packer scripts

PACKER_ROOT="$(cd "$(dirname "${BATS_TEST_DIRNAME}")" && pwd)"
SCRIPTS_DIR="${PACKER_ROOT}/scripts"

# SC1091: Don't follow source files (they only exist on target system)
SHELLCHECK_OPTS="-x -e SC1091"

# Skip all tests if shellcheck not installed
setup() {
    if ! command -v shellcheck &>/dev/null; then
        skip "shellcheck not installed (brew install shellcheck)"
    fi
}

@test "shellcheck: base/install_base_deps.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/base/install_base_deps.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: base/create-minecraft-user.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/base/create-minecraft-user.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: shared/install_minecraft_jars.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/shared/install_minecraft_jars.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/autoshutdown/autoshutdown.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/autoshutdown/autoshutdown.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/create-world/create-world.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/create-world/create-world.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/health/mc-healthcheck.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/health/mc-healthcheck.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/maps/rebuild-map.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/maps/rebuild-map.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/maps/backup-maps.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/maps/backup-maps.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/maps/build-map-manifests.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/maps/build-map-manifests.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/worlds/backup-worlds.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/worlds/backup-worlds.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/user-data/setup-env.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/user-data/setup-env.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/user-data/mount-ebs.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/user-data/mount-ebs.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/user-data/setup-maps.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/user-data/setup-maps.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

# Installation scripts
@test "shellcheck: minecraft/install_minecraft_service.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_minecraft_service.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_user_data_helpers.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_user_data_helpers.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_minecraftctl.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_minecraftctl.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_autoshutdown.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_autoshutdown.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_create_world.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_create_world.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_map_build.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_map_build.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_map_backup.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_map_backup.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_world_backup.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_world_backup.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}

@test "shellcheck: minecraft/install_mc_healthcheck.sh" {
    run shellcheck $SHELLCHECK_OPTS "${SCRIPTS_DIR}/minecraft/install_mc_healthcheck.sh"
    echo "$output"
    [ "$status" -eq 0 ]
}
