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

function Test-IsAdministrator {
    try {
        $currentIdentity = [Security.Principal.WindowsIdentity]::GetCurrent()
        $principal = New-Object Security.Principal.WindowsPrincipal($currentIdentity)
        return $principal.IsInRole([Security.Principal.WindowsBuiltInRole]::Administrator)
    }
    catch {
        return $false
    }
}

function Install-ToTarget($sourcePath, $targetDir, $scopeName) {
    $targetFile = Join-Path $targetDir "m2apps.exe"
    New-Item -ItemType Directory -Force -Path $targetDir | Out-Null
    Copy-Item $sourcePath $targetFile -Force

    $scope = [EnvironmentVariableTarget]::$scopeName
    $currentPath = [Environment]::GetEnvironmentVariable("Path", $scope)
    if ([string]::IsNullOrWhiteSpace($currentPath)) {
        $currentPath = ""
    }

    if ($currentPath -notlike "*$targetDir*") {
        $updatedPath = if ($currentPath -eq "") { $targetDir } else { "$currentPath;$targetDir" }
        [Environment]::SetEnvironmentVariable("Path", $updatedPath, $scope)
    }

    return $targetFile
}

try {
    $archRaw = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString()
    switch ($archRaw) {
        "X64" { $arch = "amd64" }
        default { Fail "Unsupported architecture: $archRaw" }
    }

    $assetName = "m2apps-windows-$arch.zip"
    $apiUrl = "https://api.github.com/repos/$RepoOwner/$RepoName/releases/latest"
    $machineTargetDir = "C:\Program Files\M2Code"
    $userTargetDir = Join-Path $env:LOCALAPPDATA "M2Code\bin"
    $tempZip = Join-Path $env:TEMP "m2apps-windows.zip"
    $extractDir = Join-Path $env:TEMP "m2apps-extract"
    $tempFile = Join-Path $extractDir "m2apps.exe"
    $installedPath = $null

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

    $isAdmin = Test-IsAdministrator
    if ($isAdmin) {
        try {
            Info "Administrator session detected. Installing to $machineTargetDir (machine scope)..."
            $installedPath = Install-ToTarget $tempFile $machineTargetDir "Machine"
            Info "Machine-level installation completed."
        }
        catch {
            Info "Machine-level install failed: $($_.Exception.Message)"
            Info "Falling back to user-level install at $userTargetDir..."
            $installedPath = Install-ToTarget $tempFile $userTargetDir "User"
            Info "User-level installation completed."
        }
    }
    else {
        Info "Non-administrator session detected. Installing to $userTargetDir (user scope)..."
        $installedPath = Install-ToTarget $tempFile $userTargetDir "User"
        Info "User-level installation completed."
    }

    if (Test-Path $tempZip) {
        Remove-Item $tempZip -Force
    }
    if (Test-Path $extractDir) {
        Remove-Item $extractDir -Recurse -Force
    }

    if (-not $installedPath) {
        Fail "Installation failed: binary path is not set"
    }

    & $installedPath --version | Out-Null
    Info "M2Apps installed successfully at $installedPath"
}
catch {
    Fail $_.Exception.Message
}
