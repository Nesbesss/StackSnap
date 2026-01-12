
package docker

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/api/types/container"
	"github.com/docker/docker/api/types/filters"
	"github.com/docker/docker/api/types/image"
	"github.com/docker/docker/api/types/mount"
	"github.com/docker/docker/api/types/system"
	"github.com/docker/docker/api/types/volume"
	"github.com/docker/docker/client"
	"github.com/docker/docker/pkg/stdcopy"
)


type Client struct {
	cli *client.Client
	ctx context.Context
}


func NewClient() (*Client, error) {
	cli, err := client.NewClientWithOpts(client.FromEnv, client.WithAPIVersionNegotiation())
	if err != nil {
		return nil, fmt.Errorf("failed to create Docker client: %w", err)
	}

	return &Client{
		cli: cli,
		ctx: context.Background(),
	}, nil
}


func (c *Client) Close() error {
	return c.cli.Close()
}


func (c *Client) Ping() error {
	_, err := c.cli.Ping(c.ctx)
	return err
}


func (c *Client) VolumeExists(name string) (bool, error) {
	_, err := c.cli.VolumeInspect(c.ctx, name)
	if err != nil {
		if client.IsErrNotFound(err) {
			return false, nil
		}
		return false, err
	}
	return true, nil
}


func (c *Client) ListVolumes(prefix string) ([]string, error) {
	opts := volume.ListOptions{}
	if prefix != "" {
		opts.Filters = filters.NewArgs(filters.Arg("name", prefix))
	}

	resp, err := c.cli.VolumeList(c.ctx, opts)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, v := range resp.Volumes {
		names = append(names, v.Name)
	}
	return names, nil
}


type ContainerInfo struct {
	ID           string
	Name         string
	Image        string
	State        string
	Health       string
	RestartCount int
	Uptime       string
	Error        string
	Paused       bool
	Labels       map[string]string
	Volumes      []string
}


func (c *Client) ListContainersUsingVolume(volumeName string) ([]ContainerInfo, error) {
	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{
		Filters: filters.NewArgs(filters.Arg("volume", volumeName)),
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list containers: %w", err)
	}

	var result []ContainerInfo
	for _, ctr := range containers {
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}
		result = append(result, ContainerInfo{
			ID:     ctr.ID,
			Name:   name,
			Image:  ctr.Image,
			State:  ctr.State,
			Paused: ctr.State == "paused",
		})
	}
	return result, nil
}


func (c *Client) ListAllContainers() ([]ContainerInfo, error) {
	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, fmt.Errorf("failed to list all containers: %w", err)
	}

	var result []ContainerInfo
	for _, ctr := range containers {
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
			if len(name) > 0 && name[0] == '/' {
				name = name[1:]
			}
		}

		health := "N/A"

		if ctr.State == "running" {
			inspect, err := c.cli.ContainerInspect(c.ctx, ctr.ID)
			if err == nil && inspect.State.Health != nil {
				health = string(inspect.State.Health.Status)
			}
		}

		result = append(result, ContainerInfo{
			ID:     ctr.ID,
			Name:   name,
			Image:  ctr.Image,
			State:  ctr.State,
			Health: health,
			Paused: ctr.State == "paused",
			Labels: ctr.Labels,
		})
	}
	return result, nil
}


func (c *Client) ListVolumesForProject(projectName string) ([]string, error) {

	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	volList, err := c.cli.VolumeList(c.ctx, volume.ListOptions{Filters: filters})
	if err != nil {
		return nil, fmt.Errorf("failed to list volumes for project %s: %w", projectName, err)
	}

	volumeMap := make(map[string]bool)
	for _, vol := range volList.Volumes {
		volumeMap[vol.Name] = true
	}



	prefix := projectName + "_"
	allVols, err := c.cli.VolumeList(c.ctx, volume.ListOptions{})
	if err == nil {
		for _, vol := range allVols.Volumes {
			if strings.HasPrefix(vol.Name, prefix) {
				volumeMap[vol.Name] = true
			}
		}
	}

	var volumes []string
	for name := range volumeMap {
		volumes = append(volumes, name)
	}
	return volumes, nil
}


