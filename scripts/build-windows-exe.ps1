param(
    [string]$ImageName = "env-edit-builder",
    [string]$OutputPath = "env-edit.exe",
    [switch]$Console
)

$ErrorActionPreference = "Stop"

$projectRoot = Split-Path -Parent $PSScriptRoot
$resolvedOutputPath = Join-Path $projectRoot $OutputPath
$outputDir = Split-Path -Parent $resolvedOutputPath
$tempOutputDir = Join-Path ([System.IO.Path]::GetTempPath()) ("env-edit-artifact-" + [guid]::NewGuid().ToString("N"))

function Invoke-Docker {
    param(
        [Parameter(ValueFromRemainingArguments = $true)]
        [string[]]$Args
    )

    & docker @Args
    if ($LASTEXITCODE -ne 0) {
        throw "Docker command failed: docker $($Args -join ' ')"
    }
}

if ($outputDir) {
    New-Item -ItemType Directory -Force -Path $outputDir | Out-Null
}

Invoke-Docker info | Out-Null

try {
    New-Item -ItemType Directory -Force -Path $tempOutputDir | Out-Null
    $goLdflags = if ($Console) { "" } else { "-H windowsgui" }
    Invoke-Docker build --build-arg "GO_LDFLAGS=$goLdflags" --target artifact --output "type=local,dest=$tempOutputDir" -t $ImageName $projectRoot
    Copy-Item (Join-Path $tempOutputDir "env-edit.exe") $resolvedOutputPath -Force
    Write-Host "Built $resolvedOutputPath"
}
finally {
    try {
        Remove-Item -Recurse -Force $tempOutputDir -ErrorAction SilentlyContinue
    }
    catch {
    }
}
