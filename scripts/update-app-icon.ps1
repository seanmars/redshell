<#
.SYNOPSIS
    Update the RedShell app icon in every location it is referenced.

.DESCRIPTION
    Takes one source image (PNG recommended, ideally 1024x1024 with transparency)
    and regenerates:

        build/appicon.png           - 1024x1024 master (Wails fallback for all platforms)
        build/windows/icon.ico      - Multi-size ICO embedded into the Windows .exe
        frontend/public/favicon.ico - Webview favicon

    Optionally replaces frontend/src/assets/logo.svg when -Svg is supplied.

    macOS: Wails regenerates iconfile.icns from build/appicon.png during
    `wails build -platform darwin`, so updating appicon.png is enough; no
    separate .icns file is maintained in this repo.

.PARAMETER Source
    Path to the new icon image (PNG recommended, 1024x1024).

.PARAMETER Svg
    Optional path to an SVG file that will replace frontend/src/assets/logo.svg.

.PARAMETER SkipFavicon
    Skip regenerating frontend/public/favicon.ico.

.EXAMPLE
    pwsh -File scripts/update-app-icon.ps1 -Source res/logo_1.png

.EXAMPLE
    pwsh -File scripts/update-app-icon.ps1 -Source ./new-icon.png -Svg ./new-logo.svg

.NOTES
    Windows only. Relies on System.Drawing (GDI+) which is supported on
    Windows PowerShell 5.1 and PowerShell 7+ on Windows.
#>

[CmdletBinding()]
param(
    [Parameter(Mandatory = $true)]
    [string]$Source,

    [string]$Svg,

    [switch]$SkipFavicon
)

$ErrorActionPreference = 'Stop'

Add-Type -AssemblyName System.Drawing

$repoRoot    = Split-Path -Parent $PSScriptRoot
$appIconPath = Join-Path $repoRoot 'build/appicon.png'
$winIcoPath  = Join-Path $repoRoot 'build/windows/icon.ico'
$favIcoPath  = Join-Path $repoRoot 'frontend/public/favicon.ico'
$logoSvgPath = Join-Path $repoRoot 'frontend/src/assets/logo.svg'

function Assert-File {
    param([string]$Path)
    if (-not (Test-Path -LiteralPath $Path)) {
        throw "File not found: $Path"
    }
}

function Ensure-Dir {
    param([string]$Path)
    $dir = Split-Path -Parent $Path
    if (-not (Test-Path -LiteralPath $dir)) {
        New-Item -ItemType Directory -Force -Path $dir | Out-Null
    }
}

function New-ResizedBitmap {
    param([System.Drawing.Image]$Image, [int]$Size)

    $bmp = New-Object System.Drawing.Bitmap $Size, $Size
    $g = [System.Drawing.Graphics]::FromImage($bmp)
    try {
        $g.CompositingQuality = [System.Drawing.Drawing2D.CompositingQuality]::HighQuality
        $g.InterpolationMode  = [System.Drawing.Drawing2D.InterpolationMode]::HighQualityBicubic
        $g.SmoothingMode      = [System.Drawing.Drawing2D.SmoothingMode]::HighQuality
        $g.PixelOffsetMode    = [System.Drawing.Drawing2D.PixelOffsetMode]::HighQuality
        $g.Clear([System.Drawing.Color]::Transparent)
        $g.DrawImage($Image, 0, 0, $Size, $Size)
    } finally {
        $g.Dispose()
    }
    return $bmp
}

function Save-ResizedPng {
    param([System.Drawing.Image]$Image, [int]$Size, [string]$OutPath)

    $bmp = New-ResizedBitmap -Image $Image -Size $Size
    try {
        $bmp.Save($OutPath, [System.Drawing.Imaging.ImageFormat]::Png)
    } finally {
        $bmp.Dispose()
    }
}

function Get-PngBytes {
    param([System.Drawing.Image]$Image, [int]$Size)

    $bmp = New-ResizedBitmap -Image $Image -Size $Size
    $ms  = New-Object System.IO.MemoryStream
    try {
        $bmp.Save($ms, [System.Drawing.Imaging.ImageFormat]::Png)
        return , $ms.ToArray()
    } finally {
        $ms.Dispose()
        $bmp.Dispose()
    }
}

