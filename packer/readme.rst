Minecraft Packer Build
======================

This directory defines the Packer configuration and provisioning scripts
for building the base Minecraft AMI.

The AMI contains all dependencies and helper scripts required to run and
manage one or more Minecraft worlds.

Architecture Overview
---------------------

::

   +------------------+        +---------------------+
   |   Terraform      |        |   Packer Build      |
   | (launch AMI,     |        | (base.pkr.hcl ->    |
   |  user_data)      |        |  minecraft.pkr.hcl) |
   +--------+---------+        +----------+----------+
            |                             |
            v                             v
   +----------------------------------------------+
   |                EC2 Instance                  |
   |----------------------------------------------|
   |  Boot (user_data, then systemd):             |
   |   - mount-ebs / setup-env / setup-maps       |
   |   - create-world.sh (create or register)     |
   |   - minecraft-active.service (start world)   |
   |   - dyndns.service (Route53 records)         |
   |                                              |
   |  Per world:                                  |
   |   - minecraft@<world>.service                |
   |   - minecraft-map-build@<world>.timer        |
   |   - minecraft-world-backup@<world>.timer     |
   |   - minecraft-map-backup@<world>.timer       |
   |                                              |
   |  Host:                                       |
   |   - autoshutdown.timer                       |
   |   - minecraftctl (world/map/backup/rcon CLI) |
   |                                              |
   |  Dependencies:                               |
   |   - OpenJDK 25, uNmINeD CLI, restic,         |
   |     AWS CLI, Caddy                           |
   +----------------------------------------------+
                      |
                      v
              +-----------------+
              |   Caddy Server  |
              | (serves maps at |
              |  /var/www/map)  |
              +-----------------+

World data lives on a separate EBS volume mounted at ``/srv/minecraft-server``
(one directory per world). It outlives AMI rebuilds; the root volume, and with
it every ``systemctl enable``, is replaced with each new AMI.

Which world runs
~~~~~~~~~~~~~~~~
Only one world runs at a time: every world uses port 25565 and RCON 25575.
Worlds are **not** enabled at boot one by one. ``minecraft-active.service``
reads the instance's ``ActiveWorld`` tag from instance metadata once
cloud-init has finished and starts that world. The control lambda sets the tag
(``POST /start?world=<name>``); see ``docs/plans/world-switching-plan.md``.

If the tag is missing, or the world can't start, it falls back to
``MC_DEFAULT_WORLD`` from ``/etc/minecraft.env`` (Terraform's ``world_name``,
usually ``default``). A world can start when its directory has:

- ``eula.txt`` with ``eula=true``
- ``server.properties`` with ``enable-rcon=true`` (autoshutdown and
  ``ExecStop`` talk to the server over RCON)
- ``server.jar`` pointing to a jar that exists in ``/opt/minecraft/jars/``

``world/level.dat`` isn't required: Minecraft writes it on a world's first
start. Check the log with ``journalctl -u minecraft-active``.

Included Tools & Scripts
------------------------

System Components
~~~~~~~~~~~~~~~~~
- **Java (OpenJDK 25)**: required for modern Minecraft server versions.
- **Minecraft server jars**: every version listed in
  ``minecraft_jars.auto.pkrvars.hcl``, in ``/opt/minecraft/jars/``. Each world's
  ``server.jar`` is a symlink to one of them.
- **minecraftctl**: Go CLI for worlds, maps, backups and RCON (see
  ``minecraftctl/README.md``). Scripts and units call it rather than using an
  RCON client directly.
- **uNmINeD CLI**: world map renderer (downloaded at build time from an S3
  mirror, since unmined.net only publishes a rolling "-dev" build with no
  stable URL to pin against; see ``unmined.auto.pkrvars.hcl`` and
  ``docs/unmined-cli/`` below).
- **restic**: world backups to the environment's S3 backup bucket.
- **AWS CLI**: map uploads and Route53 updates.
- **Caddy**: lightweight web server used to serve rendered maps.
- **mcstatus**, **nbtlib**: Python tools for server status and world data.

Systemd Units
~~~~~~~~~~~~~
- **minecraft@.service**: template unit for running a Minecraft world
  (one world = one unit). Its drop-ins back up the world and its maps when it
  stops.
- **minecraft-active.service**: starts the world named by the ``ActiveWorld``
  tag at boot (see above).
- **autoshutdown.timer** / **autoshutdown.service**: every 5 minutes; powers
  the instance off when no world is running, or after two checks with no
  players. An interactive SSH session skips shutdown.
