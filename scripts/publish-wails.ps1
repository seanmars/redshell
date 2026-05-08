<#
.SYNOPSIS
    Build a publishable release of RedShell via the Wails CLI.

.DESCRIPTION
    Wraps `wails build` with the flags this project uses for distributable
    binaries. By default produces a complete release set ready to upload to
    GitHub / GitLab releases:
      * redshell-<os>-<arch>(.exe)         - portable binary (auto-update target)
      * <ProjectName>-<arch>-installer.exe - NSIS installer (Windows only)
      * checksums.txt                      - sha256sum-compatible SHA-256 file

    All three land in dist/ by default. The portable binary is renamed to
    match the asset name the in-app auto-updater expects
    (must mirror Go's updater.AssetNameFor); the installer keeps its
    wails-generated filename. checksums.txt covers every dist artifact.

    To skip the dist copy pass -OutputDir ''; to skip the installer pass
    -Nsis:$false.

.PARAMETER Platform
    Target platform passed to `wails build -platform`. Defaults to
    windows/amd64. Common values: windows/amd64, windows/arm64,
    windows/386, linux/amd64, darwin/universal.

.PARAMETER Version
    Version string written into wails.json before the build so it lands in
    both the embedded `wails.json` consumed by main.GetAppVersion and the
    Windows VERSIONINFO resource. Falls back to wails.json's
    info.productVersion, then to `git describe`, then to "dev".

.PARAMETER Nsis
    Also produce the NSIS installer (build/bin/<ProjectName>-<arch>-installer.exe).
    Windows targets only. Defaults to $true so the standard publish flow
    yields both portable binary and installer; pass -Nsis:$false to skip.

.PARAMETER Clean
    Remove build/bin before building so stale artifacts cannot be picked
    up by mistake.

.PARAMETER Obfuscated
    Pass -obfuscated to `wails build` (uses garble; must be installed).

.PARAMETER Webview2
    Strategy for the WebView2 runtime on Windows: download | embed |
    browser | error. Default 'download' makes the installer fetch the
    runtime on first launch when missing.

.PARAMETER OutputDir
    Directory to copy the release set into. Defaults to 'dist'. Files are
    renamed to the auto-update-compatible scheme:
        redshell-<os>-<arch>(.exe)            (portable, must match
                                               updater.AssetNameFor in Go)
        <ProjectName>-<arch>-installer.exe    (installer, original wails name)
        checksums.txt                         (sha256sum-compatible)
    Created if missing. Pass an empty string to skip the dist copy.

.PARAMETER SkipFrontendInstall
    Pass -s to `wails build` to skip `pnpm install` before the frontend
    build. Speeds up iteration when node_modules is already in sync.

.PARAMETER CertThumbprint
    SHA1 thumbprint of a code-signing certificate already imported into
    the current user / local machine cert store. Preferred over -CertPath
    because no password ever appears on the command line.

.PARAMETER CertPath
    Path to a .pfx code-signing certificate. Combine with -CertPassword.
    Mutually exclusive with -CertThumbprint.

.PARAMETER CertPassword
    Password for the .pfx supplied via -CertPath.

.PARAMETER TimestampUrl
    RFC 3161 timestamp server (default http://timestamp.digicert.com).
    Required so signatures stay valid after the certificate expires.

.EXAMPLE
    pwsh -File scripts/publish-wails.ps1
    # windows/amd64 portable + NSIS installer + checksums.txt -> dist/

.EXAMPLE
    pwsh -File scripts/publish-wails.ps1 -Version 1.2.0
    # same publish defaults but pin version to 1.2.0 (overrides wails.json)

.EXAMPLE
    pwsh -File scripts/publish-wails.ps1 -CertThumbprint ABCDEF0123...
    # build + Authenticode-sign exe and installer with cert from store

.EXAMPLE
    pwsh -File scripts/publish-wails.ps1 -OutputDir '' -Nsis:$false
    # local-only build into build/bin, no installer, no dist copy

.NOTES
    Requires the wails CLI on PATH (go install github.com/wailsapp/wails/v2/cmd/wails@latest).
    Run from any directory; the script resolves the repo root from its own location.
#>

[CmdletBinding()]
param(
    [ValidateSet(
        'windows/amd64','windows/arm64','windows/386',
        'linux/amd64','linux/arm64',
        'darwin/amd64','darwin/arm64','darwin/universal'
    )]
    [string]$Platform = 'windows/amd64',

    [string]$Version,

    [switch]$Nsis = $true,

    [switch]$Clean,

    [switch]$Obfuscated,

    [ValidateSet('download','embed','browser','error')]
    [string]$Webview2 = 'download',

    [string]$OutputDir = 'dist',

    [switch]$SkipFrontendInstall,

    # Code-signing (Windows only). Either pass -CertThumbprint to use a cert
    # already imported into the user/machine cert store (preferred — keeps the
    # PFX password off the command line), or -CertPath plus -CertPassword to
    # sign with a PFX file directly. Both modes apply an RFC 3161 timestamp
    # so signatures stay valid after the cert expires.
    [string]$CertThumbprint,

    [string]$CertPath,

    [string]$CertPassword,

    [string]$TimestampUrl = 'http://timestamp.digicert.com'
)

$ErrorActionPreference = 'Stop'

$repoRoot = Split-Path -Parent $PSScriptRoot
$binDir   = Join-Path $repoRoot 'build/bin'

function Assert-Command {
    param([string]$Name)
    if (-not (Get-Command $Name -ErrorAction SilentlyContinue)) {
        throw "Required command not found on PATH: $Name"
    }
}

function Resolve-Version {
    param([string]$Explicit)
    if ($Explicit) { return $Explicit }

    $wailsJsonPath = Join-Path $repoRoot 'wails.json'
    if (Test-Path -LiteralPath $wailsJsonPath) {
        try {
            $parsed = Get-Content -LiteralPath $wailsJsonPath -Raw | ConvertFrom-Json
            if ($parsed.info -and $parsed.info.productVersion) {
                return [string]$parsed.info.productVersion
            }
        } catch {
            Write-Warning "Failed to parse wails.json: $_"
        }
    }

    if (Get-Command git -ErrorAction SilentlyContinue) {
        $described = & git -C $repoRoot describe --tags --always --dirty 2>$null
        if ($LASTEXITCODE -eq 0 -and $described) {
            return $described.Trim()
        }
    }
    return 'dev'
}

function Split-Platform {
    param([string]$Value)
    $parts = $Value.Split('/')
    if ($parts.Count -ne 2) {
        throw "Platform must be <os>/<arch>, got: $Value"
    }
    return [pscustomobject]@{ OS = $parts[0]; Arch = $parts[1] }
}

# Wails reads info.productVersion from wails.json and substitutes it into
# build/windows/info.json (PE VERSIONINFO) and the NSI installer. Patch the
# field in place so $resolvedVersion lands in both the exe metadata and the
# installer; the original file is restored in the build finally block.
function Set-WailsProductVersion {
    param(
        [Parameter(Mandatory)][string]$WailsJsonPath,
        [Parameter(Mandatory)][string]$Version
    )
    $original = Get-Content -LiteralPath $WailsJsonPath -Raw
    $json = $original | ConvertFrom-Json
    if (-not $json.PSObject.Properties['info']) {
        $json | Add-Member -MemberType NoteProperty -Name info -Value ([pscustomobject]@{})
    }
    if ($json.info.PSObject.Properties['productVersion']) {
        $json.info.productVersion = $Version
    } else {
        $json.info | Add-Member -MemberType NoteProperty -Name productVersion -Value $Version
    }
    ($json | ConvertTo-Json -Depth 20) | Set-Content -LiteralPath $WailsJsonPath -Encoding utf8NoBOM
    return $original
}

# signtool ships with the Windows 10 SDK; it is not on PATH by default.
# Search the SDK 'bin' tree for the newest x64 build. Returns $null when
# the SDK is not installed so the caller can degrade gracefully.
function Get-SignToolPath {
    $cmd = Get-Command signtool.exe -ErrorAction SilentlyContinue
    if ($cmd) { return $cmd.Source }

    $sdkRoots = @(
        "${env:ProgramFiles(x86)}\Windows Kits\10\bin",
        "${env:ProgramFiles}\Windows Kits\10\bin"
    ) | Where-Object { Test-Path -LiteralPath $_ }

    foreach ($root in $sdkRoots) {
        $candidate = Get-ChildItem -LiteralPath $root -Recurse -Filter 'signtool.exe' -ErrorAction SilentlyContinue |
            Where-Object { $_.Directory.Name -eq 'x64' } |
            Sort-Object -Property FullName -Descending |
            Select-Object -First 1
        if ($candidate) { return $candidate.FullName }
    }
    return $null
}

function Invoke-Sign {
    param(
        [Parameter(Mandatory)][string]$SignTool,
        [Parameter(Mandatory)][string]$FilePath
    )
    $signArgs = @('sign', '/fd', 'sha256', '/td', 'sha256', '/tr', $TimestampUrl)
    if ($CertThumbprint) {
        $signArgs += @('/sha1', $CertThumbprint)
    } else {
        $signArgs += @('/f', $CertPath)
        if ($CertPassword) { $signArgs += @('/p', $CertPassword) }
    }
    $signArgs += $FilePath

    & $SignTool @signArgs
    if ($LASTEXITCODE -ne 0) {
        throw "signtool failed for $FilePath (exit $LASTEXITCODE)"
    }
}

Assert-Command wails

$platformInfo  = Split-Platform $Platform
$resolvedVersion = Resolve-Version -Explicit $Version

$signRequested = [bool]($CertThumbprint -or $CertPath)
if ($signRequested -and $platformInfo.OS -ne 'windows') {
    throw "Code signing parameters are only valid for windows targets (got $Platform)."
}
if ($CertPath -and $CertThumbprint) {
    throw "Pass either -CertThumbprint or -CertPath, not both."
}
if ($CertPath -and -not (Test-Path -LiteralPath $CertPath)) {
    throw "CertPath not found: $CertPath"
}

Write-Host "Repo root : $repoRoot"
Write-Host "Platform  : $Platform"
Write-Host "Version   : $resolvedVersion"
Write-Host "WebView2  : $Webview2"
Write-Host "NSIS      : $([bool]$Nsis)"
Write-Host "Obfuscated: $([bool]$Obfuscated)"
Write-Host "Sign      : $signRequested"
Write-Host "OutputDir : $(if ($OutputDir) { $OutputDir } else { '<none>' })"

if ($Nsis -and $platformInfo.OS -ne 'windows') {
    throw "-Nsis is only supported for windows targets (got $Platform)."
}

if ($Clean -and (Test-Path -LiteralPath $binDir)) {
    Write-Host "Cleaning $binDir"
    Remove-Item -LiteralPath $binDir -Recurse -Force
}

# Strip symbol table and DWARF info to slim the binary and reduce AV
# heuristic surface. Version is read at runtime from the embedded wails.json
# (Set-WailsProductVersion below patches it in place before build), so no
# `-X main.version` is required.
#
# The portable and installer variants need DIFFERENT BuildKind values baked
# in (see internal/updater/buildkind.go). The portable build uses the
# default `BuildKind = "portable"`. The installer build sets
# `BuildKind = "installer"` via -X so the in-app updater takes the elevated
# silent-install pathway instead of the rename-trick swap (which would
# fail in Program Files).
#
# Note: the linker -X flag takes `importpath.Name=value` with NO inner
# quoting — single quotes are bash escaping that PowerShell would pass
# through literally and silently break the importpath parse, leaving the
# binary at the default BuildKind value.
$portableLdflags  = "-s -w"
$installerLdflags = "-s -w -X redshell/internal/updater.BuildKind=installer"

function Invoke-WailsBuild {
    param(
        [Parameter(Mandatory)][string]$Ldflags,
        [Parameter(Mandatory)][bool]$IncludeNsis,
        [Parameter(Mandatory)][string]$Label
    )
    $args = @(
        'build',
        '-platform', $Platform,
        '-ldflags',  $Ldflags,
        '-webview2', $Webview2,
        '-trimpath'
    )
    if ($IncludeNsis)         { $args += '-nsis' }
    if ($Obfuscated)          { $args += '-obfuscated' }
    if ($SkipFrontendInstall) { $args += '-s' }

    Write-Host ""
    Write-Host "[$Label] wails $($args -join ' ')"
    Write-Host ""

    & wails @args
    if ($LASTEXITCODE -ne 0) {
        throw "wails build [$Label] failed with exit code $LASTEXITCODE"
    }
}

$wailsJsonPath = Join-Path $repoRoot 'wails.json'
$originalWailsJson = Set-WailsProductVersion -WailsJsonPath $wailsJsonPath -Version $resolvedVersion

# Clean once up front so the per-pass invocations don't need -clean (which
# would wipe the installer artifact between the installer and portable passes).
if ($Clean -and (Test-Path -LiteralPath $binDir)) {
    Write-Host "Cleaning $binDir (already done above; safety re-check)"
}

Push-Location $repoRoot
try {
    if ($Nsis -and $platformInfo.OS -eq 'windows') {
        # Single pass: installer-only build.
        #
        # Releases ship the NSIS installer as the sole Windows artifact.
        # Existing portable users can no longer in-app update; they need to
        # download the installer manually once to migrate. The provider
        # accepts the missing portable asset (returns Release with empty
        # AssetURL) so installer clients still see the release; the portable
        # install path emits a clear error pointing the user to manual
        # installer download.
        #
        # The redshell.exe wails leaves in build/bin has BuildKind=installer
        # baked in and is wrapped inside the NSIS installer. We deliberately
        # do NOT publish that loose .exe — distributing it as if it were
        # portable would mislead users into thinking they can run it without
        # the installer, and its in-app updater would still try the
        # elevated-install pathway from wherever they put it.
        Invoke-WailsBuild -Ldflags $installerLdflags -IncludeNsis $true -Label 'installer'
    } else {
        # Non-Windows or explicitly -Nsis:$false. Linux/Mac don't have the
        # installer pathway anyway (rename_other.go is a stub), and the
        # -Nsis:$false flag is mainly an escape hatch for local testing.
        Invoke-WailsBuild -Ldflags $portableLdflags -IncludeNsis $false -Label 'portable'
    }
} finally {
    Set-Content -LiteralPath $wailsJsonPath -Value $originalWailsJson -Encoding utf8NoBOM -NoNewline
    Pop-Location
}

# Resolve produced artifacts. Wails names the portable as
# <outputfilename>(.exe) and the NSIS installer as
# <ProjectName>-<arch>-installer.exe — note <ProjectName> is the
# wails.json "name" field (case-preserved, e.g. "RedShell"), so we
# discover the installer via glob rather than guessing the case.
$exeSuffix = if ($platformInfo.OS -eq 'windows') { '.exe' } else { '' }
$exeName   = "redshell$exeSuffix"
$exePath   = Join-Path $binDir $exeName

$installerExe = $null
if ($Nsis -and $platformInfo.OS -eq 'windows') {
    $found = Get-ChildItem -LiteralPath $binDir -Filter "*-$($platformInfo.Arch)-installer.exe" -ErrorAction SilentlyContinue |
        Sort-Object -Property LastWriteTime -Descending |
        Select-Object -First 1
    if ($found) { $installerExe = $found.FullName }
}

$artifacts = @()
if ($Nsis -and $platformInfo.OS -eq 'windows') {
    # Installer-only release: ship only the NSIS installer + checksums.
    # The loose redshell.exe in build/bin is the BuildKind=installer binary
    # already wrapped inside the installer; publishing it as portable would
    # confuse users and break their in-app updater.
    if (-not $installerExe) {
        throw "Expected NSIS installer artifact not found in $binDir; cannot publish installer-only release."
    }
    $artifacts += $installerExe
} else {
    if (Test-Path -LiteralPath $exePath) { $artifacts += $exePath }
}

# Sign before any OutputDir copy so downstream artifacts inherit the signature.
if ($signRequested) {
    $signTool = Get-SignToolPath
    if (-not $signTool) {
        throw "signtool.exe not found. Install the Windows 10/11 SDK or place signtool.exe on PATH."
    }
    Write-Host ""
    Write-Host "Signing with: $signTool"
    foreach ($a in $artifacts) {
        Write-Host "  signing $a"
        Invoke-Sign -SignTool $signTool -FilePath $a
    }
}

Write-Host ""
Write-Host "Build complete. Artifacts:"
foreach ($a in $artifacts) {
    $size = (Get-Item -LiteralPath $a).Length
    $hash = (Get-FileHash -LiteralPath $a -Algorithm SHA256).Hash.ToLowerInvariant()
    Write-Host ("  {0} ({1:N0} bytes)" -f $a, $size)
    Write-Host ("    sha256: {0}" -f $hash)
}

if ($OutputDir) {
    if (-not (Test-Path -LiteralPath $OutputDir)) {
        New-Item -ItemType Directory -Force -Path $OutputDir | Out-Null
    }
    $resolvedOut = (Resolve-Path -LiteralPath $OutputDir).Path

    Write-Host ""
    Write-Host "Staging release set in $resolvedOut"

    $copiedFiles = @()
    foreach ($a in $artifacts) {
        $sourceName = Split-Path -Leaf $a
        $destName = if ($sourceName -ieq "redshell$exeSuffix") {
            # Portable: rename to the asset name the in-app auto-updater
            # expects. Must mirror Go's updater.AssetNameFor — change one and
            # the other will fail asset lookup.
            "redshell-{0}-{1}{2}" -f $platformInfo.OS, $platformInfo.Arch, $exeSuffix
        } else {
            # Installer: keep wails' original filename (case-preserved).
            $sourceName
        }
        $dest = Join-Path $resolvedOut $destName
        Copy-Item -LiteralPath $a -Destination $dest -Force
        $copiedFiles += $dest
        Write-Host "  copied  -> $dest"
    }

    # checksums.txt: sha256sum-compatible (`<hex>  <filename>`, LF endings,
    # UTF-8 no BOM) so power users can verify with `sha256sum -c`, and the
    # in-app updater's ParseChecksums can locate the portable asset's hash.
    if ($copiedFiles.Count -gt 0) {
        $checksumsPath = Join-Path $resolvedOut 'checksums.txt'
        $lines = foreach ($f in $copiedFiles) {
            $hash = (Get-FileHash -LiteralPath $f -Algorithm SHA256).Hash.ToLowerInvariant()
            $name = Split-Path -Leaf $f
            "{0}  {1}" -f $hash, $name
        }
        $body = ($lines -join "`n") + "`n"
        [System.IO.File]::WriteAllText(
            $checksumsPath,
            $body,
            [System.Text.UTF8Encoding]::new($false)
        )
        Write-Host "  wrote   -> $checksumsPath"
    }
}

Write-Host ""
Write-Host "Done."
