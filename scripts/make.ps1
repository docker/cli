<#
.NOTES
    Summary: Windows native build script.

             It does however provided the minimum necessary to support parts of local Windows
             development and Windows to Windows CI.

             Usage Examples (run from repo root):
                "scripts/make.ps1 -Client" to build docker.exe client 64-bit binary (remote repo)
                "scripts/make.ps1 -TestUnit" to run unit tests
                "scripts/make.ps1 -Daemon -TestUnit" to build the daemon and run unit tests
                "scripts/make.ps1 -All" to run everything this script knows about that can run in a container
                "scripts/make.ps1" to build the daemon binary (same as -Daemon)
                "scripts/make.ps1 -Binary" shortcut to -Client and -Daemon

.PARAMETER Binary
     Builds the client and daemon binaries. A convenient shortcut to `make.ps1 -Client -Daemon`.

.PARAMETER Race
     Use -race in go build and go test.

.PARAMETER Noisy
     Use -v in go build.

.PARAMETER ForceBuildAll
     Use -a in go build.

.PARAMETER NoOpt
     Use -gcflags -N -l in go build to disable optimisation (can aide debugging).

.PARAMETER CommitSuffix
     Adds a custom string to be appended to the commit ID (spaces are stripped).

.PARAMETER TestUnit
     Runs unit tests.

.PARAMETER All
     Runs everything this script knows about that can run in a container.


TODO
- Unify the head commit
- Add golint and other checks (swagger maybe?)

#>


param(
    [Parameter(Mandatory=$False)][switch]$Binary,
    [Parameter(Mandatory=$False)][switch]$Race,
    [Parameter(Mandatory=$False)][switch]$Noisy,
    [Parameter(Mandatory=$False)][switch]$ForceBuildAll,
    [Parameter(Mandatory=$False)][switch]$NoOpt,
    [Parameter(Mandatory=$False)][switch]$TestUnit,
    [Parameter(Mandatory=$False)][switch]$All
)

$ErrorActionPreference = "Stop"
$ProgressPreference = "SilentlyContinue"
$pushed=$False  # To restore the directory if we have temporarily pushed to one.

# Build a binary (client or daemon)
Function Execute-Build($additionalBuildTags, $directory) {
    # Generate the build flags
    $buildTags = "autogen"
    if ($Noisy)                     { $verboseParm=" -v" }
    if ($Race)                      { Write-Warning "Using race detector"; $raceParm=" -race"}
    if ($ForceBuildAll)             { $allParm=" -a" }
    if ($NoOpt)                     { $optParm=" -gcflags "+""""+"-N -l"+"""" }
    if ($additionalBuildTags -ne "") { $buildTags += $(" " + $additionalBuildTags) }

    # Do the go build in the appropriate directory
    # Note -linkmode=internal is required to be able to debug on Windows.
    # https://github.com/golang/go/issues/14319#issuecomment-189576638
    Write-Host "INFO: Building..."
    Push-Location $root\cmd\$directory; $global:pushed=$True
    $buildCommand = "go build" + `
                    $raceParm + `
                    $verboseParm + `
                    $allParm + `
                    $optParm + `
                    " -tags """ + $buildTags + """" + `
                    " -ldflags """ + "-linkmode=internal" + """" + `
                    " -o $root\build\"+$directory+".exe"
    Invoke-Expression $buildCommand
    if ($LASTEXITCODE -ne 0) { Throw "Failed to compile" }
    Pop-Location; $global:pushed=$False
}

# Run the unit tests
Function Run-UnitTests() {
    Write-Host "INFO: Running unit tests..."
    $testPath="./..."
    $goListCommand = "go list -e -f '{{if ne .Name """ + '\"github.com/docker/cli\"' + """}}{{.ImportPath}}{{end}}' $testPath"
    $pkgList = $(Invoke-Expression $goListCommand)
    if ($LASTEXITCODE -ne 0) { Throw "go list for unit tests failed" }
    $pkgList = $pkgList | Select-String -Pattern "github.com/docker/cli"
    $pkgList = $pkgList | Select-String -NotMatch "github.com/docker/cli/vendor"
    $pkgList = $pkgList | Select-String -NotMatch "github.com/docker/cli/man"
    $pkgList = $pkgList | Select-String -NotMatch "github.com/docker/cli/e2e"
    $pkgList = $pkgList -replace "`r`n", " "
    $goTestCommand = "go test" + $raceParm + " -cover -ldflags -w -tags """ + "autogen" + """ -a """ + "-test.timeout=10m" + """ $pkgList"
    Invoke-Expression $goTestCommand
    if ($LASTEXITCODE -ne 0) { Throw "Unit tests failed" }
}

# Start of main code.
Try {
    Write-Host -ForegroundColor Cyan "INFO: make.ps1 starting at $(Get-Date)"

    # Get to the root of the repo
    $root = $(Split-Path $MyInvocation.MyCommand.Definition -Parent | Split-Path -Parent)
    Push-Location $root

    # Handle the "-All" shortcut to turn on all things we can handle.
    # Note we expressly only include the items which can run in a container - the validations tests cannot
    # as they require the .git directory which is excluded from the image by .dockerignore
    if ($All) { $Client=$True; $TestUnit=$True }

    # Handle the "-Binary" shortcut to build both client and daemon.
    if ($Binary) { $Client = $True; }

    # Verify git is installed
    if ($(Get-Command git -ErrorAction SilentlyContinue) -eq $nil) { Throw "Git does not appear to be installed" }

    # Verify go is installed
    if ($(Get-Command go -ErrorAction SilentlyContinue) -eq $nil) { Throw "GoLang does not appear to be installed" }

    # Build the binaries
    if ($Client) {
        # Create the build directory if it doesn't exist
        if (-not (Test-Path ".\build")) { New-Item ".\build" -ItemType Directory | Out-Null }

        # Perform the actual build
        Execute-Build "" "docker"
    }

    # Run unit tests
    if ($TestUnit) { Run-UnitTests }

    # Gratuitous ASCII art.
    if ($Client) {
        Write-Host
        Write-Host -ForegroundColor Green " ________   ____  __."
        Write-Host -ForegroundColor Green " \_____  \ `|    `|/ _`|"
        Write-Host -ForegroundColor Green " /   `|   \`|      `<"
        Write-Host -ForegroundColor Green " /    `|    \    `|  \"
        Write-Host -ForegroundColor Green " \_______  /____`|__ \"
        Write-Host -ForegroundColor Green "         \/        \/"
        Write-Host
    }
}
Catch [Exception] {
    Write-Host -ForegroundColor Red ("`nERROR: make.ps1 failed:`n$_")

    # More gratuitous ASCII art.
    Write-Host
    Write-Host -ForegroundColor Red  "___________      .__.__             .___"
    Write-Host -ForegroundColor Red  "\_   _____/____  `|__`|  `|   ____   __`| _/"
    Write-Host -ForegroundColor Red  " `|    __) \__  \ `|  `|  `| _/ __ \ / __ `| "
    Write-Host -ForegroundColor Red  " `|     \   / __ \`|  `|  `|_\  ___// /_/ `| "
    Write-Host -ForegroundColor Red  " \___  /  (____  /__`|____/\___  `>____ `| "
    Write-Host -ForegroundColor Red  "     \/        \/             \/     \/ "
    Write-Host

    Throw $_
}
Finally {
    Pop-Location # As we pushed to the root of the repo as the very first thing
    if ($global:pushed) { Pop-Location }
    Write-Host -ForegroundColor Cyan "INFO: make.ps1 ended at $(Get-Date)"
}
