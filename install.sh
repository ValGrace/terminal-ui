#!/bin/bash
# Command History Tracker Installation Script

set -e

echo "Installing Command History Tracker..."

# Check if Go is installed
if ! command -v go &> /dev/null; then
    echo "Error: Go is not installed. Please install Go 1.24 or later."
    echo "Visit https://golang.org/dl/ for installation instructions."
    exit 1
fi

# Check Go version
GO_VERSION=$(go version | awk '{print $3}' | sed 's/go//')
REQUIRED_VERSION="1.24"

if [ "$(printf '%s\n' "$REQUIRED_VERSION" "$GO_VERSION" | sort -V | head -n1)" != "$REQUIRED_VERSION" ]; then
    echo "Error: Go version $REQUIRED_VERSION or later is required. Found: $GO_VERSION"
    exit 1
fi

# Install the tracker command
echo "Installing tracker command..."
go install ./cmd/tracker

# Verify installation
if command -v tracker &> /dev/null; then
    echo "✓ Command History Tracker installed successfully!"
    echo ""
    
    # Prompt for automatic setup
    read -p "Would you like to set up shell integration now? (Y/n): " response
    response=${response:-y}
    
    if [[ "$response" =~ ^[Yy]$ ]]; then
        echo ""
        echo "Running automatic setup..."
        tracker setup
        
        if [ $? -eq 0 ]; then
            echo ""
            echo "Setup complete! Please restart your shell or run:"
            
            # Detect shell and provide appropriate command
            if [ -n "$ZSH_VERSION" ]; then
                echo "  source ~/.zshrc"
            elif [ -n "$BASH_VERSION" ]; then
                echo "  source ~/.bashrc"
            else
                echo "  source your shell configuration file"
            fi
        else
            echo ""
            echo "Setup encountered an issue. You can run 'tracker setup' manually later."
        fi
    else
        echo ""
        echo "Setup skipped. You can run 'tracker setup' later to configure shell integration."
    fi
    
    echo ""
    echo "Quick start:"
    echo "  tracker --help       # Show all commands"
    echo "  tracker setup        # Configure shell integration"
    echo "  tracker browse       # Browse command history"
    echo "  tracker status       # Check installation status"
else
    echo "Warning: Installation completed but 'tracker' command not found in PATH."
    echo "Make sure \$GOPATH/bin is in your PATH."
    echo "Add this to your shell profile:"
    echo "  export PATH=\$PATH:\$(go env GOPATH)/bin"
fi
