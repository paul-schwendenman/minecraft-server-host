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
- **uNmINeD CLI**: world map renderer (downloaded at build time).
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
- Build the AMI with::

    AWS_PROFILE=minecraft packer build minecraft.pkr.hcl

- Launch with Terraform (see parent project modules).
- Use ``create-world.sh`` to create/manage worlds.
- Maps are available via HTTP at ``http://<server>/map/``.
