
package backup

import (
	"compress/gzip"
	"fmt"
	"os"
	"time"

	"github.com/stacksnap/stacksnap/internal/docker"
)


type BackupOptions struct {
	VolumeName      string
	OutputPath      string
	PauseContainers bool
}


type BackupResult struct {
	VolumeName       string
	OutputPath       string
	Size             int64
	Duration         time.Duration
	PausedContainers []string
}


func Backup(client *docker.Client, opts BackupOptions) (*BackupResult, error) {
	startTime := time.Now()
	var pausedContainers []string


	exists, err := client.VolumeExists(opts.VolumeName)
	if err != nil {
		return nil, fmt.Errorf("failed to check volume: %w", err)
	}
	if !exists {
		return nil, fmt.Errorf("volume %q not found", opts.VolumeName)
	}


	if opts.PauseContainers {
		containers, err := client.ListContainersUsingVolume(opts.VolumeName)
		if err != nil {
			return nil, fmt.Errorf("failed to list containers: %w", err)
		}

		for _, ctr := range containers {
			if ctr.State == "running" {
				fmt.Printf("⏸  Pausing container %s...\n", ctr.Name)
				if err := client.PauseContainer(ctr.ID); err != nil {

					for _, id := range pausedContainers {
						client.UnpauseContainer(id)
					}
					return nil, fmt.Errorf("failed to pause container %s: %w", ctr.Name, err)
				}
				pausedContainers = append(pausedContainers, ctr.ID)
			}
		}


		defer func() {
			for _, id := range pausedContainers {
				fmt.Printf("  Resuming container...\n")
				if err := client.UnpauseContainer(id); err != nil {
					fmt.Printf("  Warning: failed to unpause container: %v\n", err)
				}
			}
		}()
	}


	outputPath := opts.OutputPath
	if outputPath == "" {
		timestamp := time.Now().Format("20060102_150405")
		outputPath = fmt.Sprintf("%s_%s.tar.gz", opts.VolumeName, timestamp)
	}


	outFile, err := os.Create(outputPath)
	if err != nil {
		return nil, fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()


	gzWriter := gzip.NewWriter(outFile)

	fmt.Printf(" Backing up volume %q...\n", opts.VolumeName)


	if err := client.BackupVolume(opts.VolumeName, gzWriter); err != nil {
		gzWriter.Close()
		os.Remove(outputPath)
		return nil, fmt.Errorf("failed to create backup: %w", err)
	}


	if err := gzWriter.Close(); err != nil {
		os.Remove(outputPath)
		return nil, fmt.Errorf("failed to finalize backup: %w", err)
	}
	outFile.Sync()


	stat, err := outFile.Stat()
	if err != nil {
		return nil, fmt.Errorf("failed to stat output file: %w", err)
	}

	duration := time.Since(startTime)

	pauseNote := ""
	if len(pausedContainers) > 0 {
		pauseNote = fmt.Sprintf(" (paused %d container(s))", len(pausedContainers))
	}

	fmt.Printf(" Backup complete: %s (%.2f MB in %s)%s\n",
		outputPath,
		float64(stat.Size())/(1024*1024),
		duration.Round(time.Millisecond),
		pauseNote)

	return &BackupResult{
		VolumeName:       opts.VolumeName,
		OutputPath:       outputPath,
		Size:             stat.Size(),
		Duration:         duration,
		PausedContainers: pausedContainers,
	}, nil
}
