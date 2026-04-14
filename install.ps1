param(
    [string]$RepoOwner = "marijmokoginta",
    [string]$RepoName = "m2apps"
)

$ErrorActionPreference = "Stop"

function Fail($message) {
    Write-Host "[ERROR] $message" -ForegroundColor Red
    exit 1
}

function Info($message) {
    Write-Host "[INFO] $message" -ForegroundColor Cyan
}

try {
    $archRaw = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    switch ($archRaw) {
        "X64" { $arch = "amd64" }
        default { Fail "Unsupported architecture: $archRaw" }
    }

    $assetName = "m2apps-windows-$arch.zip"
    $apiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
    $targetDir = "C:\Program Files\M2Code"
    $targetFile = Join-Path $targetDir "m2apps.exe"
    $tempZip = Join-Path $env:TEMP "m2apps-windows.zip"
    $extractDir = Join-Path $env:TEMP "m2apps-extract"
    $tempFile = Join-Path $extractDir "m2apps.exe"

    Info "Repository: $RepoOwner/$RepoName"
    Info "Detected target asset: $assetName"
    Info "Fetching latest release metadata..."

    $release = Invoke-RestMethod -Uri $apiUrl -Method Get
    $asset = $release.assets | Where-Object { $_.name -eq $assetName } | Select-Object -First 1
    if (-not $asset) {
        Fail "Release asset not found for $assetName"
    }

    Info "Downloading release archive..."
    Invoke-WebRequest -Uri $asset.browser_download_url -OutFile $tempZip

    if (Test-Path $extractDir) {
        Remove-Item $extractDir -Recurse -Force
    }

    Info "Extracting release archive..."
    Expand-Archive -Path $tempZip -DestinationPath $extractDir -Force

    if (-not (Test-Path $tempFile)) {
        Fail "m2apps.exe not found after archive extraction"
    }

    Info "Installing to $targetFile..."
    New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
    Copy-Item $tempFile $targetFile -Force

    $currentPath = [Environment]::GetEnvironmentVariable("Path", [EnvironmentVariableTarget]::Machine)
    if ($currentPath -notlike "*$targetDir*") {
        [Environment]::SetEnvironmentVariable(
            "Path",
            "$currentPath;$targetDir",
            [EnvironmentVariableTarget]::Machine
        )
    }

    if (Test-Path $tempZip) {
        Remove-Item $tempZip -Force
    }
    if (Test-Path $extractDir) {
        Remove-Item $extractDir -Recurse -Force
    }

    & $targetFile --version | Out-Null
    Info "M2Apps installed successfully."
}
catch {
    Fail $_.Exception.Message
}
