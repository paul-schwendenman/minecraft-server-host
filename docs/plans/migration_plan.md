# Production Migration Plan

Migrate existing production AWS resources to the new modular Terraform setup under `infra/prod`.

## Summary

**What to import (preserve):**

- EBS volume (world data) - `aws_ebs_volume.minecraft_world`
- Route53 zone - `aws_route53_zone.primary`

**What to recreate fresh:**

- EC2 instance (will attach existing EBS)
- S3 buckets (webapp, backups, maps)
- Lambda functions (control, details, worlds)
- API Gateway v2 (replaces old REST API)
- CloudFront distribution
- IAM roles

**After migration:**

- Remove imported resources from old state
- Run `terraform destroy` on old setup to clean up remaining resources

## Step 0: Restore Old Terraform Config

Restore the old terraform configuration from git history:

```bash
# Restore old/ directory to infra/old
git checkout af6d79d~1 -- old/
mv old infra/old

# Or alternatively, just extract specific files:
mkdir -p infra/old
git show af6d79d~1:old/main.tf > infra/old/main.tf
git show af6d79d~1:old/variables.tf > infra/old/variables.tf
git show af6d79d~1:old/minecraft.tf > infra/old/minecraft.tf
git show af6d79d~1:old/lambda.tf > infra/old/lambda.tf
git show af6d79d~1:old/dns.tf > infra/old/dns.tf
git show af6d79d~1:old/webapp.tf > infra/old/webapp.tf
git show af6d79d~1:old/.terraform.lock.hcl > infra/old/.terraform.lock.hcl
```

## Step 1: Configure Terraform Cloud Backend

Add backend config to `infra/old/backend.tf` to pull state from Terraform Cloud:

```hcl
terraform {
  cloud {
    organization = "whatsdoom"
    workspaces {
      name = "minecraft-server"
    }
  }
  # ... rest of required_providers
}
```

Then initialize and verify:

```bash
cd infra/old
terraform init
terraform state list  # Verify we can see all resources
```

## Step 2: Extract Resource IDs

```bash
cd infra/old

# EBS Volume (old name is minecraft_world)
terraform state show aws_ebs_volume.minecraft_world | grep "^id"

# Route53 Zone
terraform state show aws_route53_zone.primary | grep -E "^(id|zone_id)"

# Note the availability zone (must match new EC2)
terraform state show aws_ebs_volume.minecraft_world | grep "availability_zone"
```

## Step 3: Create EBS Snapshot (Safety Backup)

```bash
aws ec2 create-snapshot --volume-id vol-XXXXX --description "Pre-migration backup"
```

## Step 4: Create `infra/prod/main.tf`

Copy structure from `infra/test/main.tf` with production values:

```hcl
variable "aws_profile" {
  default = "minecraft"
}

provider "aws" {
  region  = "us-east-2"
  profile = var.aws_profile
}

data "aws_ami" "minecraft" {
  most_recent = true
  owners      = ["self"]
  filter {
    name   = "name"
    values = ["minecraft-ubuntu-*"]
  }
}

# Route53 zone - managed by prod (imported from old setup)
resource "aws_route53_zone" "prod" {
  name = "minecraft.paulandsierra.com"
}

module "networking" {
  source             = "../modules/networking"
  name               = "minecraft-prod"
  vpc_cidr           = "10.1.0.0/16"        # Different from test
  public_subnet_cidr = "10.1.1.0/24"
}

module "s3_buckets" {
  source = "../modules/s3_buckets"
  name   = "minecraft-prod"
}

module "ec2_role" {
  source        = "../modules/ec2_role"
  name          = "minecraft-prod"
  map_bucket    = module.s3_buckets.map_bucket_name
  backup_bucket = module.s3_buckets.backup_bucket_name
}

module "mc_stack" {
  source           = "../modules/mc_stack"
  name             = "minecraft-prod"
  ami_id           = data.aws_ami.minecraft.id
  instance_type    = "t3.small"              # Adjust as needed
  vpc_id           = module.networking.vpc_id
  subnet_id        = module.networking.public_subnet_id
  key_name         = "minecraft-packer"
  root_volume_size = 8
  data_volume_size = 32                      # Match existing EBS size
  ssh_cidr_blocks  = ["YOUR_IP/32"]
  world_version    = "1.21.8"
  availability_zone = "us-east-2b"           # MUST match existing EBS

  iam_instance_profile = module.ec2_role.instance_profile_name
  map_bucket           = module.s3_buckets.map_bucket_name
}

module "api_lambda" {
  source          = "../modules/api_lambda"
  name            = "minecraft-prod"
  instance_id     = module.mc_stack.instance_id
  instance_arn    = module.mc_stack.instance_arn
  dns_name        = "mc.${aws_route53_zone.prod.name}"
  zone_id         = aws_route53_zone.prod.zone_id
  map_bucket_name = module.s3_buckets.map_bucket_name
  cors_origin     = "*"  # Or specific origin
}

module "web_ui" {
  source                    = "../modules/web_ui"
  name                      = "minecraft-prod"
  api_endpoint              = module.api_lambda.api_endpoint
  webapp_bucket_name        = module.s3_buckets.webapp_bucket_name
  webapp_bucket_domain_name = module.s3_buckets.webapp_bucket_domain_name
  map_bucket_name           = module.s3_buckets.map_bucket_name
  map_bucket_domain_name    = module.s3_buckets.map_bucket_domain_name
  geo_whitelist             = []  # No restrictions for prod
}

module "dns_records" {
  source         = "../modules/dns_records"
  zone_id        = aws_route53_zone.prod.zone_id
  dns_name       = "mc.${aws_route53_zone.prod.name}"
  ipv4_addresses = [module.mc_stack.public_ip]
  ipv6_addresses = module.mc_stack.ipv6_addresses
}

output "server_public_ip" { value = module.mc_stack.public_ip }
output "api_endpoint" { value = module.api_lambda.api_endpoint }
output "webapp_url" { value = module.web_ui.webapp_url }
```

