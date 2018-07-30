#!/usr/bin/env bash

if [ "$TRAVIS_PULL_REQUEST" = "false" ]; then
    exit 0
fi

go get github.com/ewgRa/gocsfixer/cmd/gocsfixer
go get github.com/ewgRa/ci-utils/cmd/diff_liner
go get github.com/ewgRa/ci-utils/cmd/github_comments_diff

cat | gocsfixer -recommend -lint > /tmp/gocsfixer.json

set -e

if [ "$(cat /tmp/gocsfixer.json)" = "[]" ]; then
    exit 0
fi

# Get PR diff and transform file lines to diff lines
curl -sH "Accept: application/vnd.github.v3.diff.json" https://api.github.com/repos/$TRAVIS_REPO_SLUG/pulls/$TRAVIS_PULL_REQUEST > /tmp/pr.diff
cat /tmp/pr.diff | diff_liner > /tmp/pr_liner.json

go run script/csfixer/main.go -pr-liner /tmp/pr_liner.json -csfixer-comments /tmp/gocsfixer.json > /tmp/comments.json

cat /tmp/comments.json

EXIT_CODE=0

if [ "$(cat /tmp/comments.json)" != "[]" ]; then
    # Get exists comments, diff them with exists same comments and send comments as review
    curl -vvv https://api.github.com/repos/$TRAVIS_REPO_SLUG/pulls/$TRAVIS_PULL_REQUEST/comments > /tmp/pr_comments.json

    cat /tmp/pr_comments.json

    github_comments_diff -comments /tmp/comments.json -exists-comments /tmp/pr_comments.json > /tmp/send_comments.json

    if [ "$(cat /tmp/send_comments.json)" != "[]" ]; then
        curl -XPOST "https://github-api-bot.herokuapp.com/send_review?repo=$TRAVIS_REPO_SLUG&pr=$TRAVIS_PULL_REQUEST&body=Thanks%20for%20PR.%20Please%20check%20results%20of%20automatic%20CI%20checks" -d @/tmp/send_comments.json
    fi

    EXIT_CODE=1
fi

exit $EXIT_CODE
