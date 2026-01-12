#!/bin/bash
set -e

APP_NAME="StackSnap"
VERSION="1.0.0"
IDENTIFIER="io.stacksnap.server"
OUTPUT_PKG="${APP_NAME}.pkg"

# Create a staging directory
STAGING_DIR="build_pkg_staging"
rm -rf "$STAGING_DIR"
mkdir -p "$STAGING_DIR/usr/local/bin"
mkdir -p "$STAGING_DIR/Library/LaunchAgents"

echo "🎨 Building Web UI..."
cd ui && npm run build && cd ..

# Ensure dist exists in cmd/stacksnap/
mkdir -p cmd/stacksnap/dist
cp -r ui/dist/* cmd/stacksnap/dist/

echo "⏳ Building Binary..."
go build -o stacksnap cmd/stacksnap/main.go

echo "📂 Preparing Files..."
# Copy binary
cp stacksnap "$STAGING_DIR/usr/local/bin/"
chmod 755 "$STAGING_DIR/usr/local/bin/stacksnap"

# Copy plist
cp scripts/com.stacksnap.server.plist "$STAGING_DIR/Library/LaunchAgents/"
chmod 644 "$STAGING_DIR/Library/LaunchAgents/com.stacksnap.server.plist"

echo "📦 Building Component Package..."
pkgbuild --root "$STAGING_DIR" \
         --identifier "$IDENTIFIER" \
         --version "$VERSION" \
         --install-location "/" \
         "$OUTPUT_PKG"

echo "🧹 Cleanup..."
rm -rf "$STAGING_DIR"

echo "✅ Package Created: $OUTPUT_PKG"
echo "👉 You can now install it with: sudo installer -pkg $OUTPUT_PKG -target /"