func (c *Client) StopContainer(containerID string) error {
	return c.cli.ContainerStop(c.ctx, containerID, container.StopOptions{})
}


func (c *Client) StartContainer(containerID string) error {
	return c.cli.ContainerStart(c.ctx, containerID, container.StartOptions{})
}


func (c *Client) PauseContainer(containerID string) error {
	if err := c.cli.ContainerPause(c.ctx, containerID); err != nil {
		return fmt.Errorf("failed to pause container %s: %w", containerID, err)
	}
	return nil
}


func (c *Client) UnpauseContainer(containerID string) error {
	if err := c.cli.ContainerUnpause(c.ctx, containerID); err != nil {
		return fmt.Errorf("failed to unpause container %s: %w", containerID, err)
	}
	return nil
}


func (c *Client) InspectContainer(containerID string) (*types.ContainerJSON, error) {
	info, err := c.cli.ContainerInspect(c.ctx, containerID)
	if err != nil {
		return nil, fmt.Errorf("failed to inspect container %s: %w", containerID, err)
	}
	return &info, nil
}


func (c *Client) ListContainersForProject(projectName string) ([]ContainerInfo, error) {
	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("com.docker.compose.project=%s", projectName))

	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, err
	}

	var result []ContainerInfo
	for _, ctr := range containers {
		name := ""
		if len(ctr.Names) > 0 {
			name = ctr.Names[0]
			if name[0] == '/' {
				name = name[1:]
			}
		}

		var volumes []string
		for _, mount := range ctr.Mounts {
			if mount.Type == "volume" {
				volumes = append(volumes, mount.Name)
			}
		}

		result = append(result, ContainerInfo{
			ID:      ctr.ID,
			Name:    name,
			Image:   ctr.Image,
			State:   ctr.State,
			Labels:  ctr.Labels,
			Volumes: volumes,
		})
	}
	return result, nil
}


func (c *Client) ExecInContainer(containerID string, cmd []string) ([]byte, error) {
	execConfig := container.ExecOptions{
		Cmd:          cmd,
		AttachStdout: true,
		AttachStderr: true,
	}

	execResp, err := c.cli.ContainerExecCreate(c.ctx, containerID, execConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create exec: %w", err)
	}

	attachResp, err := c.cli.ContainerExecAttach(c.ctx, execResp.ID, container.ExecAttachOptions{})
	if err != nil {
		return nil, fmt.Errorf("failed to attach to exec: %w", err)
	}
	defer attachResp.Close()


	var stdoutBuf, stderrBuf []byte
	stdoutBuf, _ = io.ReadAll(attachResp.Reader)


	inspectResp, err := c.cli.ContainerExecInspect(c.ctx, execResp.ID)
	if err != nil {
		return stdoutBuf, nil
	}

	if inspectResp.ExitCode != 0 {
		return stdoutBuf, fmt.Errorf("command exited with code %d: %s", inspectResp.ExitCode, string(stderrBuf))
	}

	return stdoutBuf, nil
}


func (c *Client) ensureAlpine() error {
	_, _, err := c.cli.ImageInspectWithRaw(c.ctx, "alpine:latest")
	if err != nil {
		fmt.Println("Pulling alpine:latest image...")
		reader, err := c.cli.ImagePull(c.ctx, "alpine:latest", image.PullOptions{})
		if err != nil {
			return fmt.Errorf("failed to pull alpine image: %w", err)
		}
		io.Copy(io.Discard, reader)
		reader.Close()
	}
	return nil
}


func (c *Client) BackupVolume(volumeName string, w io.Writer) error {
	if err := c.ensureAlpine(); err != nil {
		return err
	}


	resp, err := c.cli.ContainerCreate(c.ctx, &container.Config{
		Image:        "alpine:latest",
		Cmd:          []string{"tar", "-cf", "-", "-C", "/volume", "."},
		AttachStdout: true,
		AttachStderr: true,
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/volume",
			},
		},
	}, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer c.cli.ContainerRemove(c.ctx, resp.ID, container.RemoveOptions{Force: true})


	attachResp, err := c.cli.ContainerAttach(c.ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdout: true,
		Stderr: true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}
	defer attachResp.Close()


	if err := c.cli.ContainerStart(c.ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}



	_, err = stdcopy.StdCopy(w, io.Discard, attachResp.Reader)
	if err != nil {
		return fmt.Errorf("failed to read backup stream: %w", err)
	}


	statusCh, errCh := c.cli.ContainerWait(c.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return fmt.Errorf("error waiting for container: %w", err)
	case status := <-statusCh:
		if status.StatusCode != 0 {
			return fmt.Errorf("backup command failed with exit code %d", status.StatusCode)
		}
	}

	return nil
}


