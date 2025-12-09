package tools

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/mparvin/octaai/pkg/config"
	"golang.org/x/crypto/ssh"
)

// SSHTool provides SSH operations for remote servers
type SSHTool struct {
	cfg *config.Config
}

// NewSSHTool creates a new SSH tool
func NewSSHTool(cfg *config.Config) *SSHTool {
	return &SSHTool{cfg: cfg}
}

// Name implements Tool.Name
func (t *SSHTool) Name() string {
	return "ssh"
}

// Schema implements Tool.Schema
func (t *SSHTool) Schema() ToolSchema {
	return ToolSchema{
		Name:        "ssh",
		Description: "Execute commands on remote servers via SSH",
		Parameters: map[string]ParamSchema{
			"action": {
				Type:        "string",
				Description: "SSH action to perform",
				Required:    true,
				Enum:        []string{"exec", "upload", "download"},
			},
			"host": {
				Type:        "string",
				Description: "Remote host address",
				Required:    true,
			},
			"port": {
				Type:        "number",
				Description: "SSH port (default: 22)",
				Required:    false,
			},
			"user": {
				Type:        "string",
				Description: "SSH username",
				Required:    true,
			},
			"password": {
				Type:        "string",
				Description: "SSH password (if not using key)",
				Required:    false,
			},
			"key_path": {
				Type:        "string",
				Description: "Path to SSH private key",
				Required:    false,
			},
			"command": {
				Type:        "string",
				Description: "Command to execute (for exec action)",
				Required:    false,
			},
			"local_path": {
				Type:        "string",
				Description: "Local file path (for upload/download)",
				Required:    false,
			},
			"remote_path": {
				Type:        "string",
				Description: "Remote file path (for upload/download)",
				Required:    false,
			},
		},
	}
}

// Execute implements Tool.Execute
func (t *SSHTool) Execute(ctx context.Context, args map[string]interface{}) (*ToolResult, error) {
	action, ok := args["action"].(string)
	if !ok {
		return nil, fmt.Errorf("action is required")
	}

	host, ok := args["host"].(string)
	if !ok {
		return nil, fmt.Errorf("host is required")
	}

	user, ok := args["user"].(string)
	if !ok {
		return nil, fmt.Errorf("user is required")
	}

	port := 22
	if portVal, ok := args["port"].(float64); ok {
		port = int(portVal)
	}

	// Create SSH client
	client, err := t.createSSHClient(host, port, user, args)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer client.Close()

	switch action {
	case "exec":
		command, _ := args["command"].(string)
		return t.execCommand(client, command)
	case "upload":
		localPath, _ := args["local_path"].(string)
		remotePath, _ := args["remote_path"].(string)
		return t.uploadFile(client, localPath, remotePath)
	case "download":
		remotePath, _ := args["remote_path"].(string)
		localPath, _ := args["local_path"].(string)
		return t.downloadFile(client, remotePath, localPath)
	default:
		return nil, fmt.Errorf("unknown action: %s", action)
	}
}

func (t *SSHTool) createSSHClient(host string, port int, user string, args map[string]interface{}) (*ssh.Client, error) {
	var authMethods []ssh.AuthMethod

	// Try password authentication
	if password, ok := args["password"].(string); ok && password != "" {
		authMethods = append(authMethods, ssh.Password(password))
	}

	// Try key-based authentication
	keyPath, ok := args["key_path"].(string)
	if !ok || keyPath == "" {
		keyPath = t.cfg.SSH.DefaultKeyPath
	}

	if keyPath != "" {
		key, err := os.ReadFile(keyPath)
		if err == nil {
			signer, err := ssh.ParsePrivateKey(key)
			if err == nil {
				authMethods = append(authMethods, ssh.PublicKeys(signer))
			}
		}
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method available")
	}

	config := &ssh.ClientConfig{
		User:            user,
		Auth:            authMethods,
		HostKeyCallback: ssh.InsecureIgnoreHostKey(), // TODO: Implement proper host key checking
		Timeout:         30 * time.Second,
	}

	addr := fmt.Sprintf("%s:%d", host, port)
	client, err := ssh.Dial("tcp", addr, config)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	return client, nil
}

func (t *SSHTool) execCommand(client *ssh.Client, command string) (*ToolResult, error) {
	if command == "" {
		return nil, fmt.Errorf("command is required")
	}

	session, err := client.NewSession()
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer session.Close()

	output, err := session.CombinedOutput(command)

	success := err == nil
	errorMsg := ""
	if err != nil {
		errorMsg = err.Error()
	}

	return &ToolResult{
		Success: success,
		Output:  string(output),
		Error:   errorMsg,
		Data: map[string]interface{}{
			"command": command,
		},
	}, nil
}

func (t *SSHTool) uploadFile(client *ssh.Client, localPath, remotePath string) (*ToolResult, error) {
	if localPath == "" || remotePath == "" {
		return nil, fmt.Errorf("local_path and remote_path are required")
	}

	// Read local file
	content, err := os.ReadFile(localPath)
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Create remote directory if needed
	remoteDir := filepath.Dir(remotePath)
	mkdirCmd := fmt.Sprintf("mkdir -p %s", remoteDir)

	session, err := client.NewSession()
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	session.Run(mkdirCmd)
	session.Close()

	// Upload file using SCP protocol
	session, err = client.NewSession()
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer session.Close()

	// Simple approach: write content via shell redirection
	uploadCmd := fmt.Sprintf("cat > %s", remotePath)
	stdin, err := session.StdinPipe()
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	if err := session.Start(uploadCmd); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	stdin.Write(content)
	stdin.Close()

	if err := session.Wait(); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Uploaded %d bytes to %s", len(content), remotePath),
	}, nil
}

func (t *SSHTool) downloadFile(client *ssh.Client, remotePath, localPath string) (*ToolResult, error) {
	if localPath == "" || remotePath == "" {
		return nil, fmt.Errorf("local_path and remote_path are required")
	}

	session, err := client.NewSession()
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}
	defer session.Close()

	// Read remote file content
	content, err := session.Output(fmt.Sprintf("cat %s", remotePath))
	if err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Ensure local directory exists
	localDir := filepath.Dir(localPath)
	if err := os.MkdirAll(localDir, 0755); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	// Write to local file
	if err := os.WriteFile(localPath, content, 0644); err != nil {
		return &ToolResult{
			Success: false,
			Error:   err.Error(),
		}, nil
	}

	return &ToolResult{
		Success: true,
		Output:  fmt.Sprintf("Downloaded %d bytes to %s", len(content), localPath),
	}, nil
}