# Writes a Vista+ multi-entry ICO where each entry is a PNG-compressed image.
# ICO layout:
#   ICONDIR (6 bytes) + ICONDIRENTRY * N (16 bytes each) + PNG payloads concatenated.
function Write-MultiSizeIco {
    param(
        [System.Drawing.Image]$Image,
        [int[]]$Sizes,
        [string]$OutPath
    )

    $entries = @(
        foreach ($size in $Sizes) {
            [pscustomobject]@{
                Size  = $size
                Bytes = (Get-PngBytes -Image $Image -Size $size)
            }
        }
    )

    $headerSize = 6
    $entrySize  = 16
    $dataOffset = $headerSize + ($entrySize * $entries.Count)

    $fs = [System.IO.File]::Create($OutPath)
    try {
        $bw = New-Object System.IO.BinaryWriter $fs
        # ICONDIR
        $bw.Write([uint16]0)                 # Reserved, must be 0
        $bw.Write([uint16]1)                 # Type: 1 = icon
        $bw.Write([uint16]$entries.Count)    # Number of images

        $offset = $dataOffset
        foreach ($entry in $entries) {
            # Width/height of 0 signals 256 per the ICO spec.
            $dim = if ($entry.Size -ge 256) { 0 } else { [byte]$entry.Size }
            $bw.Write([byte]$dim)                  # width
            $bw.Write([byte]$dim)                  # height
            $bw.Write([byte]0)                     # palette count (0 for true-color)
            $bw.Write([byte]0)                     # reserved
            $bw.Write([uint16]1)                   # color planes
            $bw.Write([uint16]32)                  # bits per pixel
            $bw.Write([uint32]$entry.Bytes.Length) # image data size
            $bw.Write([uint32]$offset)             # image data offset
            $offset += $entry.Bytes.Length
        }
        foreach ($entry in $entries) {
            $bw.Write($entry.Bytes)
        }
        $bw.Flush()
    } finally {
        $fs.Dispose()
    }
}

Assert-File $Source
if ($Svg) { Assert-File $Svg }

$resolvedSource = (Resolve-Path -LiteralPath $Source).Path
Write-Host "Source: $resolvedSource"

$src = [System.Drawing.Image]::FromFile($resolvedSource)
try {
    Write-Host ("  Dimensions: {0}x{1}" -f $src.Width, $src.Height)
    if ($src.Width -lt 256 -or $src.Height -lt 256) {
        Write-Warning "Source is smaller than 256x256; ICO sizes above source dimension will be upscaled."
    }
    if ($src.Width -ne $src.Height) {
        Write-Warning "Source is not square; output will be squished to a square canvas."
    }

    Ensure-Dir $appIconPath
    Save-ResizedPng -Image $src -Size 1024 -OutPath $appIconPath
    Write-Host "  wrote $appIconPath (1024x1024)"

    Ensure-Dir $winIcoPath
    Write-MultiSizeIco -Image $src -Sizes @(16, 24, 32, 48, 64, 128, 256) -OutPath $winIcoPath
    Write-Host "  wrote $winIcoPath (16,24,32,48,64,128,256)"

    if (-not $SkipFavicon) {
        Ensure-Dir $favIcoPath
        Write-MultiSizeIco -Image $src -Sizes @(16, 32, 48, 64) -OutPath $favIcoPath
        Write-Host "  wrote $favIcoPath (16,32,48,64)"
    } else {
        Write-Host "  skipped $favIcoPath (-SkipFavicon)"
    }

    if ($Svg) {
        Ensure-Dir $logoSvgPath
        Copy-Item -LiteralPath $Svg -Destination $logoSvgPath -Force
        Write-Host "  wrote $logoSvgPath (copied from $Svg)"
    } else {
        Write-Host "  skipped $logoSvgPath (no -Svg provided)"
    }
} finally {
    $src.Dispose()
}

Write-Host ""
Write-Host "macOS: Wails regenerates iconfile.icns from build/appicon.png at 'wails build' time."
Write-Host "Done."
