package isolation

import (
	"fmt"
	"os/exec"
	"path/filepath"
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

// SplitCommand splits a command string into argv without invoking a shell.
func SplitCommand(command string) ([]string, error) {
	parts := strings.Fields(command)
	if len(parts) == 0 {
		return nil, fmt.Errorf("empty command")
	}
	return parts, nil
}

// WrapCommand builds a docker run invocation argv for a command.
// The command is executed as argv (image binary + args), never via `sh -c`.
func (d *Docker) WrapCommand(cwd, command string) ([]string, error) {
	cmdParts, err := SplitCommand(command)
	if err != nil {
		return nil, err
	}

	dc := d.cfg.Isolation.Docker
	mount := dc.WorkdirMount
	if mount == "" {
		mount = d.cfg.ProjectsRoot
	}

	containerWorkdir := "/workspace"
	if cwd != "" {
		resolved := config.ResolveProjectPath(d.cfg, cwd)
		if rel, err := filepath.Rel(mount, resolved); err == nil && rel != "." && !strings.HasPrefix(rel, "..") {
			containerWorkdir = "/workspace/" + filepath.ToSlash(rel)
		}
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
	// Entrypoint is the binary; remaining tokens are args — no shell.
	args = append(args, dc.Image)
	args = append(args, cmdParts...)

	return append([]string{"docker"}, args...), nil
}

// WrapArgs annotates command tool args for Docker isolation without destroying argv boundaries.
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

	wrapped := make(map[string]interface{}, len(args)+2)
	for k, v := range args {
		wrapped[k] = v
	}
	wrapped["_docker_argv"] = dockerCmd
	wrapped["_isolated"] = true

	return wrapped, true, nil
}
