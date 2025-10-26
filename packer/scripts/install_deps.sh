#!/usr/bin/env bash
set -euxo pipefail

# --- Base OS packages ---
sudo add-apt-repository -y universe
sudo apt-get update -qq
sudo DEBIAN_FRONTEND=noninteractive apt-get upgrade -y -qq
sudo apt-get install -y -qq \
  openjdk-21-jre-headless \
  screen unzip wget curl ca-certificates \
  python3-pip git build-essential jq xfsprogs

# --- Python tools (optional) ---
pip3 install --user --upgrade mcstatus nbtlib

# --- mcrcon ---
MCRCON_VERSION="0.7.2"
if ! command -v mcrcon >/dev/null 2>&1; then
  cd /tmp
  wget -q "https://github.com/Tiiffi/mcrcon/archive/refs/tags/v${MCRCON_VERSION}.tar.gz" -O mcrcon.tar.gz
  echo "1743b25a2d031b774e805f4011cb7d92010cb866e3b892f5dfc5b42080973270  mcrcon.tar.gz" | sha256sum -c -
  tar -xzf mcrcon.tar.gz
  cd "mcrcon-${MCRCON_VERSION}"
  make
  sudo make install
fi

# --- AWS CLI v2 ---
if ! command -v aws >/dev/null 2>&1; then
  echo "[*] Installing AWS CLI v2 with GPG verification"

  TMPDIR=$(mktemp -d)
  cd "$TMPDIR"

  # Fetch the CLI package and signature
  curl -s -O https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip
  curl -s -O https://awscli.amazonaws.com/awscli-exe-linux-x86_64.zip.sig

  # Import AWS CLI Team public key if not already present
  if ! gpg --list-keys A6310ACC4672475C >/dev/null 2>&1; then
    cat > awscliv2.pub <<'EOF'
-----BEGIN PGP PUBLIC KEY BLOCK-----
mQINBF2Cr7UBEADJZHcgusOJl7ENSyumXh85z0TRV0xJorM2B/JL0kHOyigQluUG
ZMLhENaG0bYatdrKP+3H91lvK050pXwnO/R7fB/FSTouki4ciIx5OuLlnJZIxSzx
PqGl0mkxImLNbGWoi6Lto0LYxqHN2iQtzlwTVmq9733zd3XfcXrZ3+LblHAgEt5G
TfNxEKJ8soPLyWmwDH6HWCnjZ/aIQRBTIQ05uVeEoYxSh6wOai7ss/KveoSNBbYz
gbdzoqI2Y8cgH2nbfgp3DSasaLZEdCSsIsK1u05CinE7k2qZ7KgKAUIcT/cR/grk
C6VwsnDU0OUCideXcQ8WeHutqvgZH1JgKDbznoIzeQHJD238GEu+eKhRHcz8/jeG
94zkcgJOz3KbZGYMiTh277Fvj9zzvZsbMBCedV1BTg3TqgvdX4bdkhf5cH+7NtWO
lrFj6UwAsGukBTAOxC0l/dnSmZhJ7Z1KmEWilro/gOrjtOxqRQutlIqG22TaqoPG
fYVN+en3Zwbt97kcgZDwqbuykNt64oZWc4XKCa3mprEGC3IbJTBFqglXmZ7l9ywG
EEUJYOlb2XrSuPWml39beWdKM8kzr1OjnlOm6+lpTRCBfo0wa9F8YZRhHPAkwKkX
XDeOGpWRj4ohOx0d2GWkyV5xyN14p2tQOCdOODmz80yUTgRpPVQUtOEhXQARAQAB
tCFBV1MgQ0xJIFRlYW0gPGF3cy1jbGlAYW1hem9uLmNvbT6JAlQEEwEIAD4CGwMF
CwkIBwIGFQoJCAsCBBYCAwECHgECF4AWIQT7Xbd/1cEYuAURraimMQrMRnJHXAUC
aGveYQUJDMpiLAAKCRCmMQrMRnJHXKBYD/9Ab0qQdGiO5hObchG8xh8Rpb4Mjyf6
0JrVo6m8GNjNj6BHkSc8fuTQJ/FaEhaQxj3pjZ3GXPrXjIIVChmICLlFuRXYzrXc
Pw0lniybypsZEVai5kO0tCNBCCFuMN9RsmmRG8mf7lC4FSTbUDmxG/QlYK+0IV/l
uJkzxWa+rySkdpm0JdqumjegNRgObdXHAQDWlubWQHWyZyIQ2B4U7AxqSpcdJp6I
S4Zds4wVLd1WE5pquYQ8vS2cNlDm4QNg8wTj58e3lKN47hXHMIb6CHxRnb947oJa
pg189LLPR5koh+EorNkA1wu5mAJtJvy5YMsppy2y/kIjp3lyY6AmPT1posgGk70Z
CmToEZ5rbd7ARExtlh76A0cabMDFlEHDIK8RNUOSRr7L64+KxOUegKBfQHb9dADY
qqiKqpCbKgvtWlds909Ms74JBgr2KwZCSY1HaOxnIr4CY43QRqAq5YHOay/mU+6w
hhmdF18vpyK0vfkvvGresWtSXbag7Hkt3XjaEw76BzxQH21EBDqU8WJVjHgU6ru+
DJTs+SxgJbaT3hb/vyjlw0lK+hFfhWKRwgOXH8vqducF95NRSUxtS4fpqxWVaw3Q
V2OWSjbne99A5EPEySzryFTKbMGwaTlAwMCwYevt4YT6eb7NmFhTx0Fis4TalUs+
j+c7Kg92pDx2uQ==
=OBAt
-----END PGP PUBLIC KEY BLOCK-----
EOF
    gpg --import awscliv2.pub
  fi

  # Verify the signature
  echo "[*] Verifying AWS CLI package signature..."
  gpg --verify awscli-exe-linux-x86_64.zip.sig awscli-exe-linux-x86_64.zip

  # Install only if verification passes
  unzip -q awscli-exe-linux-x86_64.zip -d .
  sudo ./aws/install

  cd /
  rm -rf "$TMPDIR"
fi
