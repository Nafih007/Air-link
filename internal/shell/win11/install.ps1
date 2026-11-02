# Registers a sparse package + IExplorerCommand for the Win11 top-level menu.
# Args: -Exe <path-to-airlink.exe>
param(
    [string]$Exe = "",
    [switch]$Uninstall
)

$ErrorActionPreference = "Stop"
$Clsid = "{E7B8C4A1-5D3E-4F2A-9C1B-8A6D4E2F9012}"
$PkgName = "SendToPhone.ContextMenu"
$Here = Split-Path -Parent $MyInvocation.MyCommand.Path
$Work = Join-Path $env:LOCALAPPDATA "SendToPhone\win11menu"

function Uninstall-Menu {
    Get-AppxPackage -Name $PkgName -ErrorAction SilentlyContinue | Remove-AppxPackage -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "HKCU:\Software\Classes\CLSID\$Clsid" -Recurse -Force -ErrorAction SilentlyContinue
    Remove-Item -LiteralPath "HKCU:\Software\SendToPhone" -Recurse -Force -ErrorAction SilentlyContinue
}

if ($Uninstall) {
    Uninstall-Menu
    Write-Host "Removed Win11 Air-Link package (if it was installed)."
    exit 0
}

if (-not $Exe) { throw "-Exe is required unless -Uninstall is set." }
$Exe = (Resolve-Path $Exe).Path
New-Item -ItemType Directory -Force -Path $Work | Out-Null
Copy-Item (Join-Path $Here "ExplorerCommand.cs") (Join-Path $Work "ExplorerCommand.cs") -Force

$csc = Join-Path $env:WINDIR "Microsoft.NET\Framework64\v4.0.30319\csc.exe"
if (-not (Test-Path $csc)) {
    throw "csc.exe not found (need .NET Framework 4.8). Classic menu still works via -install-shell."
}

& $csc /nologo /target:library /optimize /out:"$Work\SendToPhone.Shell.dll" "$Work\ExplorerCommand.cs"
if ($LASTEXITCODE -ne 0) { throw "csc failed" }

New-Item -Path "HKCU:\Software\SendToPhone" -Force | Out-Null
Set-ItemProperty -Path "HKCU:\Software\SendToPhone" -Name "Exe" -Value $Exe

# COM registration (HKCU) so Explorer can create the class.
$cls = "HKCU:\Software\Classes\CLSID\$Clsid"
New-Item -Path $cls -Force | Out-Null
Set-ItemProperty -Path $cls -Name "(default)" -Value "SendToPhone.ExplorerCommand"
New-Item -Path "$cls\InprocServer32" -Force | Out-Null
Set-ItemProperty -Path "$cls\InprocServer32" -Name "(default)" -Value (Join-Path $env:WINDIR "System32\mscoree.dll")
Set-ItemProperty -Path "$cls\InprocServer32" -Name "ThreadingModel" -Value "Both"
Set-ItemProperty -Path "$cls\InprocServer32" -Name "Class" -Value "SendToPhoneCommand"
Set-ItemProperty -Path "$cls\InprocServer32" -Name "Assembly" -Value "SendToPhone.Shell"
Set-ItemProperty -Path "$cls\InprocServer32" -Name "RuntimeVersion" -Value "v4.0.30319"
Set-ItemProperty -Path "$cls\InprocServer32" -Name "CodeBase" -Value "file:///$($Work.Replace('\','/'))/SendToPhone.Shell.dll"

# Do NOT set ExplorerCommandHandler on the classic verb. That CLSID makes
# Explorer ignore command\ unless a signed sparse package is trusted, which
# hid "Send to phone" under Show more options. Compact menu is a later task.
foreach ($kind in @('*', 'Directory')) {
    $vk = "HKCU:\Software\Classes\$kind\shell\SendToPhone"
    if (Test-Path -LiteralPath $vk) {
        Remove-ItemProperty -LiteralPath $vk -Name "ExplorerCommandHandler" -ErrorAction SilentlyContinue
    }
}

# Sparse package (signed with a CurrentUser self-signed cert when possible).
$Assets = Join-Path $Work "Assets"
New-Item -ItemType Directory -Force -Path $Assets | Out-Null
Add-Type -AssemblyName System.Drawing
function Write-Logo($path, $size) {
    $bmp = New-Object System.Drawing.Bitmap $size, $size
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    $g.Clear([System.Drawing.Color]::FromArgb(15, 20, 25))
    $brush = New-Object System.Drawing.SolidBrush ([System.Drawing.Color]::FromArgb(61, 139, 253))
    $g.FillEllipse($brush, 4, 4, $size - 8, $size - 8)
    $bmp.Save($path, [System.Drawing.Imaging.ImageFormat]::Png)
    $g.Dispose(); $bmp.Dispose()
}
Write-Logo (Join-Path $Assets "Square44.png") 44
Write-Logo (Join-Path $Assets "Square150.png") 150

