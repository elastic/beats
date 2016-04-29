param(
  [String] $l, [String] $url, 
  [String] $u, [String] $user, 
  [String] $i, [String] $index,
  [String] $d, [String] $dir,
  [switch] $h = $false, [switch] $help = $false
)
function prompt { "$pwd\" }

# The default value of the variable. Initialize your own variables here
$ELASTICSEARCH="http://localhost:9200"
$CURL="Invoke-RestMethod"
$KIBANA_INDEX=".kibana"
$SCRIPT=$MyInvocation.MyCommand.Name
$KIBANA_DIR=prompt

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
  -d | -dir
    Local directory where the dashboards, visualizations, searches and index pattern are saved.
    By default is $KIBANA_DIR.
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

if ($d -ne "" ){
  $KIBANA_DIR = $d
}
if ($dir -ne "" ){
  $KIBANA_DIR = $dir
}
if ($KIBANA_DIR -eq "") {
  Write-Error "Error: Missing local directory containing the Kibana dashboards, visualizations, searches and index
  patterns"
  print_usage
  exit 1
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

echo "Import dashboards from $KIBANA_DIR to $ELASTICSEARCH in $KIBANA_INDEX"

# Workaround for: https://github.com/elastic/beats-dashboards/issues/94
try {
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX" -Method PUT
} catch [System.Net.WebException] {
  # suppress 400 error, index might exist already
}
&$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/_mapping/search" -Method PUT -Body '{"search": {"properties": {"hits": {"type": "integer"}, "version": {"type": "integer"}}}}'

ForEach ($file in Get-ChildItem "$KIBANA_DIR/search/" -Filter *.json) {
  $name = [io.path]::GetFileNameWithoutExtension($file.Name)
  echo "Import search $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/search/$name" -Method PUT -Body $(Get-Content "$KIBANA_DIR/search/$file")
}

ForEach ($file in Get-ChildItem "$KIBANA_DIR/visualization/" -Filter *.json) {
  $name = [io.path]::GetFileNameWithoutExtension($file.Name)
  echo "Import visualization $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/visualization/$name" -Method PUT -Body $(Get-Content "$KIBANA_DIR/visualization/$file")
}

ForEach ($file in Get-ChildItem "$KIBANA_DIR/dashboard/" -Filter *.json) {
  $name = [io.path]::GetFileNameWithoutExtension($file.Name)
  echo "Import dashboard $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/dashboard/$name" -Method PUT -Body $(Get-Content "$KIBANA_DIR/dashboard/$file")
}

ForEach ($file in Get-ChildItem "$KIBANA_DIR/index-pattern/" -Filter *.json) {
  $json = Get-Content "$KIBANA_DIR/index-pattern/$file" -Raw | ConvertFrom-Json
  $name = $json.title
  echo "Import index-pattern $($name):"
  &$CURL -Headers $headers -Uri "$ELASTICSEARCH/$KIBANA_INDEX/index-pattern/$name" -Method PUT -Body $(Get-Content "$KIBANA_DIR/index-pattern/$file")
}