func (c *Client) RestoreVolume(volumeName string, r io.Reader) error {
	if err := c.ensureAlpine(); err != nil {
		return err
	}


	resp, err := c.cli.ContainerCreate(c.ctx, &container.Config{
		Image:       "alpine:latest",
		Cmd:         []string{"tar", "-xf", "-", "-C", "/volume"},
		OpenStdin:   true,
		StdinOnce:   true,
		AttachStdin: true,
	}, &container.HostConfig{
		Mounts: []mount.Mount{
			{
				Type:   mount.TypeVolume,
				Source: volumeName,
				Target: "/volume",
			},
		},
	}, nil, nil, "")
	if err != nil {
		return fmt.Errorf("failed to create container: %w", err)
	}
	defer c.cli.ContainerRemove(c.ctx, resp.ID, container.RemoveOptions{Force: true})


	attachResp, err := c.cli.ContainerAttach(c.ctx, resp.ID, container.AttachOptions{
		Stream: true,
		Stdin:  true,
	})
	if err != nil {
		return fmt.Errorf("failed to attach to container: %w", err)
	}
	defer attachResp.Close()


	if err := c.cli.ContainerStart(c.ctx, resp.ID, container.StartOptions{}); err != nil {
		return fmt.Errorf("failed to start container: %w", err)
	}


	_, err = io.Copy(attachResp.Conn, r)
	if err != nil {
		return fmt.Errorf("failed to write restore data: %w", err)
	}
	attachResp.CloseWrite()


	statusCh, errCh := c.cli.ContainerWait(c.ctx, resp.ID, container.WaitConditionNotRunning)
	select {
	case err := <-errCh:
		return fmt.Errorf("error waiting for container: %w", err)
	case status := <-statusCh:
		if status.StatusCode != 0 {
			logs, _ := c.cli.ContainerLogs(c.ctx, resp.ID, container.LogsOptions{ShowStderr: true})
			if logs != nil {
				errOutput, _ := io.ReadAll(logs)
				logs.Close()
				return fmt.Errorf("restore failed with code %d: %s", status.StatusCode, string(errOutput))
			}
			return fmt.Errorf("restore failed with code %d", status.StatusCode)
		}
	}

	return nil
}


func (c *Client) ListComposeProjects() ([]string, error) {
	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true})
	if err != nil {
		return nil, err
	}

	projectDirs := make(map[string]bool)
	var dirs []string

	for _, ctr := range containers {

		if dir, ok := ctr.Labels["com.docker.compose.project.working_dir"]; ok {
			if !projectDirs[dir] {
				projectDirs[dir] = true
				dirs = append(dirs, dir)
			}
		}
	}
	return dirs, nil
}


func (c *Client) GetProjectStatus(workDir string) (string, error) {

	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("com.docker.compose.project.working_dir=%s", workDir))

	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true, Filters: filters})
	if err != nil {
		return "Unknown", err
	}

	if len(containers) == 0 {
		return "Stopped", nil
	}

	runningCount := 0
	for _, ctr := range containers {
		if ctr.State == "running" {
			runningCount++
		}
	}

	if runningCount == 0 {
		return "Stopped", nil
	}
	if runningCount == len(containers) {
		return "Running", nil
	}
	return "Partial", nil
}


func (c *Client) StopProjectContainers(workDir string) error {
	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("com.docker.compose.project.working_dir=%s", workDir))

	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true, Filters: filters})
	if err != nil {
		return err
	}

	for _, ctr := range containers {
		if ctr.State == "running" {
			if err := c.cli.ContainerStop(c.ctx, ctr.ID, container.StopOptions{}); err != nil {
				return fmt.Errorf("failed to stop container %s: %w", ctr.Names[0], err)
			}
		}
	}
	return nil
}


