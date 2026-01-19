# Production Debugging

Debug issues on the production Minecraft server.

## Deploying minecraftctl to Prod

```bash
# Build for Linux
cd minecraftctl && make build-linux-amd64

# Copy to server (must be running - see below)
scp minecraftctl-linux-amd64 minecraft.paulandsierra.com:/tmp/

# Install on server
ssh minecraft.paulandsierra.com "sudo install -m 755 /tmp/minecraftctl-linux-amd64 /usr/local/bin/minecraftctl"
```

## Hot Updating Scripts with Packer

Use `packer/ssh.pkr.hcl` to deploy updated scripts to the running server without building a new AMI:

```bash
cd packer

# Copy all scripts to /tmp/scripts/ on the server
packer build \
  -var "test_host=minecraft.paulandsierra.com" \
  -var "test_private_key=~/.ssh/minecraft-packer.pem" \
  ssh.pkr.hcl
```

To run specific install scripts, uncomment the relevant provisioners in `ssh.pkr.hcl`:

```hcl
provisioner "shell" { script = "scripts/minecraft/install_map_backup.sh" }
```

This is useful for:
- Testing script changes on prod before baking a new AMI
- Hot-fixing issues without a full AMI rebuild
- Iterating on provisioning scripts during development

## Prerequisites

The EC2 server may be stopped. Before SSH:

1. **Check status:** `curl -s https://www.minecraft.paulandsierra.com/api/status`
2. **Start if needed:** `curl -s -X POST https://www.minecraft.paulandsierra.com/api/start`
3. **Sync DNS:** `curl -s -X POST https://www.minecraft.paulandsierra.com/api/syncdns`
4. **Wait for running state** before SSH

The firewall allows SSH from specific IPs only. If blocked, user must update Terraform.

## Control API

Base URL: `https://www.minecraft.paulandsierra.com/api/`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/status` | GET | Server state, IP, DNS record |
| `/start` | POST | Start EC2 instance |
| `/stop` | POST | Stop EC2 instance |
| `/syncdns` | POST | Sync Route53 DNS with current IP |

## Worlds API

Base URL: `https://maps.minecraft.paulandsierra.com/api/`

| Endpoint | Method | Description |
|----------|--------|-------------|
| `/worlds` | GET | List all worlds with previewUrl, mapUrl |
| `/worlds/{name}` | GET | World details with maps array |
| `/worlds/{name}/{map}` | GET | Individual map details |

## SSH Commands

```bash
# SSH into prod (after server started)
ssh minecraft.paulandsierra.com

# World info (needs sudo for level.dat access)
sudo minecraftctl world info <world>

# Regenerate map preview
sudo minecraftctl map preview <world> <map> --log-level verbose

# Rebuild manifests (regenerates all previews)
sudo minecraftctl map manifest <world>

# Sync maps to S3
sudo minecraftctl map backup start <world>

# Check systemd service logs
sudo journalctl -u minecraft-map-backup@<world> -n 50 --no-pager
```

## S3/CloudFront Structure

Maps bucket: `minecraft-prod-maps`
CloudFront: `maps.minecraft.paulandsierra.com`

```
maps/
├── world_manifest.json      # All worlds list
├── {world}/
│   ├── manifest.json        # World metadata
│   ├── preview.png          # World preview (copied from overworld)
│   └── {map}/
│       ├── manifest.json    # Map metadata
│       ├── preview.png      # Map preview
│       └── [tiles/]         # uNmINeD web tiles
```

Preview URLs are relative paths like `/maps/default/preview.png` served via CloudFront.

## Common Issues

### Blank preview images
- Check spawn coordinates: `sudo minecraftctl world info <world>`
- Spawn at (0,0,0) usually means format issue or unexplored area
- Minecraft 1.21+ uses `Data.spawn.pos` array instead of `Data.SpawnX/Y/Z`
- Verify chunks exist at spawn location in region files

### Preview not updating
1. Regenerate: `sudo minecraftctl map preview <world> overworld`
2. Rebuild manifest: `sudo minecraftctl map manifest <world>`
3. Sync to S3: `sudo minecraftctl map backup start <world>`
4. CloudFront may cache - check S3 directly or wait for TTL
