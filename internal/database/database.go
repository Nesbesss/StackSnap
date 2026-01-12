
package database

import (
	"bytes"
	"fmt"
	"io"
	"strings"

	"github.com/stacksnap/stacksnap/internal/docker"
)


type DatabaseType string

const (
	DatabasePostgres DatabaseType = "postgres"
	DatabaseMySQL  DatabaseType = "mysql"
	DatabaseMongo  DatabaseType = "mongodb"
	DatabaseUnknown DatabaseType = "unknown"
)


type DatabaseInfo struct {
	ContainerID  string
	ContainerName string
	Type     DatabaseType
	Image     string
}


func DetectDatabase(client *docker.Client, containerID string) (*DatabaseInfo, error) {
	info, err := client.InspectContainer(containerID)
	if err != nil {
		return nil, err
	}

	imageLower := strings.ToLower(info.Config.Image)

	var dbType DatabaseType
	switch {
	case strings.Contains(imageLower, "postgres"):
		dbType = DatabasePostgres
	case strings.Contains(imageLower, "mysql") || strings.Contains(imageLower, "mariadb"):
		dbType = DatabaseMySQL
	case strings.Contains(imageLower, "mongo"):
		dbType = DatabaseMongo
	default:
		dbType = DatabaseUnknown
	}

	name := info.Name
	if len(name) > 0 && name[0] == '/' {
		name = name[1:]
	}

	return &DatabaseInfo{
		ContainerID:  containerID,
		ContainerName: name,
		Type:     dbType,
		Image:     info.Config.Image,
	}, nil
}


func DumpPostgres(client *docker.Client, containerID string) (io.Reader, error) {


	output, err := client.ExecInContainer(containerID, []string{
		"pg_dumpall",
		"-U", "postgres",
		"--serializable-deferrable",
	})
	if err != nil {

		output, err = client.ExecInContainer(containerID, []string{
			"pg_dumpall", "-U", "postgres",
		})
		if err != nil {

			output, err = client.ExecInContainer(containerID, []string{"pg_dumpall"})
			if err != nil {
				return nil, fmt.Errorf("failed to dump postgres: %w", err)
			}
		}
	}

	return bytes.NewReader(output), nil
}


func DumpMySQL(client *docker.Client, containerID string) (io.Reader, error) {


	cmd := []string{
		"sh", "-c",
		`mysqldump --all-databases \
			--single-transaction \
			--quick \
			--routines \
			--triggers \
			--events \
			-u root -p"${MYSQL_ROOT_PASSWORD:-${MYSQL_PWD}}"`,
	}
	output, err := client.ExecInContainer(containerID, cmd)
	if err != nil {

		cmd = []string{
			"mysqldump",
			"--all-databases",
			"--single-transaction",
			"--quick",
			"--routines",
			"--triggers",
			"--events",
			"-u", "root",
		}
		output, err = client.ExecInContainer(containerID, cmd)
		if err != nil {
			return nil, fmt.Errorf("failed to dump mysql: %w", err)
		}
	}

	return bytes.NewReader(output), nil
}


func DumpMongo(client *docker.Client, containerID string) (io.Reader, error) {

	output, err := client.ExecInContainer(containerID, []string{
		"mongodump", "--archive",
	})
	if err != nil {
		return nil, fmt.Errorf("failed to dump mongodb: %w", err)
	}

	return bytes.NewReader(output), nil
}


func Dump(client *docker.Client, dbInfo *DatabaseInfo) (io.Reader, error) {
	switch dbInfo.Type {
	case DatabasePostgres:
		return DumpPostgres(client, dbInfo.ContainerID)
	case DatabaseMySQL:
		return DumpMySQL(client, dbInfo.ContainerID)
	case DatabaseMongo:
		return DumpMongo(client, dbInfo.ContainerID)
	default:
		return nil, fmt.Errorf("unsupported database type: %s", dbInfo.Type)
	}
}
