#!/usr/bin/env bash
# Opens a PR in elastic/beats that bumps the npcap OEM installer to a new version.
# Triggered by the elastic/golang-crossbuild bump-npcap pipeline.
#
# Required environment:
#   NEW_NPCAP_VERSION - the npcap version to bump to (e.g. "1.88")
#   GITHUB_TOKEN      - provided by the elastic/vault-github-token Buildkite plugin,
#                       which also configures the elastic-ci[bot] git identity
set -euo pipefail

VERSION="${NEW_NPCAP_VERSION:?NEW_NPCAP_VERSION must be set}"
DATESTAMP=$(date -u '+%Y%m%d%H%M')
BRANCH="automation/bump-npcap-${VERSION}-${DATESTAMP}"
PCAP_GO="x-pack/packetbeat/scripts/mage/pcap.go"
LICENSE_FILE="x-pack/packetbeat/npcap/installer/LICENSE"

echo "--- Checking current npcap version"
CURRENT=$(grep 'NpcapVersion' "${PCAP_GO}" | grep -oP '"\K[^"]+')
echo "Current: ${CURRENT}"
echo "Target:  ${VERSION}"

if [ "${VERSION}" = "${CURRENT}" ]; then
  echo "Already at ${VERSION} — nothing to do."
  exit 0
fi

echo "--- Creating branch ${BRANCH}"
git checkout -b "${BRANCH}"

echo "--- Updating NpcapVersion in ${PCAP_GO}"
sed -i "s/NpcapVersion = \"${CURRENT}\"/NpcapVersion = \"${VERSION}\"/" "${PCAP_GO}"

echo "--- Updating Version in ${LICENSE_FILE}"
sed -i "s/^Version: ${CURRENT}/Version: ${VERSION}/" "${LICENSE_FILE}"

echo "--- Adding changelog fragment"
TIMESTAMP=$(date +%s)
FRAGMENT="changelog/fragments/${TIMESTAMP}-bump-npcap-${VERSION}.yaml"
cat > "${FRAGMENT}" <<EOF
kind: enhancement
summary: Bump npcap OEM installer to version ${VERSION}.
component: packetbeat
EOF

echo "--- Committing changes"
git add "${PCAP_GO}" "${LICENSE_FILE}" "${FRAGMENT}"
git commit -m "Bump npcap OEM installer to version ${VERSION}

Automated version bump from the elastic/golang-crossbuild npcap pipeline."

echo "--- Pushing branch"
git push origin "${BRANCH}"

echo "--- Opening PR"
gh pr create \
  --repo elastic/beats \
  --base main \
  --head "${BRANCH}" \
  --title "Bump npcap OEM installer to version ${VERSION}" \
  --body "Bumps the npcap OEM installer to v${VERSION}.

The OEM installer artifact has been uploaded to the private GCS store by the \`golang-crossbuild-bump-npcap\` automation.

https://github.com/nmap/npcap/releases/tag/v${VERSION}" \
  --label automation \
  --label dependencies \
  --label backport-skip
