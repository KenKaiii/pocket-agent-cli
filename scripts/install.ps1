# Pocket CLI Installer for Windows (PowerShell)
# Usage: irm https://raw.githubusercontent.com/KenKaiii/pocket-agent-cli/main/scripts/install.ps1 | iex
# Compatible with Windows PowerShell 5.1 and PowerShell Core 7+

$ErrorActionPreference = "Stop"

$Repo = "KenKaiii/pocket-agent-cli"
$BinaryName = "pocket.exe"
$InstallDir = "$env:LOCALAPPDATA\Pocket"

# --- Helpers ---

function Write-Info { param($msg) Write-Host "[INFO] $msg" -ForegroundColor Green }
function Write-Warn { param($msg) Write-Host "[WARN] $msg" -ForegroundColor Yellow }
function Write-Err  { param($msg) Write-Host "[ERROR] $msg" -ForegroundColor Red; exit 1 }

# Registry-safe PATH helpers (preserves %VAR% references unlike [Environment]::SetEnvironmentVariable)
function Get-Env {
    param([String] $Key)
    $reg = Get-Item -Path 'HKCU:'
    $envKey = $reg.OpenSubKey('Environment')
    if ($null -eq $envKey) { return $null }
    $envKey.GetValue($Key, $null, [Microsoft.Win32.RegistryValueOptions]::DoNotExpandEnvironmentNames)
}

function Write-Env {
    param([String] $Key, [String] $Value)
    $reg = Get-Item -Path 'HKCU:'
    $envKey = $reg.OpenSubKey('Environment', $true)
    if ($Value.Contains('%')) {
        $kind = [Microsoft.Win32.RegistryValueKind]::ExpandString
    } elseif ($null -ne $envKey.GetValue($Key)) {
        $kind = $envKey.GetValueKind($Key)
    } else {
        $kind = [Microsoft.Win32.RegistryValueKind]::String
    }
    $envKey.SetValue($Key, $Value, $kind)
}

# Broadcast WM_SETTINGCHANGE so other programs pick up PATH change without re-login
function Publish-Env {
    if (-not ("Win32.NativeMethods" -as [Type])) {
        Add-Type -Namespace Win32 -Name NativeMethods -MemberDefinition @"
[DllImport("user32.dll", SetLastError = true, CharSet = CharSet.Auto)]
public static extern IntPtr SendMessageTimeout(
    IntPtr hWnd, uint Msg, UIntPtr wParam, string lParam,
    uint fuFlags, uint uTimeout, out UIntPtr lpdwResult);
"@
    }
    $HWND_BROADCAST = [IntPtr] 0xffff
    $WM_SETTINGCHANGE = 0x1a
    $result = [UIntPtr]::Zero
    [Win32.NativeMethods]::SendMessageTimeout($HWND_BROADCAST,
        $WM_SETTINGCHANGE, [UIntPtr]::Zero, "Environment",
        2, 5000, [ref] $result) | Out-Null
}

# --- Platform Detection ---

function Get-Platform {
    $arch = $null
    try {
        # Use reflection to load the correct assembly — direct type access gets the
        # wrong assembly on Windows PowerShell 5.1 and can return incorrect results
        $asm = [System.Reflection.Assembly]::LoadWithPartialName("System.Runtime.InteropServices.RuntimeInformation")
        $type = $asm.GetType("System.Runtime.InteropServices.RuntimeInformation")
        $prop = $type.GetProperty("OSArchitecture")
        $arch = $prop.GetValue($null).ToString()
    } catch {
        # Fallback to environment variables (handles 32-bit PS on 64-bit OS)
        if ($null -ne $env:PROCESSOR_ARCHITEW6432) {
            $arch = $env:PROCESSOR_ARCHITEW6432
        } else {
            $arch = $env:PROCESSOR_ARCHITECTURE
        }
    }

    switch ($arch) {
        { $_ -in "X64", "AMD64" } { return "windows_amd64" }
        default { Write-Err "Unsupported architecture: $arch. Only x64 is currently supported." }
    }
}

# --- GitHub Release ---

