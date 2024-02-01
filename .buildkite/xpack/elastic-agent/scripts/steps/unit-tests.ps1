$ErrorActionPreference = "Stop"

Write-Host "-- Fixing CRLF in git checkout --"
git config core.autocrlf input
git rm --quiet --cached -r .
git reset --quiet --hard

$env:GOTMPDIR = "$env:BUILDKITE_BUILD_CHECKOUT_PATH"
$env:SOURCE_DIR=".\\xpack\\elastic-agent"
$env:PIPELINE_DIR=".\\.buildkite\\xpack\\elastic-agent"

Write-Host "--- Build"
mage -d "$env:SOURCE_DIR" build

if ($LASTEXITCODE -ne 0) {
  exit 1 
}

Write-Host "--- Unit tests"
$env:TEST_COVERAGE = $true
$env:RACE_DETECTOR = $true
mage -d "$env:SOURCE_DIR" unitTest
# Copy coverage file to build directory so it can be downloaded as an artifact
# cp .\build\TEST-go-unit.cov coverage.out

if ($LASTEXITCODE -ne 0) {
  exit 1 
}


