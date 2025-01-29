Write-Host "~~~ Authenticating GCP"
# Secrets must be redacted
# https://buildkite.com/docs/pipelines/managing-log-output#redacted-environment-variables
$PRIVATE_CI_GCS_CREDENTIALS_PATH = "kv/ci-shared/platform-ingest/gcp-platform-ingest-ci-service-account"
$env:PRIVATE_CI_GCS_CREDENTIALS_SECRET = vault kv get -field plaintext -format=json $PRIVATE_CI_GCS_CREDENTIALS_PATH
$env:PRIVATE_CI_GCS_CREDENTIALS_SECRET > ".\gcp.json"
$env:GOOGLE_APPLICATION_CREDENTIALS = (Get-Item -Path ".\gcp.json").FullName
gcloud auth activate-service-account --key-file="$env:GOOGLE_APPLICATION_CREDENTIALS"