#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail
set -o verbose

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"

# Setup minecraft user
sudo adduser --system --home "${MINECRAFT_HOME}" "${MINECRAFT_USER}"
sudo addgroup --system "${MINECRAFT_GROUP}"
sudo adduser "${MINECRAFT_USER}" "${MINECRAFT_GROUP}"
sudo chown -R "${MINECRAFT_USER}.${MINECRAFT_GROUP}" "${MINECRAFT_HOME}"

# Install updates
#sudo apt update
#DEBIAN_FRONTEND=noninteractive sudo apt upgrade -y -qq

# Install java
sudo apt update
sudo apt install -qq -y openjdk-11-jdk-headless openjdk-11-jre-headless

# Download server
wget https://launcher.mojang.com/v1/objects/d0d0fe2b1dc6ab4c65554cb734270872b72dadd6/server.jar -O minecraft_server.1.14.3.jar
echo "942256f0bfec40f2331b1b0c55d7a683b86ee40e51fa500a2aa76cf1f1041b38  minecraft_server.1.14.3.jar" | shasum -a256 -c -
sudo cp minecraft_server.1.14.3.jar "${MINECRAFT_HOME}/minecraft_server.jar"

# Accept EULA
sudo -u "${MINECRAFT_USER}" tee "${MINECRAFT_HOME}/eula.txt" > /dev/null << EOF
eula=true
EOF

# Install systemd service
sudo tee /etc/systemd/system/minecraft.service > /dev/null << EOF
[Unit]
Description=minecraft-server
After=network.target

[Service]
WorkingDirectory=/srv/minecraft-server
User=minecraft
Group=minecraft

PrivateUsers=true
ProtectSystem=full
ProtectHome=true
ProtectKernelTunables=true
ProtectKernelModules=true
ProtectControlGroups=true

Type=forking
Restart=on-failure
RestartSec=20 5
ExecStart=/usr/bin/screen -h 2048 -dmS minecraft /usr/bin/java -Xms1536M -Xmx1536M -jar minecraft_server.jar nogui
#ExecStart=/bin/sh -c '/usr/bin/screen -DmS mc-%i /usr/bin/java -server -Xms512M -Xmx2048M -XX:+UseG1GC -XX:+CMSIncrementalPacing -XX:+CMSClassUnloadingEnabled -XX:ParallelGCThreads=2 -XX:MinHeapFreeRatio=5 -XX:MaxHeapFreeRatio=10 -jar $(ls -v | grep -i "FTBServer.*jar\|minecraft_server.*jar" | head -n 1) nogui'

ExecReload=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "reload"\\015'

ExecStop=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "say SERVER SHUTTING DOWN. Saving map..."\\015'
ExecStop=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "save-all"\\015'
ExecStop=/usr/bin/screen -p 0 -S minecraft -X eval 'stuff "stop"\\015'
ExecStop=/bin/sleep 10

[Install]
WantedBy=multi-user.target
EOF

# Autoshutdown script
sudo tee /srv/minecraft-server/autoshutdown.sh > /dev/null << EOF
#!/usr/bin/env bash

MINECRAFT_HOME="/srv/minecraft-server"
touch_file="\${MINECRAFT_HOME}/no_one_playing"

list_players() {
    screen -S minecraft -p 0 -X stuff "list^M"
    sleep 5
}

list_players;
last_log_line="\$(tail -n 1 \${MINECRAFT_HOME}/logs/latest.log)";
echo "\${last_log_line}"

regex="There are [0-9]+ of a max [0-9]+ players online"
regex2="There are 0 of a max [0-9]+ players online"

if [[ "\$last_log_line" =~ \$regex ]]; then
    if [[ "\$last_log_line" =~ \$regex2 ]]; then
        if [ -f "\${touch_file}" ]; then
            rm "\${touch_file}";
            poweroff;
        else
            touch "\${touch_file}";
        fi
    elif [ -f "\${touch_file}" ]; then
        rm "\${touch_file}";
    fi
elif [ -f "\${touch_file}" ]; then
    rm "\${touch_file}";
fi
EOF
sudo chmod +x /srv/minecraft-server/autoshutdown.sh

# Create CRON job
sudo tee /srv/minecraft-server/crontab > /dev/null << EOF
*/5 * * * * /srv/minecraft-server/autoshutdown.sh >/dev/null 2>&1
EOF
sudo crontab -u minecraft /srv/minecraft-server/crontab

# Start minecraft
sudo service minecraft start

# Enable on boot
sudo systemctl enable minecraft
