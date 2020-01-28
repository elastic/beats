#!/bin/bash

# Description:
#    This script finds Jenkins jobs for a given Beats PR.
#
# Usage:
#    ./find_pr_jenkins_job.sh PR_NUMBER
#
# Example:
#    ./find_pr_jenkins_job.sh 15790
#
# Dependencies:
#    curl, jq

set -e

NUM_JOBS_TO_SEARCH=100

get_pr_from_input() {
    pr=$1
    if [ -z $pr ]; then
        echo "Usage: ./find_jenkins_job.sh PR_NUMBER" >&2
        exit 1
    fi

    echo $pr
}

find_latest_beats_pr_job() {
    curl -s 'https://beats-ci.elastic.co/job/elastic+beats+pull-request/api/json' | jq '.builds[0].number'
}

find_job_for_pr() {
    job=$1
    pr=$2
    
    found=$(curl -s "https://beats-ci.elastic.co/job/elastic+beats+pull-request/$job/api/json" \
                | jq -c ".actions[] | select(._class == \"org.jenkinsci.plugins.ghprb.GhprbParametersAction\").parameters[] | select(.name == \"ghprbPullId\" and .value == \"$pr\")" \
                | wc -l)

    echo $found
}
    
main() {
    pr=$(get_pr_from_input $1)
    echo "Searching last $NUM_JOBS_TO_SEARCH Jenkins jobs for PR number: $pr..."

    n=$(find_latest_beats_pr_job $pr)
    let e=$n-$NUM_JOBS_TO_SEARCH
    
    while [ $n -gt $e ]; do
        found=$(find_job_for_pr $n $pr)
        if [ $found -gt 0 ]; then
            echo "https://beats-ci.elastic.co/job/elastic+beats+pull-request/$n/"
        fi

        let n=$n-1
    done
}

main $1
