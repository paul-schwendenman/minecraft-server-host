----------------------
Minecraft Server Host
----------------------

This repository contains the blueprint for creating
an automated and configurable minecraft server on-demand.

The project provides a webapp that can be used to start an ec2
instance which has been configured to run a minecraft server.
The instance will automatically shutdown after five minutes if
no players are online to help to reduce costs.


Packer
-------

Initialize
===========

::

    packer init minecraft.pkr.hcl

Checks
=======

::

    packer fmt minecraft.pkr.hcl
    packer validate minecraft.pkr.hcl

Build
======

::

    aws configure --profile minecraft
    AWS_PROFILE=minecraft packer build minecraft.pkr.hcl


::

    packer build -on-error=ask minecraft.pkr.hcl

::

    packer build -debug minecraft.pkr.hcl

Testing
========

::

    cd infra/test
    terraform init
    terraform plan
    terraform apply

::

    cd ui/
    pnpm build
    aws s3 sync ./dist/ s3://minecraft-test-webapp
