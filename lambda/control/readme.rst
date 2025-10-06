---------------
Control lambda
---------------

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
