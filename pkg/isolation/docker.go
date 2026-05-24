package isolation

import (
	"fmt"
	"os/exec"
	"strings"

	"github.com/mparvin/octaai/pkg/config"
)

// Docker wraps shell commands in a Docker sandbox.
type Docker struct {
	cfg *config.Config
}

// NewDocker creates a Docker isolator from config.
func NewDocker(cfg *config.Config) *Docker {
	return &Docker{cfg: cfg}
}

// Available reports whether the docker CLI is present and isolation is enabled.
func (d *Docker) Available() bool {
	if !d.cfg.Isolation.Enabled || !d.cfg.Isolation.Docker.Enabled {
		return false
	}
	_, err := exec.LookPath("docker")
	return err == nil
}

// ShouldIsolate reports whether a tool should run in Docker.
func (d *Docker) ShouldIsolate(toolName string) bool {
	if !d.Available() {
		return false
	}
	for _, t := range d.cfg.Isolation.RequireDockerFor {
		if t == toolName {
			return true
		}
	}
	return false
}

// WrapCommand builds a docker run invocation for a shell command.
func (d *Docker) WrapCommand(cwd, command string) ([]string, error) {
	if command == "" {
		return nil, fmt.Errorf("empty command")
	}

	dc := d.cfg.Isolation.Docker
	mount := dc.WorkdirMount
	if mount == "" {
		mount = d.cfg.ProjectsRoot
	}

	containerWorkdir := "/workspace"
	if cwd != "" {
		containerWorkdir = "/workspace/" + strings.TrimPrefix(strings.TrimPrefix(cwd, mount), "/")
	}

	args := []string{"run", "--rm"}
	if dc.Network != "" {
		args = append(args, "--network", dc.Network)
	}
	if dc.MemoryLimit != "" {
		args = append(args, "--memory", dc.MemoryLimit)
	}
	if dc.CPULimit != "" {
		args = append(args, "--cpus", dc.CPULimit)
	}
	if dc.ReadOnlyRoot {
		args = append(args, "--read-only", "--tmpfs", "/tmp:rw,noexec,nosuid")
	}
	args = append(args, "-v", fmt.Sprintf("%s:/workspace:rw", mount))
	args = append(args, "-w", containerWorkdir)
	args = append(args, dc.ExtraArgs...)
	args = append(args, dc.Image, "sh", "-c", command)

	return append([]string{"docker"}, args...), nil
}

// WrapArgs rewrites command tool args to execute inside Docker when applicable.
func (d *Docker) WrapArgs(toolName string, args map[string]interface{}) (map[string]interface{}, bool, error) {
	if !d.ShouldIsolate(toolName) {
		return args, false, nil
	}

	cwd, _ := args["cwd"].(string)
	cmdStr, _ := args["command"].(string)
	if cmdStr == "" {
		return args, false, fmt.Errorf("command is required for docker isolation")
	}

	dockerCmd, err := d.WrapCommand(cwd, cmdStr)
	if err != nil {
		return nil, false, err
	}

	wrapped := make(map[string]interface{}, len(args))
	for k, v := range args {
		wrapped[k] = v
	}
	wrapped["command"] = strings.Join(dockerCmd, " ")
	wrapped["cwd"] = d.cfg.ProjectsRoot
	wrapped["_isolated"] = true

	return wrapped, true, nil
}