$manifest = @"
<?xml version="1.0" encoding="utf-8"?>
<Package xmlns="http://schemas.microsoft.com/appx/manifest/foundation/windows10"
  xmlns:uap="http://schemas.microsoft.com/appx/manifest/uap/windows10"
  xmlns:uap10="http://schemas.microsoft.com/appx/manifest/uap/windows10/10"
  xmlns:desktop4="http://schemas.microsoft.com/appx/manifest/desktop/windows10/4"
  xmlns:rescap="http://schemas.microsoft.com/appx/manifest/foundation/windows10/restrictedcapabilities"
  xmlns:com="http://schemas.microsoft.com/appx/manifest/com/windows10"
  IgnorableNamespaces="uap uap10 desktop4 rescap com">
  <Identity Name="$PkgName" Publisher="CN=SendToPhone" Version="1.0.0.0" ProcessorArchitecture="neutral" />
  <Properties>
    <DisplayName>Air-Link</DisplayName>
    <PublisherDisplayName>Air-Link</PublisherDisplayName>
    <Logo>Assets\Square150.png</Logo>
    <uap10:AllowExternalContent>true</uap10:AllowExternalContent>
  </Properties>
  <Resources><Resource Language="en-us" /></Resources>
  <Dependencies>
    <TargetDeviceFamily Name="Windows.Desktop" MinVersion="10.0.19041.0" MaxVersionTested="10.0.22621.0" />
  </Dependencies>
  <Applications>
    <Application Id="SendToPhone" Executable="airlink.exe" EntryPoint="Windows.FullTrustApplication">
      <uap:VisualElements DisplayName="Air-Link" Description="Send files between your PC and phone"
        BackgroundColor="#0F1419" Square150x150Logo="Assets\Square150.png" Square44x44Logo="Assets\Square44.png" />
      <Extensions>
        <com:Extension Category="windows.comServer">
          <com:ComServer>
            <com:SurrogateServer DisplayName="Send with Air-Link">
              <com:Class Id="$Clsid" Path="SendToPhone.Shell.dll" ThreadingModel="STA" />
            </com:SurrogateServer>
          </com:ComServer>
        </com:Extension>
        <desktop4:Extension Category="windows.fileExplorerContextMenus">
          <desktop4:FileExplorerContextMenus>
            <desktop4:ItemType Type="*">
              <desktop4:Verb Id="SendToPhone" Clsid="$Clsid" />
            </desktop4:ItemType>
            <desktop4:ItemType Type="Directory">
              <desktop4:Verb Id="SendToPhone" Clsid="$Clsid" />
            </desktop4:ItemType>
          </desktop4:FileExplorerContextMenus>
        </desktop4:Extension>
      </Extensions>
    </Application>
  </Applications>
  <Capabilities>
    <rescap:Capability Name="runFullTrust" />
  </Capabilities>
</Package>
"@
Set-Content -Path (Join-Path $Work "AppxManifest.xml") -Value $manifest -Encoding UTF8

try {
    $cert = Get-ChildItem Cert:\CurrentUser\My | Where-Object { $_.Subject -eq "CN=SendToPhone" } | Select-Object -First 1
    if (-not $cert) {
        $cert = New-SelfSignedCertificate -Type CodeSigningCert -Subject "CN=SendToPhone" -CertStoreLocation Cert:\CurrentUser\My -HashAlgorithm SHA256
    }
    if ($cert) {
        $cer = Join-Path $Work "SendToPhone.cer"
        Export-Certificate -Cert $cert -FilePath $cer | Out-Null
        Import-Certificate -FilePath $cer -CertStoreLocation Cert:\CurrentUser\TrustedPeople | Out-Null
    }
} catch {
    Write-Warning "Could not create or trust a self-signed cert: $_"
}

$link = Join-Path $Work "airlink.exe"
if (Test-Path $link) { Remove-Item $link -Force }
try {
    New-Item -ItemType HardLink -Path $link -Target $Exe | Out-Null
} catch {
    Copy-Item $Exe $link -Force
}

$reg = Join-Path $Work "AppxManifest.xml"
try {
    Add-AppxPackage -Path $reg -Register -ExternalLocation $Work -ForceUpdateFromAnyVersion
    Write-Host "Registered sparse package $PkgName. Restart Explorer if the top-level item is missing."
} catch {
    Write-Warning "Add-AppxPackage failed (compact Win11 menu needs Developer Mode / a signed package). Classic Show more options still works."
}
