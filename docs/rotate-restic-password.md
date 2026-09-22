# Rotating the restic backup password

The `restic_password` Terraform variable is the encryption password for the world
backup repository (`s3:s3.us-east-2.amazonaws.com/minecraft-<env>-backups`). It is
written to `/etc/minecraft.env` on the instance by user_data
(`infra/modules/mc_stack/main.tf`) and read by `minecraftctl backup ...`.

**Do not just change the variable.** The password unlocks the existing repository,
so changing only the variable makes every existing backup unreadable. Rotate it in
the repo first (add a new key, verify, remove the old one), then update Terraform.

Restic keeps passwords as key slots that all unlock the same master key. Rotating
adds and removes slots. Nothing is re-encrypted.

## Prerequisites

- `restic` installed locally (`brew install restic`; the AMI ships 0.18.1)
- AWS credentials with read/write on the backups bucket
- The current password (`infra/<env>/terraform.tfvars`)
- The instance is **stopped**, so no backup runs during the rotation

## Steps

Replace `prod` with `test` for the test environment.

```bash
export AWS_REGION=us-east-2
export RESTIC_REPOSITORY=s3:s3.us-east-2.amazonaws.com/minecraft-prod-backups

# Generate the new password into a private file (keeps it out of shell history)
umask 077
openssl rand -hex 32 > /tmp/restic-new.pw

# Authenticate with the OLD password
read -rs -p "old password: " RESTIC_PASSWORD; export RESTIC_PASSWORD

# 1. List keys; note the ID of the current one (marked with *)
restic key list

# 2. Add the new key
restic key add --new-password-file /tmp/restic-new.pw

# 3. Verify the NEW password works BEFORE removing anything
unset RESTIC_PASSWORD
export RESTIC_PASSWORD_FILE=/tmp/restic-new.pw
restic key list          # should show two keys
restic check

# 4. Remove the old key (restic refuses to remove the key you authenticated with)
restic key remove <OLD_KEY_ID>
restic key list          # should show one key
```

Shortcut: `restic key passwd --new-password-file /tmp/restic-new.pw` does the add and
remove in one step (run it authenticated with the old password), but you still
should run the verify step afterward.

## Update Terraform

1. Copy the new password from `/tmp/restic-new.pw` into a password manager.
2. Set `restic_password` in `infra/<env>/terraform.tfvars` (gitignored) to the new value.
3. `cd infra/<env> && terraform plan`. The only difference from before should be
   the user_data on `module.mc_stack.aws_instance.minecraft`, which is folded
   into any planned instance replacement.
4. Apply. A new instance writes `RESTIC_PASSWORD` to `/etc/minecraft.env` on first
   boot. The env file lives on the root volume, so an existing instance keeps the
   old value (the user_data script skips the append if the variable is already
   set). If an instance is not being replaced, edit
   `/etc/minecraft.env` on it by hand.
5. `shred -u /tmp/restic-new.pw` (or `rm -P` on macOS).

## Verify on the server

```bash
ssh <instance>
minecraftctl backup list
minecraftctl backup check
```

Both should succeed with the new password.

## Gotchas

- **Never remove the old key until the new one is verified** (step 3). If every
  valid password is lost, the backups cannot be recovered.
- **Order matters relative to instance replacement.** If a new instance boots with
  the old password after the old key was removed, its backups fail. Update tfvars
  before applying.
- **Still exposed elsewhere.** The password sits in user_data
  (readable via `ec2:DescribeInstanceAttribute`) and in `terraform.tfstate` /
  `terraform.tfstate.backup` (local, gitignored). A later apply overwrites the
  `.backup`. Moving the secret to SSM Parameter Store or Secrets Manager,
  fetched by the instance role at boot, would remove this.
- **Test uses its own variable.** `infra/test` has a separate `restic_password`.
  If it reuses the prod value, rotate that repo too.
