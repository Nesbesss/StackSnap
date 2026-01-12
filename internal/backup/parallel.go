package backup

import (
	"fmt"
	"io"
	"os/exec"
	"runtime"
	"sync"
)


type ParallelConfig struct {
	MaxWorkers           int
	UseParallelGzip      bool
	GzipCompressionLevel int
}


func DefaultParallelConfig() ParallelConfig {
	workers := runtime.NumCPU() / 2
	if workers < 1 {
		workers = 1
	}
	if workers > 4 {
		workers = 4
	}

	return ParallelConfig{
		MaxWorkers:           workers,
		UseParallelGzip:      true,
		GzipCompressionLevel: 6,
	}
}


func pigzAvailable() bool {
	cmd := exec.Command("which", "pigz")
	return cmd.Run() == nil
}


type ParallelGzipWriter struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.Writer
	done   chan error
}



func NewParallelGzipWriter(w io.Writer, level int, threads int) (io.WriteCloser, error) {
	if !pigzAvailable() {


		return nil, fmt.Errorf("pigz not available, use standard gzip")
	}

	if threads <= 0 {
		threads = runtime.NumCPU()
	}


	cmd := exec.Command("pigz",
		"-c",
		fmt.Sprintf("-%d", level),
		"-p", fmt.Sprintf("%d", threads),
	)

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	cmd.Stdout = w

	done := make(chan error, 1)

	if err := cmd.Start(); err != nil {
		stdin.Close()
		return nil, fmt.Errorf("failed to start pigz: %w", err)
	}

	go func() {
		done <- cmd.Wait()
	}()

	return &ParallelGzipWriter{
		cmd:    cmd,
		stdin:  stdin,
		stdout: w,
		done:   done,
	}, nil
}

func (p *ParallelGzipWriter) Write(data []byte) (int, error) {
	return p.stdin.Write(data)
}

func (p *ParallelGzipWriter) Close() error {

	if err := p.stdin.Close(); err != nil {
		return err
	}


	return <-p.done
}


type VolumeBackupJob struct {
	VolumeName string
	OutputPath string
	Error      error
	Size       int64
}


func ParallelVolumeBackup(
	volumes []string,
	backupFunc func(volumeName string) (*VolumeBackupJob, error),
	cfg ParallelConfig,
	progress func(completed, total int, current string),
) ([]*VolumeBackupJob, error) {
	if cfg.MaxWorkers <= 0 {
		cfg.MaxWorkers = 1
	}

	results := make([]*VolumeBackupJob, len(volumes))
	var wg sync.WaitGroup
	sem := make(chan struct{}, cfg.MaxWorkers)
	var mu sync.Mutex
	completed := 0

	for i, vol := range volumes {
		wg.Add(1)
		go func(idx int, volumeName string) {
			defer wg.Done()


			sem <- struct{}{}
			defer func() { <-sem }()


			if progress != nil {
				mu.Lock()
				progress(completed, len(volumes), volumeName)
				mu.Unlock()
			}


			result, err := backupFunc(volumeName)
			if err != nil {
				result = &VolumeBackupJob{
					VolumeName: volumeName,
					Error:      err,
				}
			}

			results[idx] = result


			mu.Lock()
			completed++
			if progress != nil {
				progress(completed, len(volumes), volumeName)
			}
			mu.Unlock()
		}(i, vol)
	}

	wg.Wait()


	var firstError error
	for _, r := range results {
		if r != nil && r.Error != nil && firstError == nil {
			firstError = r.Error
		}
	}

	return results, firstError
}


func EstimateBackupTime(totalSizeMB int64, parallelWorkers int, uploadSpeedMBps float64) string {





	compressionRate := 50.0
	effectiveRate := compressionRate * float64(parallelWorkers)
	if effectiveRate > 200 {
		effectiveRate = 200
	}


	compressionTimeSec := float64(totalSizeMB) / effectiveRate


	uploadSizeMB := float64(totalSizeMB) * 0.5
	uploadTimeSec := uploadSizeMB / uploadSpeedMBps

	totalTimeSec := compressionTimeSec + uploadTimeSec

	if totalTimeSec < 60 {
		return fmt.Sprintf("%.0f seconds", totalTimeSec)
	} else if totalTimeSec < 3600 {
		return fmt.Sprintf("%.1f minutes", totalTimeSec/60)
	}
	return fmt.Sprintf("%.1f hours", totalTimeSec/3600)
}
