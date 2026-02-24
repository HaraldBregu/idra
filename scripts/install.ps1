# Idra installer for Windows
# Usage: irm https://example.com/install.ps1 | iex

$ErrorActionPreference = "Stop"

$Repo = "your-org/idra"
$Binary = "idra.exe"
$InstallDir = "$env:LOCALAPPDATA\Idra\bin"

function Write-Info($msg) { Write-Host "[info]  $msg" -ForegroundColor Blue }
function Write-Err($msg) { Write-Host "[error] $msg" -ForegroundColor Red; exit 1 }

# Detect architecture
function Get-Arch {
    $arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture
    switch ($arch) {
        "X64"   { return "amd64" }
        "Arm64" { return "arm64" }
        default { Write-Err "Unsupported architecture: $arch" }
    }
}

# Get latest release version
function Get-LatestVersion {
    $release = Invoke-RestMethod "https://api.github.com/repos/$Repo/releases/latest"
    if (-not $release.tag_name) { Write-Err "Could not determine latest version" }
    return $release.tag_name
}

function Install-Idra {
    $arch = Get-Arch
    $version = Get-LatestVersion
    Write-Info "Installing Idra $version for windows/$arch"

    $url = "https://github.com/$Repo/releases/download/$version/idra-windows-${arch}.exe"

    # Create install directory
    if (-not (Test-Path $InstallDir)) {
        New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
    }

    $dest = Join-Path $InstallDir $Binary
    Write-Info "Downloading $url..."
    Invoke-WebRequest -Uri $url -OutFile $dest -UseBasicParsing

    # Add to PATH if not already present
    $userPath = [Environment]::GetEnvironmentVariable("Path", "User")
    if ($userPath -notlike "*$InstallDir*") {
        [Environment]::SetEnvironmentVariable("Path", "$userPath;$InstallDir", "User")
        $env:Path = "$env:Path;$InstallDir"
        Write-Info "Added $InstallDir to PATH"
    }

    Write-Info "Installed to $dest"

    # Install and start service
    Write-Info "Installing service..."
    & $dest service install

    Write-Info "Starting service..."
    & $dest service start

    Write-Info "Done! Idra is running."
    Start-Sleep -Seconds 1
    Start-Process "http://127.0.0.1:8080"
}

Install-Idra
