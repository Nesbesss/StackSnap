package backup

import (
	"errors"
	"fmt"
)


var (
	ErrDiskFull                = errors.New("insufficient disk space for backup")
	ErrNetworkTimeout          = errors.New("network timeout during upload")
	ErrPermissionDenied        = errors.New("permission denied accessing volume")
	ErrCorruptedTar            = errors.New("tar archive corrupted during creation")
	ErrS3AccessDenied          = errors.New("S3 access denied: check credentials")
	ErrEncryptionFailed        = errors.New("encryption failed: invalid key")
	ErrDockerSocketUnavailable = errors.New("cannot connect to Docker socket")
	ErrVolumeNotFound          = errors.New("volume not found")
)


type BackupError struct {
	Phase      string
	Component  string
	Err        error
	Retryable  bool
	Suggestion string
}

func (e *BackupError) Error() string {
	msg := fmt.Sprintf("[%s] %s: %v", e.Phase, e.Component, e.Err)
	if e.Suggestion != "" {
		msg += fmt.Sprintf("\n Suggestion: %s", e.Suggestion)
	}
	return msg
}

func (e *BackupError) Unwrap() error {
	return e.Err
}


func NewBackupError(phase, component string, err error, retryable bool, suggestion string) *BackupError {
	return &BackupError{
		Phase:      phase,
		Component:  component,
		Err:        err,
		Retryable:  retryable,
		Suggestion: suggestion,
	}
}


func IsRetryable(err error) bool {
	var backupErr *BackupError
	if errors.As(err, &backupErr) {
		return backupErr.Retryable
	}


	errStr := err.Error()
	return errors.Is(err, ErrNetworkTimeout) ||
		contains(errStr, "connection reset") ||
		contains(errStr, "temporary failure") ||
		contains(errStr, "timeout")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) &&
		(s[:len(substr)] == substr || s[len(s)-len(substr):] == substr ||
			findSubstring(s, substr)))
}

func findSubstring(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
