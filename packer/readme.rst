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
   | (launch AMI)     |        | (create AMI image)  |
   +--------+---------+        +----------+----------+
            |                             |
            v                             v
   +----------------------------------------------+
   |                EC2 Instance                  |
   |----------------------------------------------|
   |  Systemd Units:                              |
   |   - minecraft@.service (per world)           |
   |   - autoshutdown.timer/service               |
   |   - map-rebuild.timer/service                |
   |                                              |
   |  Tools & Scripts:                            |
   |   - create-world.sh                          |
   |   - rebuild-map.sh                           |
   |   - autoshutdown.sh                          |
   |   - mcrcon / mcstatus                        |
   |                                              |
   |  Dependencies:                               |
   |   - OpenJDK 21                               |
   |   - uNmINeD CLI                              |
   |   - Caddy (map web server)                   |
   +----------------------------------------------+
                      |
                      v
              +-----------------+
              |   Caddy Server  |
              | (serves maps at |
              |  /var/www/map)  |
              +-----------------+

Included Tools & Scripts
------------------------

System Components
~~~~~~~~~~~~~~~~~
- **Java (OpenJDK 21)**: required for modern Minecraft server versions.
- **Caddy**: lightweight web server used to serve rendered maps.
- **uNmINeD CLI**: world map renderer (downloaded at build time from an S3
  mirror, since unmined.net only publishes a rolling "-dev" build with no
  stable URL to pin against; see ``unmined.auto.pkrvars.hcl`` and
  ``docs/unmined-cli/`` below).
- **mcrcon**: RCON client, used by scripts and automation for sending
  commands to the server.
- **mcstatus**: Python utility to query Minecraft server status.

Systemd Units
~~~~~~~~~~~~~
- **minecraft@.service**: template unit for running a Minecraft world as a
  service (one world = one unit).
- **autoshutdown.service** / **autoshutdown.timer**: checks for idle
  servers and shuts down the instance if no players or SSH sessions are active.
- **map-rebuild.service** / **map-rebuild.timer**: periodically triggers
  map regeneration for all worlds.

Helper Scripts
~~~~~~~~~~~~~~
- **create-world.sh**:
  Creates and initializes a new world directory with:
  - symlinked server jar
  - EULA acceptance
  - `server.properties` with RCON configured
  - systemd unit enabled/started
  - Caddy automatically serving the world’s map directory

- **rebuild-map.sh**:
  Renders a map for one or all worlds using uNmINeD and updates the
  landing page (`/var/www/map/index.html`) with links to available worlds.
  The currently active world(s) are highlighted.

- **autoshutdown.sh**:
  Called by systemd; queries RCON (or falls back to logs) to detect active
  players. Shuts down the machine after two idle checks.

Backups
~~~~~~~
Optional scripts can be installed for:
- **map backups to S3**
- **world backups to S3**

These are disabled by default and must be configured with AWS credentials.

Usage
-----
- The base AMI needs a presigned URL into the ``minecraft-artifacts`` S3
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

- Launch with Terraform (see parent project modules).
- Use ``create-world.sh`` to create/manage worlds.
- Maps are available via HTTP at ``http://<server>/map/``.
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
