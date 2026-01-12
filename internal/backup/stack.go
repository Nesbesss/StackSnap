
package backup

import (
	"archive/tar"
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/stacksnap/stacksnap/internal/compose"
	"github.com/stacksnap/stacksnap/internal/crypto"
	"github.com/stacksnap/stacksnap/internal/database"
	"github.com/stacksnap/stacksnap/internal/docker"
	"github.com/stacksnap/stacksnap/internal/storage"
)


type StackBackupOptions struct {
	Directory       string
	ProjectName     string
	OutputPath      string
	PauseContainers bool
	IncludeDatabase bool
	SnapshotImages  bool


	StorageProvider storage.Provider
	EncryptionKey   []byte
	Context         context.Context
	Logger          func(string)
}


type StackBackupResult struct {
	StackName        string
	OutputPath       string
	Size             int64
	Duration         time.Duration
	VolumesBackedUp  []string
	DatabasesDumped  []string
	PausedContainers int
	Encrypted        bool
}


type StackMetadata struct {
	StackName    string    `json:"stack_name"`
	CreatedAt    time.Time `json:"created_at"`
	ComposeFile  string    `json:"compose_file"`
	Volumes      []string  `json:"volumes"`
	Services     []string  `json:"services"`
	Databases    []string  `json:"databases,omitempty"`
	Secrets      []string  `json:"secrets,omitempty"`
	BuildFiles   []string  `json:"build_files,omitempty"`
	Images       []string  `json:"images,omitempty"`
	StackSnapVer string    `json:"stacksnap_version"`
	Encrypted    bool      `json:"encrypted"`
}


