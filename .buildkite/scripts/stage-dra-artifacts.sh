#!/usr/bin/env bash
##
##  Downloads Buildkite build artifacts, renames the dependencies CSV and
##  docker tarball filenames to what elastic/dra-prep-buildkite-plugin
##  expects, and stages the workflow's slice into artifacts/ for it.
##
##  On release branches, beats builds both snapshot and staging in the same
##  build, so filenames distinguish them: snapshot files contain "-SNAPSHOT"
##  (e.g. "-SNAPSHOT-linux-amd64.tar.gz", "-SNAPSHOT.csv", "-SNAPSHOT.zip");
##  staging files do not.
##
##  Runs in dra-prep-pipeline.yml, dynamically uploaded into the same build
##  that packaged the artifacts (see packaging.pipeline.yml's DRA publish
##  group), so the artifacts are already present in this build - no
##  cross-build download needed.
##

set -euo pipefail

WORKFLOW="${DRA_WORKFLOW:?DRA_WORKFLOW is required}"
VERSION_QUALIFIER="${VERSION_QUALIFIER:-}"

echo "--- Restoring artifacts from this build"
buildkite-agent artifact download "build/**/*" .

echo "--- Normalizing filenames (${WORKFLOW})"

# rename dependencies.csv to the versioned name dra-prep plugin's CSV classifier expects.
VERSION=$(make get-version)
FINAL_VERSION="${VERSION}-SNAPSHOT"
if [[ "${WORKFLOW}" != "snapshot" ]]; then
  FINAL_VERSION="${VERSION}"
fi
if [[ -n "${VERSION_QUALIFIER}" ]]; then
  FINAL_VERSION="${FINAL_VERSION}-${VERSION_QUALIFIER}"
fi

echo "Rename dependencies to ${FINAL_VERSION}"
mv build/distributions/dependencies.csv \
   build/distributions/dependencies-"${FINAL_VERSION}".csv

# rename docker files to support the unified release format.
# TODO: this could be supported by the package system itself
#       or the unified release process the one to do the transformation
#       See https://github.com/elastic/beats/pull/30895
find build/distributions -name '*linux-arm64.docker.tar.gz*' -print0 |
  while IFS= read -r -d '' file
  do
    echo "Rename file ${file}"
    mv "$file" "${file/linux-arm64.docker.tar.gz/docker-image-linux-arm64.tar.gz}"
  done

find build/distributions -name '*linux-amd64.docker.tar.gz*' -print0 |
  while IFS= read -r -d '' file
  do
    echo "Rename file ${file}"
    mv "$file" "${file/linux-amd64.docker.tar.gz/docker-image-linux-amd64.tar.gz}"
  done

echo "List all the files"
find build/distributions -type f -ls || true

echo "--- Preparing ${WORKFLOW} artifacts"
mkdir -p artifacts

if [[ "${WORKFLOW}" == "snapshot" ]]; then
  find build/distributions -type f -name "*-SNAPSHOT*" \
    -exec cp {} artifacts/ \;
else
  find build/distributions -type f ! -name "*-SNAPSHOT*" \
    -exec cp {} artifacts/ \;
fi

if ! ls artifacts/* >/dev/null 2>&1; then
  echo "ERROR: no ${WORKFLOW} artifacts found in artifacts/" >&2
  exit 1
fi

echo "Staged artifacts:"
ls -1 artifacts/
