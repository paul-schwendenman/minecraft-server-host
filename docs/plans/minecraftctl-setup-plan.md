# Plan: Self-Bootstrapping Server with `minecraftctl setup`

Status: **proposed**, not implemented. Came out of a discussion on 2026-09-25.

## Goal

Someone should be able to download the prebuilt `minecraftctl` binary onto a
fresh Linux machine, run one command, and end up with a working server:
Minecraft running under systemd, autoshutdown when idle, and optionally backups.

```bash
curl -fsSL https://raw.githubusercontent.com/paul-schwendenman/minecraft-server-host/master/minecraftctl/install.sh | bash
sudo minecraftctl setup
sudo minecraftctl world create survival --version 1.21.9
```

What's out of scope: the web UI, the control/details/worlds lambdas, several
servers, Terraform. This is the single-machine version of the stack.

## Problem

Today the server stack is only built by Packer. It's spread across
`packer/scripts/base/*.sh` and `packer/scripts/minecraft/install_*.sh`, which copy
unit files and scripts into place from `/tmp/scripts`. None of it can be run
outside a Packer build, and some of it assumes AWS:

| Piece | Where | Tied to AWS? |
|-------|-------|--------------|
| `minecraft` user, pinned uid 996 | `base/create-minecraft-user.sh` | No |
| Java, restic, yq | `base/install_base_deps.sh` | No |
| `minecraft@.service` | `minecraft/install_minecraft_service.sh` | No |
| `/etc/minecraft.env` (RCON password) | `user-data/setup-env.sh` | No (but runs from user-data) |
| Autoshutdown timer + script | `minecraft/autoshutdown/` | No, but `poweroff` only makes sense on a cloud VM |
| World backups (restic) | `minecraft/worlds/` | Repo is S3, but restic supports other backends |
| Data volume mount | `user-data/mount-ebs.sh` | Yes (EBS) |
| Dynamic DNS | `user-data/publish-dns.sh`, `dyndns/` | Yes (Route53) |
| Map builds, map upload, Caddy | `minecraft/maps/`, base deps | Upload is S3; uNmINeD comes from our S3 mirror |
| Health check | `minecraft/health/` | Uses IMDS |

## Proposal

### 1. Embed the server assets in the binary

Move the unit files, sudoers file, tmpfiles.d config and remaining shell scripts
(`autoshutdown.sh` etc.) into `minecraftctl/internal/setup/assets/` and embed them
with `go:embed`. `minecraftctl setup` installs them. The version of the assets
then always matches the binary that runs them.

### 2. Packer calls `minecraftctl setup` (one install path)

This is the most important part. Replace the `install_*.sh` provisioners in
`minecraft.pkr.hcl` with:

```hcl
provisioner "shell" { inline = ["sudo minecraftctl setup --profile aws --yes"] }
```

If Packer keeps its own scripts, the two install paths will drift apart the same
way the uid did. With one path, every AMI build tests the self-bootstrap code.

The move can be gradual: port one component at a time (autoshutdown first) and
delete its `install_*.sh` once Packer uses `setup` for it.

### 3. Components and profiles

`setup` is built from components that can be turned on and off:

| Component | Default (`standalone`) | `aws` profile |
|-----------|------------------------|---------------|
| `user` | on | on |
| `deps` (Java, restic) | on | on (+ AWS CLI, uNmINeD, Caddy) |
| `service` (`minecraft@.service`, `minecraft.env`) | on | on |
| `autoshutdown` | on, `idle-action=stop` | on, `idle-action=poweroff` |
| `backup` | off unless `--backup-repo` given | on (S3) |
| `maps` | off | on |
| `dyndns`, `health`, `ebs` | not available | on |

Flags: `--profile`, `--only <components>`, `--skip <components>`, `--yes`.

### 4. Idle action: `poweroff` or `stop`

`autoshutdown.sh` always powers the machine off. That's right on EC2, where the
control lambda starts it again, but wrong on a home PC or a VPS you pay for
monthly. Add a setting in `/etc/minecraft.env`:

```bash
IDLE_ACTION=poweroff   # or: stop
```

`stop` stops every `minecraft@*` service instead of powering off. The rest of the
script (SSH check, switch lock, RCON player count, two idle checks) stays the
same. Add bats cases for both actions.

### 5. Idempotent, with `--dry-run`

Running `setup` a second time should change nothing. Each component checks
before it writes (file contents match, user exists with the right uid, unit
enabled). `--dry-run` prints what would change. This fits with the
`minecraftctl doctor` item in [minecraftctl-plan.md](minecraftctl-plan.md)
(Phase 7): `doctor` can check that everything `setup` installed is still in place.

### 6. Supported platforms

To start with, only what the AMI is built on: Ubuntu 22.04 on amd64. `setup`
refuses to run anywhere else unless `--force` is passed. Debian and arm64 can come
later. arm64 needs a `minecraftctl-linux-arm64` release build and a
different Java package.

### 7. Later: wake on join

With `idle-action=stop` the server stays down until someone runs
`minecraftctl world start`. To match the "on-demand" feel of the AWS setup, a small
listener could take over port 25565 while the world is stopped. It answers the
server-list ping with "Server is starting…" and starts `minecraft@<world>` when a
player tries to join. [lazymc](https://github.com/timvisee/lazymc) already does
this. Either integrate it as an optional component, or write
`minecraftctl wake` using the ping/handshake code we already have for `status`.
Not part of v1.

## Open questions

- **Name:** `setup`, `bootstrap` or `install`? (`install` is confusing next to
  `install.sh`.)
- **Should `deps` install Java?** The simplest path is `apt-get install
  openjdk-25-jre-headless`. The other option is downloading a pinned Temurin
  build, which works on more distros.
- **uid 996 on other machines:** the pin exists because of the EBS volume. On a
  fresh machine 996 might already be taken. Let `--uid` override it, and fail
  loudly as `create-minecraft-user.sh` does now.
- **Where does `minecraft.env` live** when there's no separate data volume? Keep
  the `/srv/minecraft-server/minecraft.env` + symlink layout so both paths look the
  same.
- **`world create` assumes the AWS stack:** by default it writes `map-config.yml`
  and enables the world's map/backup timers (`--no-map-config`, `--no-systemd`
  skip them). On a standalone install it should only enable timers for components
  that `setup` actually installed.
- **Uninstall:** worth a `setup --remove`, or out of scope?

## Steps

1. [ ] Add `IDLE_ACTION` to `autoshutdown.sh` (+ bats tests). Useful on its own.
2. [ ] Create `internal/setup` with `go:embed` assets and the component interface
   (check / apply / describe for `--dry-run`).
3. [ ] Port `user`, `service` and `autoshutdown`. Switch those Packer provisioners
   to `minecraftctl setup --only user,service,autoshutdown`.
4. [ ] Port `deps` and `backup` (restic repo from a flag, password generated into
   `minecraft.env`).
5. [ ] Test end to end on a fresh Ubuntu VM (multipass or a throwaway EC2
   instance). Write a "Run it yourself" section in `minecraftctl/README.md`.
6. [ ] Port the AWS-only components (`maps`, `dyndns`, `health`, `ebs`), then
   delete the remaining `install_*.sh` scripts and their bats tests.
7. [ ] Later: wake on join.
