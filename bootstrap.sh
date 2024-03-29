#!/usr/bin/env bash

set -o nounset
set -o errexit
set -o pipefail
set -o verbose

exec > >(tee /var/log/user-data.log|logger -t user-data -s 2>/dev/console) 2>&1

MINECRAFT_HOME="/srv/minecraft-server"
MINECRAFT_USER="minecraft"
MINECRAFT_GROUP="minecraft"

#Setup Volume
# DEVICE_ID="$(sudo blkid | grep -v cloudimg-rootfs | cut -d":" -f 1)"
DEVICE_ID="/dev/nvme1n1"
sudo blkid

if [[ $(sudo blkid ${DEVICE_ID}) ]]; then
    echo "there is a filesystem"
else
    echo "no filesystem found"
    sudo mkfs -t ext4 ${DEVICE_ID}
fi
blkid

sudo mkdir ${MINECRAFT_HOME}
sudo tee -a /etc/fstab > /dev/null << EOF
$(blkid | grep "ext4" | grep -v cloudimg-rootfs | cut -d" " -f 2)  ${MINECRAFT_HOME} ext4 defaults,nofail 0 2
EOF
sudo mount -a

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
sudo apt install -qq -y openjdk-17-jdk-headless openjdk-17-jre-headless

# Download server
wget https://launcher.mojang.com/v1/objects/125e5adf40c659fd3bce3e66e67a16bb49ecc1b9/server.jar -O minecraft_server.1.18.1.jar
cat << EOF | shasum -a256 -c -
# 942256f0bfec40f2331b1b0c55d7a683b86ee40e51fa500a2aa76cf1f1041b38  minecraft_server.1.14.3.jar
# 80cf86dc2004ec6a2dc0183d1c75a9af3ba0669f7c332e4247afb1d76fb67e8a  minecraft_server.1.15.2.jar
# 2782d547724bc3ffc0ef6e97b2790e75c1df89241f9d4645b58c706f5e6c935b  minecraft_server.1.16.1.jar
# 32e450e74c081aec06dcfbadfa5ba9aa1c7f370bd869e658caec0c3004f7ad5b  minecraft_server.1.16.3.jar
# 444d30d903a1ef489b6737bb9d021494faf23434ca8568fd72ce2e3d40b32506  minecraft_server.1.16.4.jar
ebcd120ad81480b968a548df6ffb83b88075e95195c8ff63d461c9df4df5dbdf  minecraft_server.1.18.1.jar
EOF
sudo cp minecraft_server.1.18.1.jar "${MINECRAFT_HOME}/minecraft_server.jar"

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

regex="There are [0-9]+ of a max( of)? [0-9]+ players online"
regex2="There are 0 of a max( of)? [0-9]+ players online"

if [[ "\$last_log_line" =~ \$regex ]]; then
    if [[ "\$last_log_line" =~ \$regex2 ]]; then
        if [ -f "\${touch_file}" ]; then
            rm "\${touch_file}";
            sudo /sbin/poweroff;
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

# Allow minecraft user to run "poweroff"
sudo tee /etc/sudoers.d/shutdown > /dev/null << EOF
${MINECRAFT_USER} ALL=(root) NOPASSWD: /sbin/poweroff
EOF

# Create CRON job
sudo tee /srv/minecraft-server/crontab > /dev/null << EOF
*/5 * * * * /srv/minecraft-server/autoshutdown.sh >/dev/null 2>&1
EOF
sudo crontab -u minecraft /srv/minecraft-server/crontab

# Start minecraft
sudo service minecraft start

# Enable on boot
sudo systemctl enable minecraft
