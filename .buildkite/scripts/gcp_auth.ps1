Write-Host "~~~ Authenticating GCP"
# Secrets must be redacted
# https://buildkite.com/docs/pipelines/managing-log-output#redacted-environment-variables

$privateCIGCSServiceAccount = "kv/ci-shared/platform-ingest/gcp-platform-ingest-ci-service-account"
$tempFileName = "google-cloud-credentials.json"
$secretFileLocation = Join-Path $env:TEMP $tempFileName

$serviceAccountJsonSecret = Retry-Command -ScriptBlock {
    vault kv get -field=data -format=json $privateCIGCSServiceAccount | ConvertFrom-Json
    if ( -not $? ) { throw "Error during vault kv get" }
}

New-Item -ItemType File -Path $secretFileLocation >$null
$serviceAccountJsonPlaintextSecret = $serviceAccountJsonSecret.plaintext | ConvertTo-Json
Set-Content -Path $secretFileLocation -Value $serviceAccountJsonPlaintextSecret
if ( -not $?) { throw "Error retrieving the required field from the secret" }

gcloud auth activate-service-account --key-file $secretFileLocation
Remove-Item -Path $secretFileLocation -Force