- **minecraft-map-build@.timer** / **.service**: renders a world's maps every
  15 minutes while that world runs. The timer requires the world's
  ``minecraft@`` unit, so it's enabled per world but never started on its own.
- **minecraft-map-backup@.timer** / **.service**: uploads a world's rendered
  maps to S3 twice a day.
- **minecraft-world-backup@.timer** / **.service**: daily restic snapshot of
  a world's ``world/`` directory.
- **minecraft-world-backup.timer**, **minecraft-world-prune.timer**,
  **minecraft-map-backup.timer**, **minecraft-map-build-daily@.timer**:
  weekly full backups, prune and a daily map-build fallback. Installed but not
  enabled.
- **dyndns.service**: updates the Route53 A, AAAA and SSHFP records at boot.
- **minecraft-health.service**: one-off health check
  (``mc-healthcheck.sh``); run it by hand.

``minecraftctl world register <world>`` enables a world's three timers;
``world create`` does the same for a new world. Neither enables or starts
``minecraft@<world>``.

Helper Scripts
~~~~~~~~~~~~~~
- **create-world.sh** ``<world> <version> [seed]``: run from ``user_data`` on
  every new instance. Registers the world if it's already on the volume,
  otherwise creates it with ``minecraftctl world create``. Either way it then
  registers every other world on the volume, so all of them keep their backup
  timers. It doesn't start anything.
- **minecraft-active.sh**: the boot unit's script (see "Which world runs").
- **rebuild-map.sh**, **build-map-manifests.sh**: wrappers around
  ``minecraftctl map build`` used by the map-build units.
- **backup-maps.sh**: uploads a world's maps to the S3 map bucket.
- **autoshutdown.sh**: the idle check behind ``autoshutdown.timer``.
- **mount-ebs.sh**, **setup-env.sh**, **setup-maps.sh**,
  **configure-caddy.sh**, **publish-dns.sh**: ``user_data`` and boot helpers.

Usage
-----
- The base AMI needs a presigned URL into the ``minecraft-server-host-artifacts`` S3
  bucket for the pinned unmined-cli build (version/hash come from
  ``unmined.auto.pkrvars.hcl``, kept current by the
  ``unmined-update`` GitHub Actions workflow — see
  ``docs/github-actions.md``). The build instance has no AWS credentials of
  its own, so this has to be generated on the machine invoking ``packer``::

    cd packer
    AWS_PROFILE=minecraft ../scripts/unmined-artifact.sh presign \
      "$(grep -oP '(?<=version = ")[^"]+' unmined.auto.pkrvars.hcl)" \
      "$(grep -oP '(?<=sha256\s{2}= ")[a-f0-9]{64}' unmined.auto.pkrvars.hcl)"

- Build the AMI with::

    cd packer
    AWS_PROFILE=minecraft packer build \
      -var-file=minecraft_jars.auto.pkrvars.hcl \
      -var-file=unmined.auto.pkrvars.hcl \
      -var "unmined_cli_url=<presigned URL from above>" \
      base.pkr.hcl
    AWS_PROFILE=minecraft packer build -var-file=minecraft_jars.auto.pkrvars.hcl minecraft.pkr.hcl

  ``-var-file=unmined.auto.pkrvars.hcl`` has to be passed explicitly:
  packer only auto-loads sibling ``*.auto.pkrvars.hcl`` files when invoked
  against a directory, not when invoked against one specific ``.pkr.hcl``
  file, which is how every ``packer build``/``packer validate`` call in
  this repo works.

- Launch with Terraform (see ``infra/``). ``user_data`` mounts the volume and
  runs ``create-world.sh``.
- Manage worlds on the instance with ``minecraftctl`` (``world list``,
  ``world create``, ``world start``, ...).
- Maps are served by Caddy from ``/var/www/map`` on the server's domain.
- unmined-cli's own help output (per module/verb, plus its README) is
  mirrored at ``docs/unmined-cli/``, refreshed whenever the pin changes.


Null Builder
-------------

Run provisioners via SSH:

    packer build \
      -var "test_host=testmc.minecraft.example.com" \
      -var "test_private_key=~/.ssh/minecraft-packer.pem" \
      ssh.pkr.hcl

docker Builder
---------------

Shares ``scripts/base/install_base_deps.sh`` with the base AMI, so it needs
the same presigned unmined-cli URL (see Usage above)::

    cd packer
    AWS_PROFILE=minecraft packer build \
      -var-file=unmined.auto.pkrvars.hcl \
      -var "unmined_cli_url=<presigned URL, same as above>" \
      docker.pkr.hcl
    docker run -it minecraft-local bash