function Get-LatestVersion {
    Write-Info "Fetching latest version..."
    try {
        $release = Invoke-RestMethod -Uri "https://api.github.com/repos/$Repo/releases/latest"
        $version = $release.tag_name
        if (-not $version) { Write-Err "Could not determine latest version." }
        Write-Info "Latest version: $version"
        return $version
    } catch {
        Write-Err "Failed to fetch latest release. Check your internet connection."
    }
}

# --- Download & Install ---

function Install-Pocket {
    param($Version, $Platform)

    $downloadUrl = "https://github.com/$Repo/releases/download/$Version/pocket_${Platform}.zip"
    Write-Info "Downloading from: $downloadUrl"

    $tmpDir = Join-Path ([System.IO.Path]::GetTempPath()) ([System.IO.Path]::GetRandomFileName())
    New-Item -ItemType Directory -Path $tmpDir -Force | Out-Null

    try {
        $zipPath = Join-Path $tmpDir "pocket.zip"

        # Silence progress bar — it slows downloads 10-50x on PowerShell 5.1
        $prevProgress = $ProgressPreference
        $ProgressPreference = 'SilentlyContinue'
        try {
            Invoke-WebRequest -Uri $downloadUrl -OutFile $zipPath -UseBasicParsing
        } finally {
            $ProgressPreference = $prevProgress
        }

        Write-Info "Extracting..."
        Expand-Archive -Path $zipPath -DestinationPath $tmpDir -Force

        if (-not (Test-Path $InstallDir)) {
            New-Item -ItemType Directory -Path $InstallDir -Force | Out-Null
        }

        $src = Join-Path $tmpDir $BinaryName
        $dest = Join-Path $InstallDir $BinaryName
        Move-Item -Path $src -Destination $dest -Force

        Write-Info "Installed to: $dest"
    } catch {
        Write-Err "Failed to download or extract. Check if release exists for your platform: $Platform"
    } finally {
        Remove-Item -Path $tmpDir -Recurse -Force -ErrorAction SilentlyContinue
    }
}

# --- PATH Configuration ---

function Set-PathEntry {
    $path = Get-Env 'PATH'

    # Handle null/empty PATH (fresh Windows installs)
    if ($null -eq $path -or $path -eq '') {
        Write-Info "Adding $InstallDir to user PATH..."
        Write-Env -Key 'PATH' -Value $InstallDir
        $env:PATH = "$InstallDir;$env:PATH"
        Publish-Env
        Write-Info "PATH updated"
        return
    }

    # Check for duplicates (case-insensitive split to avoid substring false positives)
    $entries = $path -split ';' | Where-Object { $_ -ne '' }
    if ($entries -contains $InstallDir) {
        Write-Info "PATH already configured"
        return
    }

    Write-Info "Adding $InstallDir to user PATH..."
    $entries += $InstallDir
    Write-Env -Key 'PATH' -Value ($entries -join ';')

    # Update current session
    $env:PATH = "$InstallDir;$env:PATH"

    # Broadcast change to other programs
    Publish-Env
    Write-Info "PATH updated"
}

# --- Main ---

Write-Host ""
Write-Host "+===========================================+" -ForegroundColor Cyan
Write-Host "|        Pocket CLI Installer (Windows)     |" -ForegroundColor Cyan
Write-Host "+===========================================+" -ForegroundColor Cyan
Write-Host ""

# Ensure TLS 1.2 is enabled (PS 5.1 defaults to TLS 1.0/1.1, GitHub requires 1.2)
$originalTls = [Net.ServicePointManager]::SecurityProtocol
[Net.ServicePointManager]::SecurityProtocol = [Net.ServicePointManager]::SecurityProtocol -bor [Net.SecurityProtocolType]::Tls12

try {
    $platform = Get-Platform
    Write-Info "Detected platform: $platform"

    $version = Get-LatestVersion
    Install-Pocket -Version $version -Platform $platform
    Set-PathEntry
} finally {
    [Net.ServicePointManager]::SecurityProtocol = $originalTls
}

Write-Host ""
Write-Host "============================================" -ForegroundColor Green
Write-Host "  Pocket CLI installed successfully!" -ForegroundColor Green
Write-Host "============================================" -ForegroundColor Green
Write-Host ""
Write-Host "Open a new terminal and run:" -ForegroundColor White
Write-Host "  pocket commands" -ForegroundColor Yellow
Write-Host ""