func BackupStack(client *docker.Client, opts StackBackupOptions) (*StackBackupResult, error) {
	startTime := time.Now()
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





	preflightResult := PreflightChecks(client, opts)
	if len(preflightResult.Warnings) > 0 {
		log("⚠️  Pre-flight check warnings:\n")
		for _, warning := range preflightResult.Warnings {
			icon := "ℹ️"
			if warning.Severity == "warning" {
				icon = "⚠️"
			} else if warning.Severity == "error" {
				icon = "❌"
			}
			log("%s  %s\n", icon, warning.Message)
			if warning.Fix != "" {
				log("   💡 %s\n", warning.Fix)
			}
		}
		if !preflightResult.CanProceed {
			return nil, fmt.Errorf("pre-flight checks failed - cannot proceed with backup")
		}
		log("\n")
	}


	var stack *compose.Stack
	if opts.Directory != "" {
		s, err := compose.DiscoverStack(opts.Directory)
		if err != nil {
			return nil, fmt.Errorf("failed to discover stack in %s: %w", opts.Directory, err)
		}
		stack = s
	} else if opts.ProjectName != "" {

		log("ℹ️  Using label-based discovery for project: %s\n", opts.ProjectName)
		vols, err := client.ListVolumesForProject(opts.ProjectName)
		if err != nil {
			log("Warning: failed to list volumes for project %s: %v\n", opts.ProjectName, err)
		}
		stack = &compose.Stack{
			Name:         opts.ProjectName,
			NamedVolumes: vols,
			IsStandalone: true,
		}
	} else {
		return nil, fmt.Errorf("either Directory or ProjectName must be provided for backup")
	}

	log("🐳 Backing up stack: %s\n", stack.Name)
	if opts.EncryptionKey != nil {
		log("🔒 Encryption enabled (AES-256-CTR)\n")
	}
	if opts.StorageProvider != nil {
		log("☁️  Uploading to remote storage\n")
	}


	timestamp := time.Now().Format("20060102_150405")
	filename := fmt.Sprintf("%s_%s.tar.gz", stack.Name, timestamp)
	if opts.EncryptionKey != nil {
		filename += ".enc"
	}



	var finalWriter io.WriteCloser
	var uploadErrCh chan error

	if opts.StorageProvider != nil {


		pr, pw := io.Pipe()
		finalWriter = pw

		uploadErrCh = make(chan error, 1)
		go func() {
			log("☁️  Starting upload to: %s\n", filename)
			err := opts.StorageProvider.Upload(ctx, filename, pr)
			if err != nil {
				log("❌ Upload failed: %v\n", err)
			} else {
				log("✅ Upload complete\n")
			}

			io.Copy(io.Discard, pr)
			uploadErrCh <- err
		}()
	} else {

		outputPath := opts.OutputPath
		if outputPath == "" {
			outputPath = filename
		}
		f, err := os.Create(outputPath)
		if err != nil {
			return nil, fmt.Errorf("failed to create output file: %w", err)
		}
		finalWriter = f
		filename = outputPath
	}





	var outputStream io.WriteCloser = finalWriter


	if opts.EncryptionKey != nil {
		encWriter, err := crypto.NewEncryptWriter(opts.EncryptionKey, outputStream)
		if err != nil {
			finalWriter.Close()
			return nil, fmt.Errorf("failed to create encryption writer: %w", err)
		}
		outputStream = encWriter
	}


	gzWriter := gzip.NewWriter(outputStream)


	tarWriter := tar.NewWriter(gzWriter)




	allContainers, err := client.ListContainersForProject(stack.Name)
	if err != nil {
		log("Warning: failed to list containers for project %s: %v\n", stack.Name, err)

		for _, volName := range stack.NamedVolumes {
			ctrs, _ := client.ListContainersUsingVolume(volName)
			for _, c := range ctrs {
				exists := false
				for _, e := range allContainers {
					if e.ID == c.ID {
						exists = true
						break
					}
				}
				if !exists {
					allContainers = append(allContainers, c)
				}
			}
		}
	}


	for _, ctr := range allContainers {
		for _, vName := range ctr.Volumes {
			alreadyTracked := false
			for _, existing := range stack.NamedVolumes {
				if existing == vName {
					alreadyTracked = true
					break
				}
			}
			if !alreadyTracked {
				log("ℹ️  Found implicit volume mount: %s (Adding to backup)\n", vName)
				stack.NamedVolumes = append(stack.NamedVolumes, vName)
			}
		}
	}


	var pausedContainers []string
	if opts.PauseContainers {
		for _, ctr := range allContainers {
			if ctr.State == "running" {


				dbInfo, _ := database.DetectDatabase(client, ctr.ID)
				if dbInfo != nil && dbInfo.Type != database.DatabaseUnknown {
					log("ℹ️  Skipping pause for DB container: %s\n", ctr.Name)
					continue
				}

				log("⏸️  Pausing %s...\n", ctr.Name)
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
				log("▶️  Resuming container...\n")
				client.UnpauseContainer(id)
			}
		}()


		if len(pausedContainers) > 0 {
			log("⏳ Waiting for database write queues to drain...\n")
			time.Sleep(2 * time.Second)
		}
	}


	var backedUpImages []string
	if opts.SnapshotImages {
		log("📸 Creating container snapshots...\n")

		imgTmpDir, err := os.MkdirTemp("", "stacksnap-images")
		if err == nil {
			defer os.RemoveAll(imgTmpDir)

			for _, ctr := range allContainers {

				if ctr.State == "running" || ctr.State == "paused" {
					log("   - Snapshotting %s...\n", ctr.Name)


					ts := time.Now().Format("20060102150405")
					safeName := strings.ReplaceAll(ctr.Name, "/", "")


					backupTag := fmt.Sprintf("stacksnap-backup-%s:%s", safeName, ts)


					imgID, err := client.CommitContainer(ctr.ID, backupTag)
					if err != nil {
						log("⚠️  Failed to commit container %s: %v\n", ctr.Name, err)
						continue
					}


					tarName := fmt.Sprintf("%s.tar", safeName)
					outPath := filepath.Join(imgTmpDir, tarName)


					if err := client.SaveImage(backupTag, outPath); err != nil {
						log("⚠️  Failed to save image %s: %v\n", backupTag, err)
						client.RemoveImage(imgID)
						continue
					}


					f, err := os.Open(outPath)
					if err == nil {
						info, _ := f.Stat()
						header, _ := tar.FileInfoHeader(info, "")
						header.Name = fmt.Sprintf("images/%s", tarName)

						if err := tarWriter.WriteHeader(header); err == nil {
							io.Copy(tarWriter, f)
							backedUpImages = append(backedUpImages, backupTag)
						}
						f.Close()
					}


					client.RemoveImage(imgID)
				}
			}
		} else {
			log("⚠️  Failed to create temp dir for images: %v\n", err)
		}
	}


	var databasesDumped []string
	if opts.IncludeDatabase {
		for _, ctr := range allContainers {
			dbInfo, err := database.DetectDatabase(client, ctr.ID)
			if err != nil || dbInfo.Type == database.DatabaseUnknown {
				continue
			}

			log("🗄️  Dumping %s database from %s...\n", dbInfo.Type, ctr.Name)


			isCurrentlyPaused := false
			for _, id := range pausedContainers {
				if id == ctr.ID {
					isCurrentlyPaused = true
					break
				}
			}

			if isCurrentlyPaused {
				client.UnpauseContainer(ctr.ID)
			}

			dumpReader, err := database.Dump(client, dbInfo)
			if err != nil {
				log("⚠️  Warning: failed to dump database %s: %v\n", ctr.Name, err)
				if isCurrentlyPaused {
					client.PauseContainer(ctr.ID)
				}
				continue
			}


			if isCurrentlyPaused {
				client.PauseContainer(ctr.ID)
			}


			dumpData, err := io.ReadAll(dumpReader)
			if err != nil {
				log("⚠️  Warning: failed to read database dump: %v\n", err)
				continue
			}

			dumpFilename := fmt.Sprintf("%s_%s_dump.sql", ctr.Name, dbInfo.Type)
			if err := addToTar(tarWriter, dumpFilename, dumpData); err != nil {
				log("⚠️  Warning: failed to add dump to archive: %v\n", err)
				continue
			}
			databasesDumped = append(databasesDumped, string(dbInfo.Type))
		}
	}


	composeData, err := os.ReadFile(stack.ComposeFile)
	if err == nil {
		addToTar(tarWriter, filepath.Base(stack.ComposeFile), composeData)
	}


	for _, envPath := range stack.EnvFiles {
		envData, err := os.ReadFile(envPath)
		if err == nil {
			addToTar(tarWriter, filepath.Base(envPath), envData)
		}
	}


	for _, secPath := range stack.SecretFiles {
		secData, err := os.ReadFile(secPath)
		if err == nil {
			addToTar(tarWriter, filepath.Base(secPath), secData)
		}
	}


	for _, bPath := range stack.BuildFiles {
		bData, err := os.ReadFile(bPath)
		if err == nil {
			addToTar(tarWriter, filepath.Base(bPath), bData)
		}
	}


	var volumesBackedUp []string
	for _, volName := range stack.NamedVolumes {
		log("🔄 Backing up volume %s...\n", volName)

		pr, pw := io.Pipe()

		errCh := make(chan error, 1)
		go func() {
			err := client.BackupVolume(volName, pw)
			pw.CloseWithError(err)
			errCh <- err
		}()























		tempFile, err := os.CreateTemp("", "stacksnap-vol-*.tar")
		if err != nil {
			log("⚠️  Failed to create temp file: %v\n", err)
			continue
		}


		size, err := io.Copy(tempFile, pr)

		backupErr := <-errCh
		if backupErr != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			log("⚠️  Failed to backup volume %s: %v\n", volName, backupErr)
			continue
		}


		tempFile.Seek(0, 0)

		header := &tar.Header{
			Name:    filepath.Join("volumes", volName+".tar"),
			Size:    size,
			Mode:    0644,
			ModTime: time.Now(),
		}

		if err := tarWriter.WriteHeader(header); err != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			log("⚠️  Failed to write header: %v\n", err)
			continue
		}

		if _, err := io.Copy(tarWriter, tempFile); err != nil {
			tempFile.Close()
			os.Remove(tempFile.Name())
			log("⚠️  Failed to copy volume data: %v\n", err)
			continue
		}

		tempFile.Close()
		os.Remove(tempFile.Name())
		volumesBackedUp = append(volumesBackedUp, volName)
	}

	var metadataSecrets []string
	for _, s := range stack.SecretFiles {
		metadataSecrets = append(metadataSecrets, filepath.Base(s))
	}

	var metadataBuildFiles []string
	for _, b := range stack.BuildFiles {
		metadataBuildFiles = append(metadataBuildFiles, filepath.Base(b))
	}

	var serviceNames []string
	for k := range stack.Services {
		serviceNames = append(serviceNames, k)
	}


	metadata := StackMetadata{
		StackName:    stack.Name,
		CreatedAt:    time.Now(),
		ComposeFile:  filepath.Base(stack.ComposeFile),
		Volumes:      volumesBackedUp,
		Services:     serviceNames,
		Databases:    databasesDumped,
		Secrets:      metadataSecrets,
		BuildFiles:   metadataBuildFiles,
		Images:       backedUpImages,
		StackSnapVer: "1.0",
		Encrypted:    opts.EncryptionKey != nil,
	}
	metadataJSON, _ := json.MarshalIndent(metadata, "", "  ")
	addToTar(tarWriter, "metadata.json", metadataJSON)


	tarWriter.Close()
	gzWriter.Close()

	if opts.EncryptionKey != nil {
		outputStream.Close()
	}

	finalWriter.Close()


	if uploadErrCh != nil {
		err := <-uploadErrCh
		if err != nil {
			return nil, fmt.Errorf("upload failed: %w", err)
		}
	}

	duration := time.Since(startTime)
	var finalSize int64 = 0
	if f, ok := finalWriter.(*os.File); ok {
		stat, err := f.Stat()
		if err == nil {
			finalSize = stat.Size()
		} else {

			if info, err := os.Stat(f.Name()); err == nil {
				finalSize = info.Size()
			}
		}
	}

	log("✅ Stack backup complete: %s (Duration: %s)\n",
		filename,
		duration.Round(time.Millisecond))

	return &StackBackupResult{
		StackName:        stack.Name,
		OutputPath:       filename,
		Size:             finalSize,
		Duration:         duration,
		VolumesBackedUp:  volumesBackedUp,
		DatabasesDumped:  databasesDumped,
		PausedContainers: len(pausedContainers),
		Encrypted:        opts.EncryptionKey != nil,
	}, nil
}


func addToTar(tw *tar.Writer, name string, data []byte) error {
	header := &tar.Header{
		Name:    name,
		Size:    int64(len(data)),
		Mode:    0644,
		ModTime: time.Now(),
	}

	if err := tw.WriteHeader(header); err != nil {
		return err
	}

	_, err := tw.Write(data)
	return err
}
