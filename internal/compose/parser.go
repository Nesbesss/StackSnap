
package compose

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)


type ComposeFile struct {
	Version  string                `yaml:"version"`
	Services map[string]Service    `yaml:"services"`
	Volumes  map[string]VolumeSpec `yaml:"volumes"`
	Secrets  map[string]SecretSpec `yaml:"secrets"`
}


type Service struct {
	Image       string        `yaml:"image"`
	Volumes     []string      `yaml:"volumes"`
	Environment []string      `yaml:"environment"`
	EnvFile     interface{}   `yaml:"env_file"`
	Secrets     []interface{} `yaml:"secrets"`
}


type VolumeSpec struct {
	Driver     string            `yaml:"driver"`
	DriverOpts map[string]string `yaml:"driver_opts"`
	External   bool              `yaml:"external"`
}


type SecretSpec struct {
	File     string `yaml:"file"`
	External bool   `yaml:"external"`
}


type VolumeMount struct {
	Source      string
	Target      string
	IsNamed     bool
	ServiceName string
}


type ServiceDiagnostics struct {
	ContainerID     string `json:"container_id"`
	State           string `json:"state"`
	Health          string `json:"health"`
	RestartCount    int    `json:"restart_count"`
	Uptime          string `json:"uptime"`
	DiagnosticError string `json:"diagnostic_error"`
	Paused          bool   `json:"paused"`
}


type Stack struct {
	Name         string
	Status       string
	ComposeFile  string
	Services     map[string]ServiceDiagnostics
	VolumeMounts []VolumeMount
	NamedVolumes []string
	EnvFiles     []string
	SecretFiles  []string
	BuildFiles   []string
	IsStandalone bool
}


func FindComposeFile(dir string) (string, error) {
	candidates := []string{
		"docker-compose.yml",
		"docker-compose.yaml",
		"compose.yml",
		"compose.yaml",
	}

	for _, name := range candidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			return path, nil
		}
	}

	return "", fmt.Errorf("no docker-compose file found in %s", dir)
}


func Parse(path string) (*ComposeFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read compose file: %w", err)
	}

	var compose ComposeFile
	if err := yaml.Unmarshal(data, &compose); err != nil {
		return nil, fmt.Errorf("failed to parse compose file: %w", err)
	}

	return &compose, nil
}


func DiscoverStack(dir string) (*Stack, error) {
	composePath, err := FindComposeFile(dir)
	if err != nil {
		return nil, err
	}

	compose, err := Parse(composePath)
	if err != nil {
		return nil, err
	}


	absDir, _ := filepath.Abs(dir)
	projectName := filepath.Base(absDir)

	stack := &Stack{
		Name:        projectName,
		ComposeFile: composePath,
	}


	stack.Services = make(map[string]ServiceDiagnostics)
	for serviceName, service := range compose.Services {
		stack.Services[serviceName] = ServiceDiagnostics{
			State: "Unknown",
		}

		for _, vol := range service.Volumes {
			mount := parseVolumeMount(vol, serviceName, compose.Volumes)
			stack.VolumeMounts = append(stack.VolumeMounts, mount)
		}
	}


	for volName := range compose.Volumes {
		fullName := fmt.Sprintf("%s_%s", projectName, volName)
		stack.NamedVolumes = append(stack.NamedVolumes, fullName)
	}


	envs := []string{".env"}
	for _, svc := range compose.Services {
		switch v := svc.EnvFile.(type) {
		case string:
			envs = append(envs, v)
		case []interface{}:
			for _, item := range v {
				if s, ok := item.(string); ok {
					envs = append(envs, s)
				}
			}
		}
	}


	seenEnv := make(map[string]bool)
	for _, env := range envs {
		path := filepath.Join(dir, env)
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			if !seenEnv[abs] {
				stack.EnvFiles = append(stack.EnvFiles, path)
				seenEnv[abs] = true
			}
		}
	}


	for _, sec := range compose.Secrets {
		if sec.File != "" && !sec.External {
			path := filepath.Join(dir, sec.File)
			if _, err := os.Stat(path); err == nil {
				abs, _ := filepath.Abs(path)
				stack.SecretFiles = append(stack.SecretFiles, abs)
			}
		}
	}


	buildCandidates := []string{"Dockerfile", "Dockerfile.dev", "Dockerfile.prod", ".dockerignore"}
	for _, name := range buildCandidates {
		path := filepath.Join(dir, name)
		if _, err := os.Stat(path); err == nil {
			abs, _ := filepath.Abs(path)
			stack.BuildFiles = append(stack.BuildFiles, abs)
		}
	}

	return stack, nil
}


func parseVolumeMount(mount string, serviceName string, namedVolumes map[string]VolumeSpec) VolumeMount {

	var source, target string


	colonIdx := -1
	for i := 0; i < len(mount); i++ {
		if mount[i] == ':' {

			if i == 1 && len(mount) > 2 && mount[2] == '\\' {
				continue
			}
			colonIdx = i
			break
		}
	}

	if colonIdx == -1 {

		return VolumeMount{
			Source:      mount,
			Target:      mount,
			IsNamed:     true,
			ServiceName: serviceName,
		}
	}

	source = mount[:colonIdx]
	target = mount[colonIdx+1:]


	if idx := findLastColon(target); idx != -1 {
		target = target[:idx]
	}


	isNamed := true
	if len(source) > 0 && (source[0] == '/' || source[0] == '.' || source[0] == '~') {
		isNamed = false
	}

	return VolumeMount{
		Source:      source,
		Target:      target,
		IsNamed:     isNamed,
		ServiceName: serviceName,
	}
}

func findLastColon(s string) int {
	for i := len(s) - 1; i >= 0; i-- {
		if s[i] == ':' {
			return i
		}
	}
	return -1
}
