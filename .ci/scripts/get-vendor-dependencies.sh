#!/usr/bin/env bash
#
# Given the go module it will list all the dependencies that will be later on
# used by the CI to enable/disable specific stages as long as the changeset
# matches any of those patterns.
#

GO_VERSION=${GO_VERSION:?"GO_VERSION environment variable is not set"}
BEATS=${1:?"parameter missing."}
eval "$(gvm "${GO_VERSION}")"

go list -deps ./"${BEATS}" \
| grep 'elastic/beats' \
| sort \
| sed -e "s#github.com/elastic/beats/v8/##g" \
| awk '{print "^" $1 "/.*"}'
