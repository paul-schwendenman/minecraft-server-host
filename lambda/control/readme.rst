---------------
Control lambda
---------------

FastAPI + Mangum lambda for EC2 start/stop/status and Route53 DNS sync.

Deployment
----------

**CI/CD (recommended):** Push changes to ``master`` branch. The ``lambdas-deploy.yml`` workflow automatically builds and deploys.

**Manual build:**

::

    uv export --frozen --no-dev --no-editable -o requirements.txt
    uv pip install \
        --no-installer-metadata \
        --no-compile-bytecode \
        --python-platform x86_64-manylinux2014 \
        --python 3.13 \
        --target packages \
        -r requirements.txt

::

    cd packages
    zip -r ../package.zip .
    cd ..

::

    zip -r package.zip app

Or use ``make control`` from repo root.

Local Development
-----------------

Localstack commands::

    uv run localstack start -d
    uv run tflocal plan
