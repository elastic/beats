#!/usr/bin/env bash

STDIN=`cat | $GOCSFIXER -recommend -lint`
EXIT_CODE=$?

set -e

echo "$STDIN"

if [ "$STDIN" = "" ]; then
    exit 0
fi

if [ "$TRAVIS_PULL_REQUEST" = "false" ]; then
    exit 0
fi

if [ "$GOCSFIXER_GITHUB_TOKEN" = "" ]; then
    exit 0
fi

json_escape () {
    printf '%s' "$1" | python -c 'import json,sys; print(json.dumps(sys.stdin.read()))'
}

BODY=$(json_escape "Gocsfixer check:
\`\`\`
$STDIN
\`\`\`")

curl -s -H "Authorization: token $GOCSFIXER_GITHUB_TOKEN" -X POST -d "{\"body\": $BODY}" "https://api.github.com/repos/$TRAVIS_REPO_SLUG/issues/$TRAVIS_PULL_REQUEST/comments" | ((grep created_at -q) || (echo "Error when try post comment to Pull Request" && false))

exit $EXIT_CODE
