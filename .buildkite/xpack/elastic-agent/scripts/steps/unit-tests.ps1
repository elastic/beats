$ErrorActionPreference = "Stop"

Write-Host "-- Fixing CRLF in git checkout --"
git config core.autocrlf input
git rm --quiet --cached -r .
git reset --quiet --hard

$env:GOTMPDIR = "$env:BUILDKITE_BUILD_CHECKOUT_PATH"

Write-Host "--- Build"
mage build

if ($LASTEXITCODE -ne 0) {
  exit 1 
}

Write-Host "--- Unit tests"
$env:TEST_COVERAGE = $true
$env:RACE_DETECTOR = $true
mage unitTest
# Copy coverage file to build directory so it can be downloaded as an artifact
cp .\build\TEST-go-unit.cov coverage.out

if ($LASTEXITCODE -ne 0) {
  exit 1 
}


