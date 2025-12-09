#!/usr/bin/env bash
set -euxo pipefail

mkdir -p /srv/minecraft-server/maps
if [ ! -L /var/www/map ]; then
  rm -rf /var/www/map
  ln -s /srv/minecraft-server/maps /var/www/map
fi
