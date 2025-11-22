# Build script for Command History Tracker
# Supports cross-platform builds for Windows, macOS, and Linux

param(
    [string]$Version = "0.1.0",
    [string]$GitCommit = "",
    [string]$BuildDate = "",
    [switch]$All,
    [switch]$Windows,
    [switch]$Linux,
    [switch]$MacOS,
    [switch]$Clean
)

# Set build date if not provided
if ($BuildDate -eq "") {
    $BuildDate = Get-Date -Format "yyyy-MM-ddTHH:mm:ssZ"
}

# Get git commit if not provided
if ($GitCommit -eq "") {
    try {
        $GitCommit = git rev-parse --short HEAD 2>$null
        if ($LASTEXITCODE -ne 0) {
            $GitCommit = "dev"
        }
    } catch {
        $GitCommit = "dev"
    }
}

# Build flags
$ldflags = "-X command-history-tracker/internal/version.Version=$Version " +
           "-X command-history-tracker/internal/version.GitCommit=$GitCommit " +
           "-X command-history-tracker/internal/version.BuildDate=$BuildDate"

# Output directory
$outDir = "dist"

# Clean build directory
if ($Clean) {
    Write-Host "Cleaning build directory..." -ForegroundColor Yellow
    if (Test-Path $outDir) {
        Remove-Item -Recurse -Force $outDir
    }
    Write-Host "✓ Clean complete" -ForegroundColor Green
    exit 0
}

# Create output directory
if (!(Test-Path $outDir)) {
    New-Item -ItemType Directory -Path $outDir | Out-Null
}

Write-Host "Building Command History Tracker v$Version" -ForegroundColor Cyan
Write-Host "Git Commit: $GitCommit" -ForegroundColor Gray
Write-Host "Build Date: $BuildDate" -ForegroundColor Gray
Write-Host ""

# Build function
function Build-Binary {
    param(
        [string]$OS,
        [string]$Arch,
        [string]$Output
    )
    
    Write-Host "Building for $OS/$Arch..." -ForegroundColor Yellow
    
    $env:GOOS = $OS
    $env:GOARCH = $Arch
    $env:CGO_ENABLED = "1"
    
    $outputPath = Join-Path $outDir $Output
    
    go build -ldflags $ldflags -o $outputPath ./cmd/tracker
    
    if ($LASTEXITCODE -eq 0) {
        $size = (Get-Item $outputPath).Length / 1MB
        Write-Host "✓ Built $Output ($([math]::Round($size, 2)) MB)" -ForegroundColor Green
    } else {
        Write-Host "✗ Failed to build $Output" -ForegroundColor Red
        exit 1
    }
}

# Build for specified platforms
if ($All -or (!$Windows -and !$Linux -and !$MacOS)) {
    # Build all platforms by default
    Build-Binary -OS "windows" -Arch "amd64" -Output "tracker-windows-amd64.exe"
    Build-Binary -OS "windows" -Arch "arm64" -Output "tracker-windows-arm64.exe"
    Build-Binary -OS "linux" -Arch "amd64" -Output "tracker-linux-amd64"
    Build-Binary -OS "linux" -Arch "arm64" -Output "tracker-linux-arm64"
    Build-Binary -OS "darwin" -Arch "amd64" -Output "tracker-darwin-amd64"
    Build-Binary -OS "darwin" -Arch "arm64" -Output "tracker-darwin-arm64"
} else {
    if ($Windows) {
        Build-Binary -OS "windows" -Arch "amd64" -Output "tracker-windows-amd64.exe"
        Build-Binary -OS "windows" -Arch "arm64" -Output "tracker-windows-arm64.exe"
    }
    
    if ($Linux) {
        Build-Binary -OS "linux" -Arch "amd64" -Output "tracker-linux-amd64"
        Build-Binary -OS "linux" -Arch "arm64" -Output "tracker-linux-arm64"
    }
    
    if ($MacOS) {
        Build-Binary -OS "darwin" -Arch "amd64" -Output "tracker-darwin-amd64"
        Build-Binary -OS "darwin" -Arch "arm64" -Output "tracker-darwin-arm64"
    }
}

Write-Host ""
Write-Host "Build complete! Binaries are in the '$outDir' directory." -ForegroundColor Green
