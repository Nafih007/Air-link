# Build the desktop app (airlink.exe). Requires Wails CLI.
$ErrorActionPreference = "Stop"
Set-Location (Split-Path -Parent $PSScriptRoot)
$ver = "dev"
if ($env:AIRLINK_VERSION) { $ver = $env:AIRLINK_VERSION }
if (-not (Get-Command wails -ErrorAction SilentlyContinue)) {
    Write-Error "wails not found. Install: go install github.com/wailsapp/wails/v2/cmd/wails@v2.17.0"
}
wails build -ldflags "-X main.version=$ver"
Get-Item .\build\bin\airlink.exe | Format-List FullName, Length, LastWriteTime
Write-Host "Done. Optional: .\build\bin\airlink.exe --install-menu"
