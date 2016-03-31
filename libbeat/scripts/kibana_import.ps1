param(
  [String] $l, [String] $url, 
  [String] $u, [String] $user, 
  [String] $i, [String] $index,
  [switch] $h = $false, [switch] $help = $false
)

# The default value of the variable. Initialize your own variables here
$ELASTICSEARCH="http://localhost:9200"
$CURL="Invoke-RestMethod"
$KIBANA_INDEX=".kibana"
$SCRIPT=$MyInvocation.MyCommand.Name

# Verify that Invoke-RestMethod is present. It was added in PS 3.
if (!(Get-Command $CURL -errorAction SilentlyContinue))
{
  Write-Error "$CURL cmdlet was not found. You may need to upgrade your PowerShell version."
  exit 1
}

function print_usage() {
  echo @"

Load the dashboards, visualizations and index patterns into the given
Elasticsearch instance.

Usage:
  $SCRIPT -url $ELASTICSEARCH -user admin -index $KIBANA_INDEX
Options:
  -h | -help
    Print the help menu.
  -l | -url
    Elasticseacrh URL. By default is $ELASTICSEARCH.
  -u | -user
    Username and password for authenticating to Elasticsearch using Basic
    Authentication. The username and password should be separated by a
    colon (i.e. "user:secret"). By default no username and password are
    used.
  -i | -index
    Kibana index pattern where to save the dashboards, visualizations,
    index patterns. By default is $KIBANA_INDEX.

"@
}

if ($help -or $h) {
  print_usage
  exit 0
}
if ($args -ne "") {
  Write-Error "Error: Unknown option $args"
  print_usage
  exit 1
}

if ($l -ne "" ) {
  $ELASTICSEARCH=$l
}
if ($url -ne "") {
  $ELASTICSEARCH=$url
}
if ($ELASTICSEARCH -eq "") {
  Write-Error "Error: Missing Elasticsearch URL"
  print_usage
  exit 1
}

if ($u -ne "" ){
  $user = $u
}
if ($user -ne "") {
  $base64AuthInfo = [Convert]::ToBase64String([Text.Encoding]::ASCII.GetBytes(("{0}" -f $user)))
  $headers=@{Authorization=("Basic $base64AuthInfo")}
}

if ($i -ne "") {
  $KIBANA_INDEX=$i
}
if ($index -ne "") {
  $KIBANA_INDEX=$index
}
if ($KIBANA_INDEX -eq "") {
  Write-Error "Error: Missing Kibana index pattern"
  print_usage
  exit 1
}

$DIR="./dashboards"
echo "Loading dashboards to $ELASTICSEARCH in $KIBANA_INDEX"  

ForEach ($file in Get-ChildItem "$DIR/search/" -Filter *.json) {
  $name = [io.path]::GetFileNameWithoutExtension($file.Name)
  echo "Loading search $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/search/$name" -Method PUT -Body $(Get-Content "$DIR/search/$file")
}

ForEach ($file in Get-ChildItem "$DIR/visualization/" -Filter *.json) {
  $name = [io.path]::GetFileNameWithoutExtension($file.Name)
  echo "Loading visualization $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/visualization/$name" -Method PUT -Body $(Get-Content "$DIR/visualization/$file")
}

ForEach ($file in Get-ChildItem "$DIR/dashboard/" -Filter *.json) {
  $name = [io.path]::GetFileNameWithoutExtension($file.Name)
  echo "Loading dashboard $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/dashboard/$name" -Method PUT -Body $(Get-Content "$DIR/dashboard/$file")
}

ForEach ($file in Get-ChildItem "$DIR/index-pattern/" -Filter *.json) {
  $json = Get-Content "$DIR/index-pattern/$file" -Raw | ConvertFrom-Json
  $name = $json.title
  echo "Loading index-pattern $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/index-pattern/$name" -Method PUT -Body $(Get-Content "$DIR/index-pattern/$file")
}
