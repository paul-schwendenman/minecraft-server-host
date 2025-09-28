Minimal deploy
---------------

This terraform project is purely to test AMIs by building ephemeral instances

Helpful commands::

Install the dependencies::

    terraform init

See the resources that will be made::

    terraform plan

Build the server::

    terraform apply

Create a keypair::

    aws ec2 create-key-pair \
      --key-name minecraft-packer \
      --query 'KeyMaterial' \
      --output text > ~/.ssh/minecraft-packer.pem

Example `terraform.tfvars`::

    ami_id   = "ami-058a4a376ca9d2803"
    key_name = "minecraft-packer"


Example output from from `terraform apply`::

    Apply complete! Resources: 2 added, 0 changed, 0 destroyed.

    Outputs:

    instance_id = "i-01ba2e4f84d1eb980"
    public_dns = "ec2-3-135-194-106.us-east-2.compute.amazonaws.com"
    public_ip = "3.135.194.106"

::

    ssh -i ~/.ssh/minecraft-packer.pem ubuntu@<public_ip>

Clean up everything::

    terraform destroy


Use an AWS profile::

    aws configure --profile minecraft

And then append your commands with `AWS_PROFILE=minecraft`

---

::

    systemctl list-timers --all | grep autoshutdown
    journalctl -u autoshutdown.timer -f
    journalctl -u autoshutdown.service -f
    journalctl -u autoshutdown.service -n 50 --no-pager

::

    sudo systemctl status minecraft@default
    journalctl -u minecraft@default -n 50 --no-pager

Manual trigger::

    sudo systemctl start autoshutdown.service
