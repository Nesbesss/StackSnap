# StackSnap

StackSnap is a lightweight, professional-grade backup and recovery appliance designed specifically for Docker Compose environments. It bridges the gap between manual `tar` commands and complex enterprise backup suites by providing a streamlined, verifiable workflow for containerized applications.

### Why StackSnap?

Self-hosting complex stacks (Nextcloud, Ghost, Home Assistant) often leads to a "backup gap"—you have the files, but you're not sure if the database dump is consistent or if the restore actually works. StackSnap automates the heavy lifting: it identifies your databases, performs safe hot-dumps, snapshots volumes, and encrypts everything before it leaves your host.

## Key Capabilities

- **Atomic-Style Backups**: Orchestrates database dumps (Postgres, MySQL, Redis) and volume snapshots in a single, coherent archive.
- **Verification Engine**: Locally validates archives to ensure they aren't corrupted and can actually be extracted.
- **Hybrid Storage**: Push backups to local storage for speed, or S3-compatible providers (AWS, Backblaze, Minio) for off-site redundancy.
- **Zero-Knowledge Architecture**: All encryption (AES-256) happens locally. Your storage provider never sees your raw data or your keys.
- **Real-Time Instrumentation**: Stream verbose operation logs directly to your browser via Server-Sent Events (SSE).

## Getting Started

### Installation

Download the [latest release package](StackSnap.pkg) for macOS or build from source:

```bash
git clone https://github.com/Nesbesss/StackSnap.git
cd stacksnap
go build -o stacksnap ./cmd/stacksnap
```

### Quick Setup

1. Launch the server: `./stacksnap`
2. Access the dashboard at `http://localhost:8080`.
3. Follow the onboarding flow to set up your primary storage provider.
4. Your running Docker Compose projects will be automatically detected.

---

## Frequently Asked Questions

### The Basics

**How does this differ from just running rsync on /var/lib/docker/volumes?**
`rsync` on a live volume is dangerous. If a database is writing to a file while `rsync` reads it, you end up with a "torn page"—a corrupted backup that looks fine until you try to restore it. StackSnap uses `docker exec` to trigger native database dump tools (like `pg_dump`) to ensure internal consistency before backing up the volume.

**Does it support Docker Swarm?**
Currently, we focus exclusively on Docker Compose/Docker Desktop environments. Swarm support is on our long-term roadmap.

### The Hard Questions

**How do you guarantee database consistency without pausing the containers?**
We use the "Hot Dump" pattern. We initiate a database-level snapshot (e.g., `pg_dump` with `--single-transaction`). This allows the database to remain online for users while we extract a point-in-time consistent state. For non-database volumes, we recommend the "Safe Mode" option in the UI, which briefly pauses the app containers while the volume is archived to prevent file-system-level inconsistencies.

**Is zero-knowledge encryption useful if the encryption key lives on the same server as the backups?**
If you store your backups and your keys on the same physical disk, no. However, StackSnap is designed to push backups to *remote* S3 storage. By keeping the encryption key local (or in your head) and only shipping encrypted blobs to the cloud, you protect yourself against a compromise of your cloud provider. Even if an attacker gains full access to your S3 bucket, they cannot read your data.

**What happens if a backup fails halfway through?**
StackSnap follows an "all-or-nothing" principle. We write to a temporary staging area. Only after the database dump, volume capture, and encryption are successfully completed do we "promote" the backup to your storage provider. If an operation fails, we automatically clean up the partial artifacts to prevent disk bloat.

**Can I restore a backup to a different server?**
Yes. Since StackSnap archives the `docker-compose.yml` along with the data, you can install StackSnap on a fresh machine, connect your S3 bucket, and hit "Restore". It will pull the images, recreate the volumes, and bring the stack back exactly as it was.

---

## Technical Specs

- **Backend**: Go 1.21+ (Docker SDK, AWS SDK)
- **Frontend**: React 18, TypeScript, Tailwind CSS
- **Encryption**: AES-256-GCM
- **Communication**: REST API + SSE for real-time logs

## Safety & Disclaimer

StackSnap is currently in Beta. While we use it to protect our own infrastructure, you should always perform "Fire Drills"—regularly test your restore process to ensure your configuration and keys are correct. The authors are not responsible for data loss.

---
Licensed under MIT.
