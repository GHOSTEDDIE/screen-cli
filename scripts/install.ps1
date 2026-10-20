param([string]$From = "$PSScriptRoot\..\dist", [string]$BinDir = "$env:LOCALAPPDATA\screen-cli\bin")
$ErrorActionPreference = 'Stop'
$arch = [System.Runtime.InteropServices.RuntimeInformation]::OSArchitecture.ToString().ToLower()
if ($arch -eq 'x64') { $arch = 'amd64' }
if ($arch -notin @('amd64', 'arm64')) { throw 'Unsupported architecture' }
$name = "screen-cli_windows_$arch.exe"
$entry = Get-Content (Join-Path $From 'SHA256SUMS') | Where-Object { ($_ -split '\s+')[1] -eq $name }
if (@($entry).Count -ne 1) { throw 'Checksum entry missing' }
$expected = ($entry -split '\s+')[0]
$source = Join-Path $From $name
if ((Get-FileHash $source -Algorithm SHA256).Hash.ToLower() -ne $expected) { throw 'Checksum mismatch' }
New-Item -ItemType Directory -Path $BinDir -Force | Out-Null
$temporary = Join-Path $BinDir ([Guid]::NewGuid().ToString() + '.exe')
try {
  Copy-Item $source $temporary
  Move-Item $temporary (Join-Path $BinDir 'screen-cli.exe') -Force
} finally { if (Test-Path $temporary) { Remove-Item $temporary } }
& (Join-Path $BinDir 'screen-cli.exe') --version
