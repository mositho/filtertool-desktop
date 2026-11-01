$ErrorActionPreference = 'Stop'

Push-Location -LiteralPath (Split-Path -Parent $PSScriptRoot)
# Use a new directory on every invocation, including local runs outside GitHub Actions.
$temp = Join-Path ([System.IO.Path]::GetTempPath()) ('filtertool-installer-' + [guid]::NewGuid().ToString('N'))
New-Item -ItemType Directory -Path $temp | Out-Null
try {
    if (!(Test-Path 'bin/Filtertool.exe')) { throw 'Build bin/Filtertool.exe first with npm run build (WSL/Linux is fine)' }
    # Pin the portable archive; do not install NSIS or execute package-manager scripts.
    $archive = Join-Path $temp 'nsis-3.13.zip'
    # SourceForge otherwise redirects PowerShell's browser-style UA to an HTML landing page.
    Invoke-WebRequest 'https://downloads.sourceforge.net/project/nsis/NSIS%203/3.13/nsis-3.13.zip' -UserAgent 'curl/8.14.1' -OutFile $archive -UseBasicParsing
    if ((Get-FileHash $archive -Algorithm SHA256).Hash -ne 'ba63dffc4410ee89193e1cb5a41989991bd77c61068da17e3156d136b7b0b3d8') {
        throw 'NSIS archive checksum mismatch'
    }
    Expand-Archive $archive -DestinationPath $temp
    $makensis = Join-Path $temp 'nsis-3.13/makensis.exe'
    if ((& $makensis /VERSION) -ne 'v3.13') { throw 'Unexpected NSIS version' }

    Invoke-WebRequest 'https://go.microsoft.com/fwlink/p/?LinkId=2124703' -OutFile 'build/windows/MicrosoftEdgeWebview2Setup.exe' -UseBasicParsing
    $signature = Get-AuthenticodeSignature 'build/windows/MicrosoftEdgeWebview2Setup.exe'
    if ($signature.Status -ne 'Valid' -or $signature.SignerCertificate.Subject -notmatch '(^|,\s*)O=Microsoft Corporation(,|$)') {
        throw 'WebView2 bootstrapper must have a valid Microsoft Corporation signature'
    }
    $version = (Get-Content package.json | ConvertFrom-Json).version
    & $makensis "/DVERSION=$version" build/windows/installer.nsi
    if ($LASTEXITCODE -ne 0) { throw 'Installer build failed' }
    Get-FileHash bin/Filtertool.exe, bin/Filtertool-Setup.exe -Algorithm SHA256 |
        ForEach-Object { "$($_.Hash.ToLower())  $([IO.Path]::GetFileName($_.Path))" } |
        Set-Content -Encoding ascii bin/SHA256SUMS.txt
} finally {
    Remove-Item -LiteralPath $temp -Recurse -Force
    Pop-Location
}
