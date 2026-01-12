package backup

import (
	"context"
	"fmt"
	"os"
	"syscall"

	"github.com/stacksnap/stacksnap/internal/docker"
)


type PreflightWarning struct {
	Severity string
	Message string
	Fix   string
}


type PreflightResult struct {
	Warnings  []PreflightWarning
	CanProceed bool
}


func PreflightChecks(client *docker.Client, opts StackBackupOptions) *PreflightResult {
	result := &PreflightResult{
		Warnings:  []PreflightWarning{},
		CanProceed: true,
	}


	if err := client.Ping(); err != nil {
		result.Warnings = append(result.Warnings, PreflightWarning{
			Severity: "error",
			Message: "Docker socket not accessible: " + err.Error(),
			Fix:   "Ensure Docker is running and you have permission to access /var/run/docker.sock",
		})
		result.CanProceed = false
		return result
	}


	estimatedSize := estimateBackupSize(client, opts)
	requiredSpace := estimatedSize * 2

	var stat syscall.Statfs_t
	if err := syscall.Statfs(os.TempDir(), &stat); err == nil {
		availableSpace := int64(stat.Bavail * uint64(stat.Bsize))

		if availableSpace < requiredSpace {
			result.Warnings = append(result.Warnings, PreflightWarning{
				Severity: "warning",
				Message: fmt.Sprintf("Low disk space: %s available, backup may need %s",
					humanizeBytes(availableSpace),
					humanizeBytes(requiredSpace)),
				Fix: "Free up disk space or use a different temp directory",
			})
		} else if availableSpace < requiredSpace*2 {
			result.Warnings = append(result.Warnings, PreflightWarning{
				Severity: "info",
				Message: fmt.Sprintf("Disk space is adequate but tight: %s available",
					humanizeBytes(availableSpace)),
				Fix: "Consider freeing up more space for safety",
			})
		}
	}


	if opts.Directory != "" {


		if _, err := os.Stat(opts.Directory); os.IsNotExist(err) {
			result.Warnings = append(result.Warnings, PreflightWarning{
				Severity: "error",
				Message: "Stack directory not found: " + opts.Directory,
				Fix:   "Verify the directory path is correct",
			})
			result.CanProceed = false
		}
	}


	if opts.StorageProvider != nil {
		ctx := opts.Context
		if ctx == nil {
			ctx = context.Background()
		}


		if _, err := opts.StorageProvider.List(ctx, ""); err != nil {
			result.Warnings = append(result.Warnings, PreflightWarning{
				Severity: "warning",
				Message: "S3 storage not reachable: " + err.Error(),
				Fix:   "Check your AWS credentials and network connectivity",
			})

		}
	}


	if opts.EncryptionKey != nil {
		if len(opts.EncryptionKey) != 32 {
			result.Warnings = append(result.Warnings, PreflightWarning{
				Severity: "error",
				Message: fmt.Sprintf("Invalid encryption key length: %d bytes (expected 32)", len(opts.EncryptionKey)),
				Fix:   "Use a 32-byte (256-bit) encryption key",
			})
			result.CanProceed = false
		}
	}

	return result
}


func estimateBackupSize(client *docker.Client, opts StackBackupOptions) int64 {



	return 10 * 1024 * 1024 * 1024
}


func humanizeBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.1f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}
