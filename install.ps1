# Command History Tracker Installation Script for Windows

$ErrorActionPreference = "Stop"

Write-Host "Installing Command History Tracker..." -ForegroundColor Cyan

# Check if Go is installed
try {
    $goVersion = go version
    Write-Host "Found Go: $goVersion" -ForegroundColor Green
} catch {
    Write-Host "Error: Go is not installed. Please install Go 1.24 or later." -ForegroundColor Red
    Write-Host "Visit https://golang.org/dl/ for installation instructions." -ForegroundColor Yellow
    exit 1
}

# Install the tracker command
Write-Host "Installing tracker command..." -ForegroundColor Cyan
go install ./cmd/tracker

# Verify installation
$trackerPath = Get-Command tracker -ErrorAction SilentlyContinue

if ($trackerPath) {
    Write-Host "✓ Command History Tracker installed successfully!" -ForegroundColor Green
    Write-Host ""
    
    # Prompt for automatic setup
    Write-Host "Would you like to set up shell integration now? Y/n: " -ForegroundColor Cyan -NoNewline
    $response = Read-Host
    
    if ($response -eq "" -or $response -eq "y" -or $response -eq "Y") {
        Write-Host ""
        Write-Host "Running automatic setup..." -ForegroundColor Cyan
        & tracker setup
        
        if ($LASTEXITCODE -eq 0) {
            Write-Host ""
            Write-Host "Setup complete! Please restart your shell or run:" -ForegroundColor Green
            Write-Host "  . `$PROFILE" -ForegroundColor Yellow
        } else {
            Write-Host ""
            Write-Host "Setup encountered an issue. You can run 'tracker setup' manually later." -ForegroundColor Yellow
        }
    } else {
        Write-Host ""
        Write-Host "Setup skipped. You can run 'tracker setup' later to configure shell integration." -ForegroundColor Cyan
    }
    
    Write-Host ""
    Write-Host "Quick start:" -ForegroundColor Cyan
    Write-Host "  tracker --help       # Show all commands" -ForegroundColor White
    Write-Host "  tracker setup        # Configure shell integration" -ForegroundColor White
    Write-Host "  tracker browse       # Browse command history" -ForegroundColor White
    Write-Host "  tracker status       # Check installation status" -ForegroundColor White
} else {
    Write-Host "Warning: Installation completed but 'tracker' command not found in PATH." -ForegroundColor Yellow
    Write-Host "Make sure Go bin directory is in your PATH." -ForegroundColor Yellow
    $goPath = go env GOPATH
    Write-Host "Add this to your PATH: $goPath\bin" -ForegroundColor Yellow
}
