----------------------
Minecraft Server Host
----------------------

This repository contains the blueprint for creating
an automated and configurable minecraft server on-demand.

The project provides a webapp that can be used to start an ec2
instance which has been configured to run a minecraft server.
The instance will autmatically shutdown after five minutes if
no players are online to help to reduce costs.

Setup
------

1. Install `terraform`_

    .. _terraform: https://www.terraform.io/downloads.html

#. Initialize terraform::

    terraform init

#. Create a ``terraform.tfvars`` file::

    tee terraform.tfvars << EOF
    dns_name        = "minecraft.example.com"
    webapp_dns_name = "www.minecraft.example.com"
    region          = "us-west-1"
    EOF

#. Edit the newly created tfvars file to provide your region
   preference and domain names.

#. Run apply to provision the infrastructure::

    terraform apply

#. Once the apply is complete your server should be running and
   the UI and lambda should be created to control the server.


Tearing down the server
------------------------

The resources can be removed by running::

    terraform destroy
