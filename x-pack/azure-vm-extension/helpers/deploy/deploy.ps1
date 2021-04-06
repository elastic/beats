$json = Get-Content deploy.json | ConvertFrom-Json
$json.variables.version = '1.0.0.0'
$json.resources[1].properties.regions = '[ "east us", "west us"]'
$json | ConvertTo-Json -Depth 6 | Out-File test.json


function deploy
{
    Param
    (
        [Parameter(Mandatory=$true, Position=0)]
        [string] $dev,
        [Parameter(Mandatory=$true, Position=2)]
        [string] $version,
        [Parameter(Mandatory=$true, Position=3)]
        [string] $os,
        [Parameter(Mandatory=$true, Position=4)]
        $regions
    )

    $dReplacements = @{
        "\\u003c" = "<"
        "\\u003e" = ">"
        "\\u0027" = "'"
    }

    $sInFile = "deploy.json.template"
    $file= "deploy$os".ToLower()
    $sOutFile = "$file.json"

    $json = Get-Content $sInFile | ConvertFrom-Json
    $json.variables.version = "$version"
    if ($dev -eq "test") {
        $json.variables.publisherName = "Elastic.Test"
        $json.variables.typeName = "ElasticAgentTest.$os".ToLower()
    }
    elseif ($dev -eq "prod")
    {
        $json.variables.publisherName = "Elastic"
        $json.variables.typeName = "ElasticAgent.$os".ToLower()
    }
    $json.resources[1].properties.regions = "$regions"
    $json.resources[1].properties.supportedOS = "$os"
    $json | ConvertTo-Json -Depth 6 | Out-File $sOutFile

    $sRawJson = Get-Content -Path $sOutFile | Out-String
    foreach ($oEnumerator in $dReplacements.GetEnumerator())
    {
        $sRawJson = $sRawJson -replace $oEnumerator.Key, $oEnumerator.Value
    }
    $sRawJson | Out-File -FilePath $sOutFile

}

$regions = @("east us","la la")
deploy "test" "1.0.5" "Windows" $regions

