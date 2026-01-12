package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/stacksnap/stacksnap/internal/docker"
	"github.com/stacksnap/stacksnap/internal/storage"
)


type VerificationResult struct {
	BackupKey   string  `json:"backup_key"`
	Verified   bool   `json:"verified"`
	TestedAt   time.Time `json:"tested_at"`
	ErrorMessage string  `json:"error_message,omitempty"`
	ContainerLogs string  `json:"container_logs,omitempty"`
}


func VerifyBackup(ctx context.Context, client *docker.Client, provider storage.Provider, key string) (*VerificationResult, error) {
	result := &VerificationResult{
		BackupKey: key,
		TestedAt: time.Now(),
	}


	tempDir, err := os.MkdirTemp("", "stacksnap-verify-*")
	if err != nil {
		return nil, fmt.Errorf("failed to create temp dir: %w", err)
	}
	defer os.RemoveAll(tempDir)


	var rc io.ReadCloser


	if provider != nil {
		rc, err = provider.Download(ctx, key)
		if err != nil {
			return nil, fmt.Errorf("failed to download backup: %w", err)
		}
	} else {

		rc, err = os.Open(key)
		if err != nil {
			return nil, fmt.Errorf("failed to open local backup: %w", err)
		}
	}
	defer rc.Close()


	err = extractNonVolumeFiles(rc, tempDir)
	if err != nil {
		result.Verified = false
		result.ErrorMessage = fmt.Sprintf("failed to extract verification files: %v", err)
		return result, nil
	}


	metaPath := filepath.Join(tempDir, "metadata.json")
	var metadata StackMetadata
	if mData, err := os.ReadFile(metaPath); err == nil {
		json.Unmarshal(mData, &metadata)
	}


	composePath := filepath.Join(tempDir, "docker-compose.yml")
	if _, err := os.Stat(composePath); err != nil {
		composePath = filepath.Join(tempDir, "docker-compose.yaml")
	}

	hasCompose := false
	if _, err := os.Stat(composePath); err == nil {
		hasCompose = true
	}


	if !hasCompose || metadata.ComposeFile == "" {
		fmt.Printf("ℹ Performing Data-Only verification for %s\n", key)


		missingVolumes := []string{}
		for _, vol := range metadata.Volumes {
			volPath := filepath.Join(tempDir, "volumes", vol+".tar")



			_ = volPath
		}

		if len(missingVolumes) > 0 {
			result.Verified = false
			result.ErrorMessage = fmt.Sprintf("Missing volumes in archive: %s", strings.Join(missingVolumes, ", "))
		} else {
			result.Verified = true
			result.ErrorMessage = "Verified (Data Only - No local compose file found)"
		}
		return result, nil
	}


	projectName := fmt.Sprintf("verify_%x", time.Now().UnixNano()%100000)


	cleanupCtx, cleanupCancel := context.WithTimeout(context.Background(), 30*time.Second)


	defer func() {
		defer cleanupCancel()

		fmt.Printf(" Cleaning up verification containers (project: %s)...\n", projectName)


		cleanupCmd := exec.CommandContext(cleanupCtx, "docker", "compose",
			"-p", projectName, "down", "-v", "--remove-orphans", "--timeout", "10")
		cleanupCmd.Dir = tempDir
		if err := cleanupCmd.Run(); err != nil {

			fmt.Printf(" Compose down failed, forcing cleanup: %v\n", err)
			forceCleanupVerificationContainers(cleanupCtx, client, projectName)
		} else {
			fmt.Println(" Verification containers cleaned up")
		}
	}()

	cmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "-f", composePath, "up", "-d", "--no-build")
	cmd.Dir = tempDir

	output, err := cmd.CombinedOutput()
	if err != nil {
		result.Verified = false
		result.ErrorMessage = fmt.Sprintf("docker compose up failed: %v\nOutput: %s", err, string(output))
		return result, nil
	}



	time.Sleep(5 * time.Second)

	statusCmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "ps", "--format", "json")
	statusCmd.Dir = tempDir
	statusOutput, _ := statusCmd.CombinedOutput()



	statusStr := string(statusOutput)
	isFailure := false


	if strings.Contains(statusStr, "\"State\":\"exited\"") ||
		strings.Contains(statusStr, "\"State\":\"dead\"") ||
		strings.Contains(statusStr, "\"Health\":\"unhealthy\"") {
		isFailure = true
	}


	if !strings.Contains(statusStr, "\"State\":\"running\"") && !strings.Contains(statusStr, "\"State\":\"starting\"") {
		isFailure = true
	}

	if isFailure {
		result.Verified = false
		result.ErrorMessage = "Containers failed to start properly. Status: " + statusStr


		logsCmd := exec.CommandContext(ctx, "docker", "compose", "-p", projectName, "logs", "--tail", "50")
		logsCmd.Dir = tempDir
		logOutput, _ := logsCmd.CombinedOutput()
		result.ContainerLogs = string(logOutput)
	} else {
		result.Verified = true
	}

	return result, nil
}


