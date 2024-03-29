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

#. Install `AWS CLI`_

    .. _AWS CLI: https://docs.aws.amazon.com/cli/latest/userguide/install-cliv2.html

#. Configure AWS CLI::

    $ aws configure
    AWS Access Key ID [None]: accesskey
    AWS Secret Access Key [None]: secretkey
    Default region name [None]: us-west-2
    Default output format [None]:

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

Terraform state backup
-----------------------

Terraform has the ability to store it's state in a remote store.
For more information, see the `documentation`_ for more information.

.. _documentation: https://www.terraform.io/docs/backends/types/remote.html

Example (i.e. ``backend.tf``)::

    terraform {
        backend "remote" {
            hostname     = "app.terraform.io"
            organization = "example"

            workspaces {
                name = "minecraft-server"
            }
        }
    }

Example costs (from 2020)
---------------------------

Since May, I've been running the server as a ``t3.medium`` instance to server about
4 active players at a time. Below I've included a chart with a couple sample months
representing different levels of activity.

======= ======= ==============
Use     Hours   Monthly Cost
======= ======= ==============
Low     6       $2.71
Medium  140     $10.59
Heavy   275     $21.89
======= ======= ==============


asdf - extendable version manager
----------------------------------

The ``asdf`` tool can be used to manage your local terraform installation.

Install plugin::

    asdf plugin-add terraform https://github.com/Banno/asdf-hashicorp.git

Install the tools::

    asdf install


aws cli
----------

Install::

    pip install --user awscli


Configure::

    aws configure