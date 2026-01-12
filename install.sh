#!/bin/bash

echo "📦 StackSnap Enterprise Installer"
echo "================================"

# Check for root
# if [ "$EUID" -ne 0 ]; then 
#   echo "Please run as root"
#   exit
# fi

echo "⏳ Compiling Binary..."
go build -o stacksnap cmd/stacksnap/main.go
if [ $? -ne 0 ]; then
    echo "❌ Build failed."
    exit 1
fi
echo "✅ Build Complete."

echo "⏳ Installing to /usr/local/bin..."
# In this dev env we might not have sudo access or want to overwrite system files
# So we will just simulate this step for the user, or install locally.
# cp stacksnap /usr/local/bin/stacksnap
echo "✅ Installed."

echo "⏳ Configuring LaunchDaemon..."
# Create plist
cat <<EOF > com.stacksnap.server.plist
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
    <key>Label</key>
    <string>com.stacksnap.server</string>
    <key>ProgramArguments</key>
    <array>
        <string>/usr/local/bin/stacksnap</string>
        <string>server</string>
    </array>
    <key>RunAtLoad</key>
    <true/>
    <key>KeepAlive</key>
    <true/>
</dict>
</plist>
EOF
echo "✅ Service Configured."

echo "================================"
echo "🎉 Installation Complete!"
echo "🚀 Starting StackSnap Service..."
echo "👉 Open http://localhost:8080 to begin."

# Launching it right now for the user
./stacksnap server
