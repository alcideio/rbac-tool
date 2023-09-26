#!/bin/bash

# Check if the version and installation directory arguments are provided
if [ $# -ne 2 ]; then
  echo "Usage: $0 <goreleaser_version> <install_directory>"
  exit 1
fi

# Extract the version and installation directory arguments
GORELEASER_VERSION="$1"
INSTALL_DIR="$2"

# Specify the URL of the Goreleaser tar.gz archive for your operating system and architecture
GORELEASER_URL="https://github.com/goreleaser/goreleaser/releases/download/v$GORELEASER_VERSION/goreleaser_$(uname -s)_$(uname -m).tar.gz"

# Ensure the installation directory exists
mkdir -p "$INSTALL_DIR"

# Download Goreleaser tar.gz archive and extract it to the specified directory
curl -L "$GORELEASER_URL" | tar xz -C "$INSTALL_DIR"

# Make the extracted binary executable (assuming it's named "goreleaser")
chmod +x "$INSTALL_DIR/goreleaser"

# Display a success message
echo "Goreleaser $GORELEASER_VERSION has been downloaded and installed to $INSTALL_DIR"
