package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/stacksnap/stacksnap/internal/crypto"
	"github.com/stacksnap/stacksnap/internal/docker"
	"github.com/stacksnap/stacksnap/internal/storage"
)


type StackRestoreOptions struct {
	StackName       string
	InputPath       string
	StorageProvider storage.Provider
	EncryptionKey   []byte
	Context         context.Context
	Logger          func(string)
}


func RestoreStack(client *docker.Client, opts StackRestoreOptions) error {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}


	log := func(format string, args ...interface{}) {
		msg := fmt.Sprintf(format, args...)
		fmt.Print(msg)
		if opts.Logger != nil {
			opts.Logger(msg)
		}
	}

	log("  Restoring stack %s from %s...\n", opts.StackName, opts.InputPath)


	var reader io.ReadCloser
	var err error

	if opts.StorageProvider != nil {
		log("  Downloading from remote storage...\n")
		reader, err = opts.StorageProvider.Download(ctx, opts.InputPath)
		if err != nil {
			return fmt.Errorf("failed to download backup: %w", err)
		}
	} else {
		reader, err = os.Open(opts.InputPath)
		if err != nil {
			return fmt.Errorf("failed to open backup file: %w", err)
		}
	}
	defer reader.Close()


	var input io.Reader = reader
	if opts.EncryptionKey != nil {
		log(" Decrypting parameters...\n")
		decReader, err := crypto.NewDecryptReader(opts.EncryptionKey, reader)
		if err != nil {
			return fmt.Errorf("failed to create decryption reader: %w", err)
		}
		input = decReader
	}


	gzReader, err := gzip.NewReader(input)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
	}
	defer gzReader.Close()




	var restartedContainers []string
	serviceToImage := make(map[string]string)
	var projectWorkingDir string
	var projectConfigFile string

	if opts.StackName != "" {
		ctrs, err := client.ListContainersForProject(opts.StackName)
		if err == nil {
			for _, ctr := range ctrs {

				containerName := strings.TrimPrefix(ctr.Name, "/")
				serviceToImage[containerName] = ctr.Image


				if projectWorkingDir == "" {
					if wd, ok := ctr.Labels["com.docker.compose.project.working_dir"]; ok {
						projectWorkingDir = wd
					}
					if cf, ok := ctr.Labels["com.docker.compose.project.config_files"]; ok {
						projectConfigFile = cf
					}
				}

				if ctr.State == "running" {
					log("⏸  Stopping container %s for restore...\n", ctr.Name)
					if err := client.StopContainer(ctr.ID); err == nil {
						restartedContainers = append(restartedContainers, ctr.ID)
					}
				}
			}
		}

	}


	defer func() {



		recreated := false
		if projectWorkingDir != "" {
			log(" Recreating containers via Docker Compose in %s...\n", projectWorkingDir)


			args := []string{"compose"}

			if projectConfigFile != "" {

				configs := strings.Split(projectConfigFile, ",")
				for _, cfg := range configs {
					args = append(args, "-f", cfg)
				}
			}


			args = append(args, "up", "-d")

			cmd := exec.Command("docker", args...)
			cmd.Dir = projectWorkingDir
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err == nil {
				recreated = true
				log(" Containers recreated successfully\n")
			} else {
				log("  Map-based restore failed: %v. Falling back to simple restart.\n", err)
			}
		}

		if !recreated {
			for _, id := range restartedContainers {
				log("  Restarting container %s after restore (legacy restart)...\n", id)
				if err := client.StartContainer(id); err != nil {
					log("  Warning: failed to restart container %s: %v\n", id, err)
				}
			}
		}
	}()

	log(" Restoring volume from archive...\n")


	tarReader := tar.NewReader(gzReader)
	foundVolumes := 0

	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read tar header: %w", err)
		}


		if strings.HasPrefix(header.Name, "volumes/") && strings.HasSuffix(header.Name, ".tar") {

			baseName := filepath.Base(header.Name)
			volName := strings.TrimSuffix(baseName, ".tar")

			log(" Restoring volume: %s (Size: %d bytes)\n", volName, header.Size)
















			err := client.RestoreVolume(volName, tarReader)
			if err != nil {
				log("  Failed to restore volume %s: %v\n", volName, err)
			} else {
				log(" Volume %s restored\n", volName)
				foundVolumes++
			}
		} else if strings.HasPrefix(header.Name, "images/") && strings.HasSuffix(header.Name, ".tar") {

			log(" Restoring snapshot image: %s...\n", header.Name)


			tmpFile, err := os.CreateTemp("", "stacksnap-image-*.tar")
			if err != nil {
				log("  Failed to create temp file for image %s: %v\n", header.Name, err)
				continue
			}
			defer os.Remove(tmpFile.Name())

			if _, err := io.Copy(tmpFile, tarReader); err != nil {
				tmpFile.Close()
				log("  Failed to write image temp file %s: %v\n", header.Name, err)
				continue
			}
			tmpFile.Close()

			if err := client.LoadImage(tmpFile.Name()); err != nil {
				log("  Failed to load image %s: %v\n", header.Name, err)
				continue
			}


			baseName := filepath.Base(header.Name)
			serviceName := strings.TrimSuffix(baseName, ".tar")


			if targetImage, ok := serviceToImage[serviceName]; ok && targetImage != "" {



				log("    Retagging to: %s\n", targetImage)


				filterPattern := fmt.Sprintf("stacksnap-backup-%s", serviceName)
				out, err := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", fmt.Sprintf("reference=%s:*", filterPattern)).Output()

				if err != nil {

					out, _ = exec.Command("bash", "-c", fmt.Sprintf("docker images --format '{{.Repository}}:{{.Tag}}' | grep 'stacksnap-backup-%s:' | head -1", serviceName)).Output()
				}

				sourceTag := strings.TrimSpace(string(out))

				if strings.Contains(sourceTag, "\n") {
					sourceTag = strings.Split(sourceTag, "\n")[0]
				}

				if sourceTag != "" && sourceTag != "<none>:<none>" {
					if err := client.TagImage(sourceTag, targetImage); err != nil {
						log("  Failed to retag %s to %s: %v\n", sourceTag, targetImage, err)
					} else {
						log(" Image restored: %s -> %s\n", sourceTag, targetImage)
					}
				} else {

					debugOut, _ := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}", "--filter", "reference=stacksnap-backup*").Output()
					log("  Could not find loaded image for %s\n", serviceName)
					if len(debugOut) > 0 {
						log("   Available backup images: %s\n", strings.TrimSpace(string(debugOut)))
					}
				}
			} else {
				log(" Snapshot loaded for %s (no retagging - service not running)\n", serviceName)
			}

		}

	}

	if foundVolumes == 0 {
		return fmt.Errorf("no volumes found in backup archive (is this a valid stack backup?)")
	}

	log(" Stack restore complete!\n")
	return nil
}


func PeekBackup(opts StackRestoreOptions) ([]string, error) {
	ctx := opts.Context
	if ctx == nil {
		ctx = context.Background()
	}

	var reader io.ReadCloser
	var err error

	if opts.StorageProvider != nil {
		reader, err = opts.StorageProvider.Download(ctx, opts.InputPath)
		if err != nil {
			return nil, err
		}
	} else {
		reader, err = os.Open(opts.InputPath)
		if err != nil {
			return nil, err
		}
	}
	defer reader.Close()

	var input io.Reader = reader
	if opts.EncryptionKey != nil {
		decReader, err := crypto.NewDecryptReader(opts.EncryptionKey, reader)
		if err != nil {
			return nil, err
		}
		input = decReader
	}

	gzReader, err := gzip.NewReader(input)
	if err != nil {
		return nil, err
	}
	defer gzReader.Close()

	tarReader := tar.NewReader(gzReader)
	var files []string
	for {
		header, err := tarReader.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
		files = append(files, header.Name)
	}

	return files, nil
}
