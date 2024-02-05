# Set error handling
$ErrorActionPreference = "Stop"

# Define a function to checkout and merge
function Checkout-Merge {
    param (
        [string]$targetBranch,
        [string]$prCommit,
        [string]$mergeBranch
    )

    if (-not $targetBranch) {
        Write-Host "No pull request target branch"
        exit 1
    }

    git fetch -v origin $targetBranch
    git checkout FETCH_HEAD
    Write-Host "Current branch: $(git rev-parse --abbrev-ref HEAD)"

    # Create a temporary branch to merge the PR with the target branch
    git checkout -b $mergeBranch
    Write-Host "New branch created: $(git rev-parse --abbrev-ref HEAD)"

    # Set author identity so it can be used for git merge
    git config user.name "github-merged-pr-post-checkout"
    git config user.email "auto-merge@buildkite"

    git merge --no-edit $prCommit

    if ($LASTEXITCODE -ne 0) {
        $mergeResult = $LASTEXITCODE
        Write-Host "Merge failed: $mergeResult"
        git merge --abort
        exit $mergeResult
    }
}

$pullRequest = $env:BUILDKITE_PULL_REQUEST

if ($pullRequest -eq "false") {
    Write-Host "Not a pull request, skipping"
    exit 0
}

$targetBranch = $env:BUILDKITE_PULL_REQUEST_BASE_BRANCH
$prCommit = $env:BUILDKITE_COMMIT
$prId = $env:BUILDKITE_PULL_REQUEST
$mergeBranch = "pr_merge_$prId"

Checkout-Merge $targetBranch $prCommit $mergeBranch

Write-Host "Commit information"
git --no-pager log --format=%B -n 1

Write-Host "Fixing CRLF in git checkout --"
git config core.autocrlf true

# Ensure Buildkite groups are rendered
Write-Host ""
