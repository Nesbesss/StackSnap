package main

import (
	"context"
	"fmt"
	"net/http"
	"os"

	"embed"
	"io/fs"

	"github.com/spf13/cobra"
	"github.com/stacksnap/stacksnap/internal/api"
	"github.com/stacksnap/stacksnap/internal/backup"
	"github.com/stacksnap/stacksnap/internal/compose"
	"github.com/stacksnap/stacksnap/internal/docker"
	"github.com/stacksnap/stacksnap/internal/storage"
)

//go:embed dist
var uiEmbed embed.FS

var version = "0.3.0"

func main() {
	rootCmd := &cobra.Command{
		Use:     "stacksnap",
		Short:   "Docker backups that actually work when you need them",
		Version: version,
	}

	rootCmd.AddCommand(discoverCmd())
	rootCmd.AddCommand(backupCmd())
	rootCmd.AddCommand(backupStackCmd())
	rootCmd.AddCommand(restoreCmd())
	rootCmd.AddCommand(listCmd())
	rootCmd.AddCommand(serverCmd())

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func discoverCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "discover",
		Short: "Discover docker-compose stacks and their volumes",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			stack, err := compose.DiscoverStack(cwd)
			if err != nil {
				return err
			}

			fmt.Printf(" Stack: %s\n", stack.Name)
			fmt.Printf(" Compose file: %s\n", stack.ComposeFile)
			fmt.Printf("\n Services:\n")
			for _, svc := range stack.Services {
				fmt.Printf("  • %s\n", svc)
			}

			if len(stack.NamedVolumes) > 0 {
				fmt.Printf("\n Named Volumes:\n")
				for _, vol := range stack.NamedVolumes {
					fmt.Printf("  • %s\n", vol)
				}
			}

			if len(stack.VolumeMounts) > 0 {
				fmt.Printf("\n Volume Mounts:\n")
				for _, mount := range stack.VolumeMounts {
					volType := "bind"
					if mount.IsNamed {
						volType = "named"
					}
					fmt.Printf("  • %s → %s (%s, service: %s)\n",
						mount.Source, mount.Target, volType, mount.ServiceName)
				}
			}

			return nil
		},
	}
}

func backupCmd() *cobra.Command {
	var output string
	var pause bool

	cmd := &cobra.Command{
		Use:   "backup <volume-name>",
		Short: "Backup a Docker volume to a .tar.gz file",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeName := args[0]

			client, err := docker.NewClient()
			if err != nil {
				return err
			}
			defer client.Close()

			if err := client.Ping(); err != nil {
				return fmt.Errorf("cannot connect to Docker: %w", err)
			}

			_, err = backup.Backup(client, backup.BackupOptions{
				VolumeName:      volumeName,
				OutputPath:      output,
				PauseContainers: pause,
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: <volume>_<timestamp>.tar.gz)")
	cmd.Flags().BoolVarP(&pause, "pause", "p", true, "Pause containers during backup for consistency")
	return cmd
}

func backupStackCmd() *cobra.Command {
	var output string
	var pause bool
	var dumpDatabases bool

	var s3Bucket string
	var s3Region string
	var s3Endpoint string
	var s3AccessKey string
	var s3SecretKey string

	var encryptionKey string

	cmd := &cobra.Command{
		Use:   "backup-stack",
		Short: "Backup an entire docker-compose stack (all volumes + compose file)",
		RunE: func(cmd *cobra.Command, args []string) error {
			cwd, err := os.Getwd()
			if err != nil {
				return fmt.Errorf("failed to get current directory: %w", err)
			}

			var keyBytes []byte
			if encryptionKey != "" {
				if len(encryptionKey) != 32 {
					return fmt.Errorf("encryption key must be exactly 32 bytes (got %d)", len(encryptionKey))
				}
				keyBytes = []byte(encryptionKey)
			}

			var provider storage.Provider
			if s3Bucket != "" {
				var err error
				provider, err = storage.NewS3Provider(context.Background(), s3Bucket, s3Region, s3Endpoint, s3AccessKey, s3SecretKey)
				if err != nil {
					return err
				}
			}

			client, err := docker.NewClient()
			if err != nil {
				return err
			}
			defer client.Close()

			if err := client.Ping(); err != nil {
				return fmt.Errorf("cannot connect to Docker: %w", err)
			}

			_, err = backup.BackupStack(client, backup.StackBackupOptions{
				Directory:       cwd,
				OutputPath:      output,
				PauseContainers: pause,
				IncludeDatabase: dumpDatabases,
				StorageProvider: provider,
				EncryptionKey:   keyBytes,
			})
			return err
		},
	}

	cmd.Flags().StringVarP(&output, "output", "o", "", "Output file path (default: <stack>_<timestamp>.tar.gz)")
	cmd.Flags().BoolVarP(&pause, "pause", "p", true, "Pause containers during backup for consistency")
	cmd.Flags().BoolVarP(&dumpDatabases, "databases", "d", true, "Dump databases (PostgreSQL, MySQL) before backup")

	cmd.Flags().StringVar(&s3Bucket, "s3-bucket", "", "S3 bucket name to upload backup to")
	cmd.Flags().StringVar(&s3Region, "s3-region", "us-east-1", "AWS region")
	cmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", "", "S3 endpoint URL (for LocalStack/MinIO)")
	cmd.Flags().StringVar(&s3AccessKey, "s3-access-key", "", "AWS Access Key ID")
	cmd.Flags().StringVar(&s3SecretKey, "s3-secret-key", "", "AWS Secret Access Key")

	cmd.Flags().StringVar(&encryptionKey, "encryption-key", "", "32-byte encryption key for AES-256")

	return cmd
}

func restoreCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "restore <volume-name> <backup-file>",
		Short: "Restore a Docker volume from a .tar.gz backup",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			volumeName := args[0]
			backupFile := args[1]

			client, err := docker.NewClient()
			if err != nil {
				return err
			}
			defer client.Close()

			if err := client.Ping(); err != nil {
				return fmt.Errorf("cannot connect to Docker: %w", err)
			}

			_, err = backup.Restore(client, backup.RestoreOptions{
				VolumeName: volumeName,
				InputPath:  backupFile,
			})
			return err
		},
	}
}

func listCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "list [prefix]",
		Short: "List Docker volumes",
		Args:  cobra.MaximumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			client, err := docker.NewClient()
			if err != nil {
				return err
			}
			defer client.Close()

			if err := client.Ping(); err != nil {
				return fmt.Errorf("cannot connect to Docker: %w", err)
			}

			prefix := ""
			if len(args) > 0 {
				prefix = args[0]
			}

			volumes, err := client.ListVolumes(prefix)
			if err != nil {
				return err
			}

			if len(volumes) == 0 {
				fmt.Println("No volumes found")
				return nil
			}

			fmt.Printf(" Docker Volumes:\n")
			for _, v := range volumes {
				fmt.Printf("  • %s\n", v)
			}
			return nil
		},
	}
}

func serverCmd() *cobra.Command {
	var port int

	var s3Bucket string
	var s3Region string
	var s3Endpoint string
	var s3AccessKey string
	var s3SecretKey string

	cmd := &cobra.Command{
		Use:   "server",
		Short: "Start the API server and Web UI",
		RunE: func(cmd *cobra.Command, args []string) error {

			var provider storage.Provider
			if s3Bucket != "" {
				var err error
				provider, err = storage.NewS3Provider(context.Background(), s3Bucket, s3Region, s3Endpoint, s3AccessKey, s3SecretKey)
				if err != nil {
					return err
				}
				fmt.Printf(" Using S3 Storage: %s (Endpoint: %s)\n", s3Bucket, s3Endpoint)
			}

			var uiFS fs.FS
			uiFS, err := fs.Sub(uiEmbed, "dist")
			if err != nil {

				fmt.Println("Warning: Could not load embedded UI:", err)
			}

			server := api.NewServer(provider, uiFS)

			addr := fmt.Sprintf(":%d", port)
			fmt.Printf(" Starting server on http://localhost%s\n", addr)
			return http.ListenAndServe(addr, server)
		},
	}

	cmd.Flags().IntVarP(&port, "port", "P", 8080, "Port to run the server on")

	cmd.Flags().StringVar(&s3Bucket, "s3-bucket", "", "S3 bucket name")
	cmd.Flags().StringVar(&s3Region, "s3-region", "us-east-1", "AWS region")
	cmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", "", "S3 endpoint URL")
	cmd.Flags().StringVar(&s3AccessKey, "s3-access-key", "", "AWS Access Key ID")
	cmd.Flags().StringVar(&s3SecretKey, "s3-secret-key", "", "AWS Secret Access Key")

	return cmd
}
