#!/usr/bin/env bats
# Tests for minecraft-stop-gate.sh (installed as
# /usr/local/libexec/minecraft/stop-gate), the ExecStopPost gate used by the
# minecraft@.service backup and map-build drop-ins.

load test_helper

setup() {
    setup_test_fixtures
    unset SERVICE_RESULT EXIT_CODE EXIT_STATUS

    SCRIPT="${SCRIPTS_DIR}/minecraft/minecraft-stop-gate.sh"

    mock_backup
    mock_systemctl
}

teardown() {
    teardown_test_fixtures
}

# The gated command: records that it ran and exits with $CMD_STATUS
mock_backup() {
    cat > "${MOCK_BIN}/backup" << EOF
#!/usr/bin/env bash
echo "backup \$*" >> "${TEST_TEMP_DIR}/mock_calls.log"
exit \${CMD_STATUS:-0}
EOF
    chmod +x "${MOCK_BIN}/backup"
}

# Mock systemctl: `show -P <prop> <unit>` prints $MOCK_START or $MOCK_EXIT
# (microseconds) for the two timestamps, or fails if MOCK_SYSTEMCTL_FAIL is set.
mock_systemctl() {
    cat > "${MOCK_BIN}/systemctl" << EOF
#!/usr/bin/env bash
echo "systemctl \$*" >> "${TEST_TEMP_DIR}/mock_calls.log"
if [[ -n "\${MOCK_SYSTEMCTL_FAIL:-}" ]]; then exit 1; fi
case "\$3" in
  ExecMainStartTimestampMonotonic) echo "\${MOCK_START:-}" ;;
  ExecMainExitTimestampMonotonic) echo "\${MOCK_EXIT:-}" ;;
  *) exit 1 ;;
esac
EOF
    chmod +x "${MOCK_BIN}/systemctl"
}

# A bare `! cmd` doesn't fail a bats test, so negate inside a function
refute_mock_called_with() {
    ! grep -q "$1" "${TEST_TEMP_DIR}/mock_calls.log" 2>/dev/null
}

# Uptime of $1 seconds, starting at an arbitrary monotonic time
set_uptime() {
    export MOCK_START=5000000
    export MOCK_EXIT=$((5000000 + $1 * 1000000))
}

@test "stop-gate: success runs the command" {
    export SERVICE_RESULT=success

    run bash "$SCRIPT" --unit minecraft@w.service -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: SERVICE_RESULT unset (not under systemd) runs the command" {
    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
    refute_mock_called_with "systemctl"
}

@test "stop-gate: failure without --min-uptime skips" {
    export SERVICE_RESULT=exit-code EXIT_CODE=exited EXIT_STATUS=1
    set_uptime 3600

    run bash "$SCRIPT" --unit minecraft@w.service -- backup create w

    [ "$status" -eq 0 ]
    [[ "$output" == *"skipping backup"* ]]
    [[ "$output" == *"SERVICE_RESULT=exit-code EXIT_CODE=exited EXIT_STATUS=1"* ]]
    refute_mock_called_with "backup create"
}

@test "stop-gate: failure after a short run skips" {
    export SERVICE_RESULT=exit-code EXIT_CODE=exited EXIT_STATUS=1
    set_uptime 30

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    [[ "$output" == *"skipping backup"* ]]
    [[ "$output" == *"uptime=30s"* ]]
    refute_mock_called_with "backup create"
    assert_mock_called_with "systemctl show -P ExecMainStartTimestampMonotonic minecraft@w.service"
    assert_mock_called_with "systemctl show -P ExecMainExitTimestampMonotonic minecraft@w.service"
}

@test "stop-gate: failure after a long run runs the command" {
    export SERVICE_RESULT=exit-code EXIT_CODE=exited EXIT_STATUS=1
    set_uptime 3600

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: uptime exactly at the minimum runs the command" {
    export SERVICE_RESULT=exit-code
    set_uptime 300

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: timeout after a long run runs the command" {
    export SERVICE_RESULT=timeout
    set_uptime 3600

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: zero timestamps fail open" {
    export SERVICE_RESULT=exit-code
    export MOCK_START=0 MOCK_EXIT=0

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    [[ "$output" == *"uptime unknown"* ]]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: empty timestamps fail open" {
    export SERVICE_RESULT=exit-code
    export MOCK_START="" MOCK_EXIT=""

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: exit timestamp not after start fails open" {
    export SERVICE_RESULT=exit-code
    export MOCK_START=9000000 MOCK_EXIT=5000000

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: systemctl failing fails open" {
    export SERVICE_RESULT=exit-code
    export MOCK_SYSTEMCTL_FAIL=1

    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime 300 -- backup create w

    [ "$status" -eq 0 ]
    [[ "$output" == *"uptime unknown"* ]]
    assert_mock_called_with "backup create w"
}

@test "stop-gate: the command's exit status passes through" {
    export SERVICE_RESULT=success
    export CMD_STATUS=3

    run bash "$SCRIPT" --unit minecraft@w.service -- backup create w

    [ "$status" -eq 3 ]
}

@test "stop-gate: missing -- is a usage error" {
    run bash "$SCRIPT" --unit minecraft@w.service backup create w

    [ "$status" -eq 2 ]
    [[ "$output" == *"Usage"* ]]
    refute_mock_called_with "backup"
}

@test "stop-gate: missing --unit is a usage error" {
    run bash "$SCRIPT" -- backup create w

    [ "$status" -eq 2 ]
}

@test "stop-gate: empty command is a usage error" {
    run bash "$SCRIPT" --unit minecraft@w.service --

    [ "$status" -eq 2 ]
}

@test "stop-gate: non-numeric --min-uptime is a usage error" {
    run bash "$SCRIPT" --unit minecraft@w.service --min-uptime soon -- backup create w

    [ "$status" -eq 2 ]
}

@test "stop-gate: installer and drop-ins agree on the libexec path" {
    local gate=/usr/local/libexec/minecraft/stop-gate

    grep -qF "minecraft-stop-gate.sh\" ${gate}" "${SCRIPTS_DIR}/minecraft/install_minecraft_service.sh"
    grep -qF "ExecStopPost=${gate} " "${SCRIPTS_DIR}/minecraft/worlds/minecraft-override-world-backup.conf"
    grep -qF "ExecStopPost=${gate} " "${SCRIPTS_DIR}/minecraft/maps/minecraft-override-rebuild.conf"
    ! grep -rn "/usr/local/bin/minecraft-stop-gate" "${SCRIPTS_DIR}"
}
