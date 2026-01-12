# StackSnap

StackSnap is a precision backup engine for Docker Compose environments. It solves the "trust gap" in self-hosting by providing a verifiable, encrypted, and automated recovery workflow that just works.

No more "hoping" your cron job ran correctly. No more manual database dumps. StackSnap handles the orchestration, so you can focus on your code.

## Core Pillars

- **Synchronized Orchestration**: We coordinate database-native exports (Postgres, MariaDB/MySQL, Redis) with filesystem snapshots to ensure a consistent point-in-time recovery.
- **Zero-Trust Encryption**: Your data is encrypted locally with AES-256-GCM. Your storage provider (S3/Minio) never sees a single byte of raw data.
- **Streamed Visibility**: Real-time log streaming via SSE gives you absolute certainty during long-running operations.
- **Verification First**: Every backup can be automatically verified. If it can't be decompressed or decrypted, it's not a backup—it's just a file.

## Getting Started

### The Quick Way
Download the [compiled package](StackSnap.pkg) and install it on your Mac.

### The Developer Way
```bash
git clone https://github.com/Nesbesss/StackSnap.git
cd stacksnap
go build -o stacksnap ./cmd/stacksnap
./stacksnap
```
Visit `http://localhost:8080` to start the onboarding.

---

## Detailed FAQ

### The Practical Stuff

**Does this scale to 100GB+ volumes?**
Yes. We stream the archive directly to the storage provider where possible, avoiding massive memory buffers. However, your local disk needs enough temp space for the initial database dump file before encryption.

**What happens if my S3 credentials expire mid-backup?**
The operation fails gracefully. StackSnap detects the network/auth error, broadcasts a failure to the UI, and cleans up the local staging artifacts. We don't leave "ghost files" on your disk.

### The Internal "Hard" Questions

**How do you handle 'Snapshot Isolation' during a hot dump?**
We don't just 'copy files'. For Postgres, we use `pg_dump --single-transaction`, which forces the database to provide a snapshot-isolated view of the data. For volumes, we use an 'Atomic Copy' pattern. While it's not a block-level filesystem snapshot (like ZFS/LVM), it minimizes the window of inconsistency by pausing the application runtime (if Safe Mode is enabled) during the archive phase.

**Go is garbage-collected. Does the encryption process cause GC spikes on large archives?**
We use a streaming buffer approach. We read from the source tarball in 32KB chunks, run them through the AES-256-GCM cipher, and write the result immediately. This keeps the memory footprint (RSS) stable regardless of whether you're backing up 10MB or 10GB.

**What if the StackSnap process itself crashes during a restore?**
This is the most critical failure mode. During restore, we capture the state of your existing containers. If a crash occurs during extraction, your containers might be stopped. However, because we don't 'delete' your old data until the new data is extractable, you can always manually restart your old stack using the existing Docker volumes. StackSnap acts as an orchestrator, not a black-box storage layer.

**How do you prevent 'Version Drift' if I restore an old backup to a newer version of Docker Compose?**
Each backup includes the exact `docker-compose.yml` that was active at the time of the backup. When you restore, we use that specific manifest to recreate the services. This ensures that even if you've changed your local files in the meantime, the restored environment matches the data exactly.

## Telemetry & Privacy

StackSnap includes anonymized telemetry via PostHog to help us understand usage patterns and catch bugs. 
- **What we track**: Application startup, success/failure of backup/restore operations, and UI interaction counts.
- **What we NEVER track**: Backup keys, S3 credentials, filenames, or any sensitive data from your containers.
- **Goal**: This data is used solely to improve the software and monitor its health in diverse environments.

---
Built by engineers, for engineers. Distributed under MIT.