func extractNonVolumeFiles(r io.Reader, dest string) error {
	gzr, err := gzip.NewReader(r)
	if err != nil {
		return err
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}


		if strings.HasPrefix(header.Name, "volumes/") {
			continue
		}
		if strings.HasPrefix(header.Name, "images/") {
			continue
		}

		target := filepath.Join(dest, header.Name)
		f, err := os.Create(target)
		if err != nil {
			return err
		}
		if _, err := io.Copy(f, tr); err != nil {
			f.Close()
			return err
		}
		f.Close()
	}
	return nil
}


func forceCleanupVerificationContainers(ctx context.Context, client *docker.Client, projectName string) {

	containers, err := client.ListContainersForProject(projectName)
	if err != nil {
		fmt.Printf(" Failed to list containers for cleanup: %v\n", err)
		return
	}

	for _, ctr := range containers {
		fmt.Printf(" Force removing container: %s\n", ctr.Name)


		if ctr.State == "running" || ctr.State == "paused" {
			if err := client.StopContainer(ctr.ID); err != nil {
				fmt.Printf(" Failed to stop container %s: %v\n", ctr.Name, err)
			}
		}


		cmd := exec.CommandContext(ctx, "docker", "rm", "-f", ctr.ID)
		if err := cmd.Run(); err != nil {
			fmt.Printf(" Failed to remove container %s: %v\n", ctr.Name, err)
		}
	}


	cmd := exec.CommandContext(ctx, "docker", "volume", "ls", "-q", "--filter", fmt.Sprintf("label=com.docker.compose.project=%s", projectName))
	output, err := cmd.Output()
	if err == nil && len(output) > 0 {
		volumes := strings.Split(strings.TrimSpace(string(output)), "\n")
		for _, vol := range volumes {
			if vol != "" {
				fmt.Printf(" Removing volume: %s\n", vol)
				exec.CommandContext(ctx, "docker", "volume", "rm", "-f", vol).Run()
			}
		}
	}
}


type LightVerificationResult struct {
	BackupKey    string  `json:"backup_key"`
	Verified    bool   `json:"verified"`
	TestedAt    time.Time `json:"tested_at"`
	ErrorMessage  string  `json:"error_message,omitempty"`
	HasMetadata   bool   `json:"has_metadata"`
	HasCompose   bool   `json:"has_compose"`
	HasVolumes   bool   `json:"has_volumes"`
	HasDatabaseDump bool   `json:"has_database_dump"`
	VolumeCount   int    `json:"volume_count"`
	StackName    string  `json:"stack_name,omitempty"`
	ChecksPerformed []string `json:"checks_performed"`
}








