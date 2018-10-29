function Exec {
    [CmdletBinding()]
    param(
        [Parameter(Mandatory = $true)]
        [scriptblock]$cmd,
        [string]$errorMessage = ($msgs.error_bad_command -f $cmd)
    )

    try {
        $global:lastexitcode = 0
        & $cmd
        if ($lastexitcode -ne 0) {
            throw $errorMessage
        }
    }
    catch [Exception] {
        throw $_
    }
}

# Fix an issue with Windows' MAX_PATH limitation, this can cause compiler issues if the files are
# nested too deep. To solve this situation we are using 'subst' to map the workspace to a drive.
# If we have already mapped the workspace to a drive we just use the drive.
function SetupDrive {
  foreach($line in subst) {
    $drive = $line.SubString(0, 2)
    $path = $line.SubString(8)
    if ($path -eq $env:WORKSPACE) {
      echo "Found existing mapping $path to $drive"
      $env:WORKSPACE = $drive
      return
    }
  }
  # No existing drive found lets create a new mapping.
  AssignDrive
}
function AssignDrive() {
  $letters = @('A', 'B', 'C', 'D', 'E', 'F', 'G', 'H', 'I', 'J', 'K', 'L', 'M', 'N', 'O', 'P', 'Q', 'R', 'S', 'T', 'U', 'V', 'X', 'Y', 'Z')
  Foreach($l in $letters) {
    $drive = "${l}:"
    &subst $drive $env:WORKSPACE | Out-Null
    if ($lastexitcode -eq 0) {
      echo "Create new mapping for $env:WORKSPACE to $drive"
      $env:WORKSPACE = $drive
      return
    }
  }
  throw "cannot use 'subst' to create a drive for the workspace"
}

echo "Configure virtual drive"
SetupDrive

# CD into the new drive before running any tests
echo "Switching to $env:WORKSPACE\src\github.com\elastic\beats"
Set-Location "$env:WORKSPACE\src\github.com\elastic\beats"

# Setup Go.
$env:GOPATH = "$env:WORKSPACE\"
$env:PATH = "$env:GOPATH\bin;C:\tools\mingw64\bin;$env:PATH"
& gvm --format=powershell $(Get-Content .go-version) | Invoke-Expression

# Write cached magefile binaries to workspace to ensure
# each run starts from a clean slate.
$env:MAGEFILE_CACHE = "$env:WORKSPACE\.magefile"

# Configure testing parameters.
$env:TEST_COVERAGE = "true"
$env:RACE_DETECTOR = "true"

# Install mage from vendor.
exec { go install github.com/elastic/beats/vendor/github.com/magefile/mage } "mage install FAILURE"

if (Test-Path "$env:beat") {
    cd "$env:beat"
} else {
    echo "$env:beat does not exist"
    New-Item -ItemType directory -Path build | Out-Null
    New-Item -Name build\TEST-empty.xml -ItemType File | Out-Null
    exit
}

if (Test-Path "build") { Remove-Item -Recurse -Force build }
New-Item -ItemType directory -Path build\coverage | Out-Null
New-Item -ItemType directory -Path build\system-tests | Out-Null
New-Item -ItemType directory -Path build\system-tests\run | Out-Null

echo "Building fields.yml"
exec { mage fields } "mage fields FAILURE"

echo Get-Location
echo "Building $env:beat"
exec { mage build } "Build FAILURE"

echo "Unit testing $env:beat"
exec { mage goTestUnit } "mage goTestUnit FAILURE"

echo "System testing $env:beat"
# Get a CSV list of package names.
$packages = $(go list ./... | select-string -Pattern "/vendor/" -NotMatch | select-string -Pattern "/scripts/cmd/" -NotMatch)
$packages = ($packages|group|Select -ExpandProperty Name) -join ","
exec { go test -race -c -cover -covermode=atomic -coverpkg $packages } "go test -race -cover FAILURE"
Set-Location -Path tests/system
exec { nosetests --with-timer --with-xunit --xunit-file=../../build/TEST-system.xml } "System test FAILURE"