## Step 5: Initialize Terraform

```bash
cd infra/prod
terraform init
```

## Step 6: Import Resources

```bash
# Import existing EBS volume
terraform import 'module.mc_stack.aws_ebs_volume.world' vol-XXXXX

# Import Route53 zone
terraform import 'aws_route53_zone.prod' ZXXXXXXXXXXXXX
```

## Step 7: Plan and Apply

```bash
# Review what will be created
terraform plan

# Apply (will create new resources, attach existing EBS)
terraform apply
```

**Expected behavior:**

- EBS volume: No changes (imported)
- Route53 zone: No changes (imported)
- EC2 instance: Created, attaches imported EBS
- Everything else: Created fresh

## Step 8: Verify

1. SSH to new server, verify world data mounted correctly
2. Start Minecraft server, verify world loads
3. DNS will automatically point to new server IP

## Step 9: Remove Imported Resources from Old State

Now remove the imported resources from `infra/old` state so they don't get destroyed:

```bash
cd infra/old

# Remove EBS volume from old state (now managed by infra/prod)
terraform state rm 'aws_ebs_volume.minecraft_world'

# Remove Route53 zone from old state (now managed by infra/prod)
terraform state rm 'aws_route53_zone.primary'

# Also remove the volume attachment (references both EBS and old EC2)
terraform state rm 'aws_volume_attachment.ebs_att'

# Remove DNS records and SSM parameters that reference the zone
terraform state rm 'aws_route53_record.minecraft'
terraform state rm 'aws_route53_record.www'
terraform state rm 'aws_ssm_parameter.minecraft_route53_zone_id'
terraform state rm 'aws_ssm_parameter.minecraft_dns_name'
```

## Step 10: Destroy Old Resources

Now you can safely destroy the remaining old resources:

```bash
# Preview what will be destroyed (should NOT include EBS or zone)
terraform plan -destroy

# Destroy remaining resources (old EC2, Lambda, API Gateway, CloudFront, S3)
terraform destroy
```

**Remaining resources that will be destroyed:**

- Old EC2 instance
- Old Lambda function (`MinecraftAPI`)
- Old API Gateway REST API
- Old CloudFront distribution
- Old S3 buckets (lambda code bucket, webapp bucket if separate)
- Old IAM roles and policies

## Critical Files

| File                             | Action                              |
| -------------------------------- | ----------------------------------- |
| `infra/old/`                     | Restore from git (commit af6d79d~1) |
| `infra/old/main.tf`              | Add Terraform Cloud backend config  |
| `infra/prod/main.tf`             | Create (based on test/main.tf)      |
| `infra/test/main.tf`             | Reference template                  |
| `infra/modules/mc_stack/main.tf` | Has `prevent_destroy` on EBS        |

## Safety Notes

1. **EBS volume has `prevent_destroy = true`** in mc_stack module - Terraform cannot accidentally delete it
2. **Create snapshot before migration** as additional safety
3. **Test environment unaffected** - separate VPC, separate resources
4. **Route53 zone ownership** - prod will own the zone; test uses `data` block to reference it
5. **State file backup** - consider backing up old state file before removing resources from it
6. **Order matters** - always import into new state BEFORE removing from old state

## Follow-up Tasks

1. **Make user_data scripts configurable for test vs prod**
   - Problem: The `create-world.sh` script runs on every new instance, which is fine for test but not for prod (where we import an existing EBS with world data)
   - Solution: Add a variable like `create_new_world = true/false` to mc_stack module to conditionally run create-world.sh
   - Also consider: Different world versions, seeds, etc. between environments

2. **EBS attachment race condition**
   - Problem: Cloud-init user_data runs before Terraform completes the EBS volume attachment
   - The mount-ebs.sh script waits up to 120 seconds, but attachment can take longer
   - Workaround: After first boot, manually run `/usr/local/bin/mount-ebs.sh` if volume wasn't detected
   - Future fix: Consider using a systemd service with `RequiresMountsFor` or increase wait timeout

3. **Filesystem type mismatch**
   - The `mount-ebs.sh` script creates new volumes with xfs (`mkfs.xfs`)
   - The imported production EBS volume uses ext4
   - The fstab entry added by mount-ebs.sh hardcodes `xfs` as the filesystem type
   - Fix needed: Either detect filesystem type dynamically, or use `auto` in fstab instead of `xfs`
