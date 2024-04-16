set -euo pipefail

sudo apt-get update -y
DEBIAN_FRONTEND=noninteractive sudo apt-get install --no-install-recommends --yes msitools