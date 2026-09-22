# TODO: `minecraftctl backup rotate-password`

Status: **not started** — filed as a future enhancement, not urgent.

## Motivation

Rotating the restic backup encryption password is currently a manual runbook
(see `docs/rotate-restic-password.md`) run by hand over SSH: generate a
password, `restic key add`, verify with `restic key list` + `restic check`
under the new password, `restic key remove` the old one, then hand-edit
`/etc/minecraft.env`. The riskiest part — removing the old key before
confirming the new one actually works — is only as safe as doing the steps
in the right order under pressure.

## Scope

A `minecraftctl backup rotate-password` command that automates the
**server-side** sequence atomically:

1. Generate a new password
2. `restic key add` with it
3. Verify: `restic key list` + `restic check` authenticated with the *new*
   password specifically
4. Only on success, `restic key remove` the old key
5. Update `RESTIC_PASSWORD` in `/etc/minecraft.env`
6. Print the new password once, with a reminder to copy it into
   `infra/<env>/terraform.tfvars` and a password manager

## Explicitly out of scope

- Updating `terraform.tfvars` — that file lives on the operator's laptop,
  not the server. `minecraftctl` runs on the instance and has no business
  reaching back to a dev machine's local Terraform config. This stays a
  manual step no matter what.

## Gaps against current code (this isn't a small wiring change)

- `pkg/envfile` only supports `Load` (read) today — no write-back method
  exists. Writing the rotated password into `/etc/minecraft.env` needs new
  code.
- `pkg/backup/backup.go` has no `restic key` wrapper at all (only
  create/restore/prune/stats/check/list). Key management is a new
  capability.
- Needs its own safety checks: refuse to remove the old key if `restic
  check` under the new password fails; requires root (env file write);
  probably wants a `--yes`/confirmation gate given it's destructive.

## Priority

Low — this is the first time the password has ever been rotated. Revisit
if rotation becomes routine (e.g. after a real incident, or if a
periodic-rotation policy gets adopted).
