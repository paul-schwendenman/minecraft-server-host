Install AWS CLI::

    sudo apt install python3-pip -y
    pip3 install --user awscli

    PATH="$HOME/.local/bin/:$PATH"
    aws configure

    sudo systemctl stop minecraft

Restore::

    sudo $(which aws) s3 sync s3://minecraft-backup-001 /srv/minecraft-server/

Backup::

    sudo $(which aws) s3 sync /srv/minecraft-server/ s3://minecraft-backup-001