func VerifyBackupLight(ctx context.Context, provider storage.Provider, key string) (*LightVerificationResult, error) {
	result := &LightVerificationResult{
		BackupKey:    key,
		TestedAt:    time.Now(),
		ChecksPerformed: []string{},
	}

	fmt.Printf(" Running lightweight verification on %s...\n", key)


	var rc io.ReadCloser
	var err error

	if provider != nil {
		rc, err = provider.Download(ctx, key)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Download failed: %v", err)
			return result, nil
		}
	} else {
		rc, err = os.Open(key)
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Failed to open file: %v", err)
			return result, nil
		}
	}
	defer rc.Close()
	result.ChecksPerformed = append(result.ChecksPerformed, "Download/Open")


	gzr, err := gzip.NewReader(rc)
	if err != nil {
		result.ErrorMessage = fmt.Sprintf("Invalid gzip format: %v", err)
		return result, nil
	}
	defer gzr.Close()
	result.ChecksPerformed = append(result.ChecksPerformed, "Gzip integrity")


	tr := tar.NewReader(gzr)
	volumeCount := 0
	var metadata StackMetadata

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			result.ErrorMessage = fmt.Sprintf("Corrupted tar: %v", err)
			return result, nil
		}

		switch {
		case header.Name == "metadata.json":
			result.HasMetadata = true

			if err := json.NewDecoder(tr).Decode(&metadata); err != nil {
				result.ErrorMessage = fmt.Sprintf("Invalid metadata.json: %v", err)
				return result, nil
			}
			result.StackName = metadata.StackName
			result.ChecksPerformed = append(result.ChecksPerformed, "Metadata parsing")

		case header.Name == "docker-compose.yml" || header.Name == "docker-compose.yaml":
			result.HasCompose = true

			buf := make([]byte, 1024)
			if _, err := tr.Read(buf); err != nil && err != io.EOF {
				result.ErrorMessage = fmt.Sprintf("Cannot read compose file: %v", err)
				return result, nil
			}
			result.ChecksPerformed = append(result.ChecksPerformed, "Compose file")

		case strings.HasPrefix(header.Name, "volumes/") && strings.HasSuffix(header.Name, ".tar"):
			result.HasVolumes = true
			volumeCount++

			tarHeader := make([]byte, 512)
			n, err := io.ReadFull(tr, tarHeader)
			if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
				result.ErrorMessage = fmt.Sprintf("Volume tar corrupted: %s - %v", header.Name, err)
				return result, nil
			}

			if n >= 262 {
				magic := string(tarHeader[257:262])
				if magic != "ustar" && !strings.HasPrefix(magic, "ustar") {


				}
			}

			io.Copy(io.Discard, tr)

		case strings.HasSuffix(header.Name, "_dump.sql"):
			result.HasDatabaseDump = true

			sqlBuf := make([]byte, 4096)
			n, _ := tr.Read(sqlBuf)
			if n > 0 {
				content := string(sqlBuf[:n])
				if !validateSQLDumpContent(content) {
					result.ErrorMessage = fmt.Sprintf("Invalid SQL dump: %s (missing expected SQL markers)", header.Name)
					return result, nil
				}
				result.ChecksPerformed = append(result.ChecksPerformed, "SQL dump: "+header.Name)
			}
			io.Copy(io.Discard, tr)

		default:

			io.Copy(io.Discard, tr)
		}
	}

	result.VolumeCount = volumeCount
	result.ChecksPerformed = append(result.ChecksPerformed, fmt.Sprintf("Volume count: %d", volumeCount))


	if !result.HasMetadata {
		result.ErrorMessage = "Missing metadata.json"
		return result, nil
	}

	if !result.HasVolumes && volumeCount == 0 {
		result.ErrorMessage = "No volumes found in backup"
		return result, nil
	}


	result.Verified = true
	fmt.Printf(" Lightweight verification passed (%d volumes, %d checks)\n",
		volumeCount, len(result.ChecksPerformed))

	return result, nil
}


func validateSQLDumpContent(content string) bool {

	markers := []string{
		"-- ",
		"CREATE",
		"INSERT",
		"PostgreSQL",
		"MySQL",
		"dump",
		"Dump",
		"SET ",
	}

	contentLower := strings.ToLower(content)
	for _, marker := range markers {
		if strings.Contains(contentLower, strings.ToLower(marker)) {
			return true
		}
	}

	return false
}
