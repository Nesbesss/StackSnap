# StackSnap Documentation (v1.0 BETA)

## ⚠️ Important Disclaimer
StackSnap is currently in **BETA**. 
- Use at your own risk.
- The authors are not liable for any data loss.
- **Always verify your backups** by performing a test restore before relying on them for production data.

## Compatibility List
StackSnap has been tested and is known to work with the following:

### Databases (Live Dumps)
- **PostgreSQL**: Tested with versions 12, 13, 14, 15, 16.
- **MySQL / MariaDB**: Tested with standard official images.
- **MongoDB**: Basic support via `mongodump`.
- **Redis**: RDB snapshot capture.

### Common Stacks
- **Nextcloud**: Verified (Postgres + Volumes).
- **Home Assistant**: Verified (Volumes).
- **Ghost**: Verified (MySQL + Volumes).
- **Standard Nginx/Apache**: Verified (Volumes).

*If you find a stack and it works (or doesn't), please report it to our community tracker.*

## Backup Modes
1. **Hot Dump (Default)**: Dumps databases via `docker exec` while they are running. Fast and no downtime.
2. **Pause Apps (Optional)**: Pauses application containers that write to volumes, but keeps the Database running for a clean dump. Best balance of consistency and uptime.
3. **Full Pause (Internal)**: Not recommended for high-uptime apps, but available for maximum consistency.

## License
StackSnap is licensed under the MIT License.
- **No Warranty**: The software is provided "as is", without warranty of any kind.
