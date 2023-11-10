$ErrorActionPreference = "Stop" # set -e
# Forcing to checkout again all the files with a correct autocrlf.
# Doing this here because we cannot set git clone options before.
function fixCRLF {
    Write-Host "-- Fixing CRLF in git checkout --"
    git config core.autocrlf input
    git rm --quiet --cached -r .
    git reset --quiet --hard
}

function withGoJUnitReport {
    Write-Host "-- Install go-junit-report --"
    go install github.com/jstemmer/go-junit-report/v2@latest
}

Write-Host "--- Prepare enviroment"
fixCRLF
withGoJUnitReport

Write-Host "--- Run test"
$ErrorActionPreference = "Continue" # set +e
mkdir -p build
$OUT_FILE="output-report.out"
go test "./..." -v > $OUT_FILE
$EXITCODE=$LASTEXITCODE
$ErrorActionPreference = "Stop"

# Buildkite collapse logs under --- symbols
# need to change --- to anything else or switch off collapsing (note: not available at the moment of this commit)
$contest = Get-Content $OUT_FILE
foreach ($line in $contest) {
    $changed = $line -replace '---', '----'
    Write-Host $changed
}

Get-Content $OUT_FILE | go-junit-report > "uni-junit-win.xml"
Get-Content "uni-junit-win.xml" -Encoding Unicode | Set-Content -Encoding UTF8 "junit-win.xml"
Remove-Item "uni-junit-win.xml", "$OUT_FILE"

Exit $EXITCODE
