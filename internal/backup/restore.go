
package backup

import (
	"compress/gzip"
	"context"
	"fmt"
	"os"
	"time"

	"github.com/stacksnap/stacksnap/internal/docker"
	"github.com/stacksnap/stacksnap/internal/storage"
)


type RestoreOptions struct {
	VolumeName string
	InputPath  string

	StorageProvider storage.Provider
	Context         context.Context
	Logger          func(string)
}


type RestoreResult struct {
	VolumeName string
	InputPath  string
	Duration   time.Duration
}


func Restore(client *docker.Client, opts RestoreOptions) (*RestoreResult, error) {
	startTime := time.Now()


	stat, err := os.Stat(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("backup file not found: %w", err)
	}


	exists, err := client.VolumeExists(opts.VolumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to check volume: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("volume %q not found", opts.VolumeName)
	}

	fmt.Printf(" Restoring volume %q from %s (%.2f MB)...\n",
		opts.VolumeName,
		opts.InputPath,
		float64(stat.Size())/(1024*1024))


	inFile, err := os.Open(opts.InputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open backup file: %w", err)
	}
	defer inFile.Close()


	gzReader, err := gzip.NewReader(inFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read gzip: %w", err)
	}
	defer gzReader.Close()


	if err := client.RestoreVolume(opts.VolumeName, gzReader); err != nil {
		return nil, fmt.Errorf("failed to restore volume: %w", err)
	}

	duration := time.Since(startTime)

	fmt.Printf(" Restore complete: %s restored in %s\n",
		opts.VolumeName,
		duration.Round(time.Millisecond))

	return &RestoreResult{
		VolumeName: opts.VolumeName,
		InputPath:  opts.InputPath,
		Duration:   duration,
	}, nil
}
