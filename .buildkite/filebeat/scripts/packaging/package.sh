#!/usr/bin/env bash

set -euo pipefail

echo ":: Start Packaging ::"
mage -d filebeat package

calculate_tags() {
  if [ "$snapshot" = true ]; then
    aliasVersion="${VERSION%.*}${IMG_POSTFIX}"
    sourceTag+=${IMG_POSTFIX}
  fi

  local tags="${BUILDKITE_COMMIT}"

#  if isPR; then
#    tags+=("pr-${CHANGE_ID}")
#  else
#    tags+=("${sourceTag}")
#  fi
#
#  if ! isPR && [ -n "$(aliasVersion)" ]; then
#    tags+=("${aliasVersion}")
#  fi
#
#  echo "${tags[@]}"
}

#buildkite-agent annotate "Tag '$TAG' has been created." --style 'success' --context 'ctx-success'

#set_git_config() {
#  git config user.name "${GITHUB_USERNAME_SECRET}"
#  git config user.email "${GITHUB_EMAIL_SECRET}"
#}
#
#set_git_config