func (c *Client) StartProjectContainers(workDir string) error {
	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("com.docker.compose.project.working_dir=%s", workDir))

	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true, Filters: filters})
	if err != nil {
		return err
	}

	for _, ctr := range containers {
		if ctr.State != "running" {
			if err := c.cli.ContainerStart(c.ctx, ctr.ID, container.StartOptions{}); err != nil {
				return fmt.Errorf("failed to start container %s: %w", ctr.Names[0], err)
			}
		}
	}
	return nil
}


func (c *Client) GetContainerLogs(containerID string, tail int) (string, error) {
	options := container.LogsOptions{
		ShowStdout: true,
		ShowStderr: true,
		Tail:       fmt.Sprintf("%v", tail),
	}

	reader, err := c.cli.ContainerLogs(c.ctx, containerID, options)
	if err != nil {
		return "", err
	}
	defer reader.Close()

	var stdout, stderr bytes.Buffer
	_, _ = stdcopy.StdCopy(&stdout, &stderr, reader)

	return stdout.String() + stderr.String(), nil
}


func (c *Client) GetProjectDiagnostics(workDir string) (map[string]ContainerInfo, error) {
	filters := filters.NewArgs()
	filters.Add("label", fmt.Sprintf("com.docker.compose.project.working_dir=%s", workDir))

	containers, err := c.cli.ContainerList(c.ctx, container.ListOptions{All: true, Filters: filters})
	if err != nil {
		return nil, err
	}

	diagnostics := make(map[string]ContainerInfo)
	for _, ctr := range containers {
		inspect, err := c.cli.ContainerInspect(c.ctx, ctr.ID)
		if err != nil {
			continue
		}

		name := ctr.Names[0]
		if len(name) > 0 && name[0] == '/' {
			name = name[1:]
		}

		serviceName := ctr.Labels["com.docker.compose.service"]

		health := "N/A"
		if inspect.State.Health != nil {
			health = inspect.State.Health.Status
		}

		diagnostics[serviceName] = ContainerInfo{
			ID:           ctr.ID,
			Name:         name,
			Image:        ctr.Image,
			State:        inspect.State.Status,
			Health:       health,
			RestartCount: inspect.RestartCount,
			Uptime:       inspect.State.StartedAt,
			Error:        inspect.State.Error,
			Paused:       inspect.State.Paused,
		}
	}
	return diagnostics, nil
}


func (c *Client) CommitContainer(containerID string, ref string) (string, error) {
	resp, err := c.cli.ContainerCommit(c.ctx, containerID, container.CommitOptions{
		Reference: ref,
		Comment:   "Created by StackSnap Backup",
	})
	if err != nil {
		return "", fmt.Errorf("failed to commit container: %w", err)
	}
	return resp.ID, nil
}


func (c *Client) SaveImage(imageID string, outputPath string) error {
	reader, err := c.cli.ImageSave(c.ctx, []string{imageID})
	if err != nil {
		return fmt.Errorf("failed to save image: %w", err)
	}
	defer reader.Close()

	file, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer file.Close()

	if _, err := io.Copy(file, reader); err != nil {
		return fmt.Errorf("failed to write image to file: %w", err)
	}

	return nil
}


func (c *Client) LoadImage(inputPath string) error {
	file, err := os.Open(inputPath)
	if err != nil {
		return fmt.Errorf("failed to open image file: %w", err)
	}
	defer file.Close()

	resp, err := c.cli.ImageLoad(c.ctx, file, true)
	if err != nil {
		return fmt.Errorf("failed to load image: %w", err)
	}
	defer resp.Body.Close()


	io.Copy(io.Discard, resp.Body)

	return nil
}


func (c *Client) TagImage(imageID, targetRef string) error {
	return c.cli.ImageTag(c.ctx, imageID, targetRef)
}


func (c *Client) RemoveImage(imageID string) error {
	_, err := c.cli.ImageRemove(c.ctx, imageID, image.RemoveOptions{Force: true})
	return err
}



func (c *Client) Info() (system.Info, error) {
	return c.cli.Info(c.ctx)
}


func (c *Client) DiskUsage() (types.DiskUsage, error) {
	return c.cli.DiskUsage(c.ctx, types.DiskUsageOptions{})
}
