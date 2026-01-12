# StackSnap

> **⚠️ BETA VERSION 1.0**: This software is in early preview. Use with caution. Always verify your backups.

StackSnap is a professional-grade backup appliance for Docker Compose stacks. It provides one-click backups, automated database dumps, and instant verification of your recovery plans.

## Features
- 🚀 **One-Click Backups**: Snapshot your entire Compose project in seconds.
- 🗄️ **Smart DB Dumps**: Native hot backups for Postgres, MySQL, and more.
- 🔐 **Zero-Knowledge Encryption**: AES-256 encryption before your data leaves the host.
- ☁️ **S3 Support**: Ship your backups to AWS, Backblaze, or Minio.
- 🔍 **Auto-Verification**: Automatically test your restore process in a safe sandbox.

## Documentation
Please see [DOCS.md](./DOCS.md) for full setup instructions, compatibility lists, and safety warnings.

## Quick Start
1. Install the [latest package](StackSnap.pkg).
2. Open the dashboard at `http://localhost:8080`.
3. Configure your storage (Local or S3).
4. Start protecting your stacks.

## License
MIT
