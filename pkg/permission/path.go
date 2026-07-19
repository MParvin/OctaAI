package permission

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/mparvin/octaai/pkg/config"
)

// ResolveSafePath expands, absolutes, and (when possible) resolves symlinks for a path.
// For paths that do not exist yet, it resolves the deepest existing ancestor and
// rejoins the remaining components so symlink escapes via parents are detected.
func ResolveSafePath(cfg *config.Config, path string) (string, error) {
	if path == "" {
		return "", fmt.Errorf("empty path")
	}
	full := config.ResolveProjectPath(cfg, path)
	abs, err := filepath.Abs(full)
	if err != nil {
		return "", fmt.Errorf("invalid path %q: %w", path, err)
	}
	return evalPath(abs)
}

func evalPath(abs string) (string, error) {
	resolved, err := filepath.EvalSymlinks(abs)
	if err == nil {
		return resolved, nil
	}
	if !os.IsNotExist(err) {
		// If the leaf does not exist, walk up; other errors are fatal.
		if !isNotExist(err) {
			return "", fmt.Errorf("resolve path %q: %w", abs, err)
		}
	}

	// Walk up until an existing ancestor is found, then rejoin the tail.
	cur := abs
	var missing []string
	for {
		resolved, err := filepath.EvalSymlinks(cur)
		if err == nil {
			if len(missing) == 0 {
				return resolved, nil
			}
			parts := append([]string{resolved}, reverse(missing)...)
			return filepath.Join(parts...), nil
		}
		if cur == filepath.Dir(cur) {
			return "", fmt.Errorf("resolve path %q: %w", abs, err)
		}
		missing = append(missing, filepath.Base(cur))
		cur = filepath.Dir(cur)
	}
}

func isNotExist(err error) bool {
	return os.IsNotExist(err) || strings.Contains(strings.ToLower(err.Error()), "no such file")
}

func reverse(in []string) []string {
	out := make([]string, len(in))
	for i := range in {
		out[i] = in[len(in)-1-i]
	}
	return out
}

// PathAllowed reports whether absPath is within one of the allowed roots.
// Both path and roots are evaluated with symlink resolution when possible.
func PathAllowed(cfg *config.Config, absPath string) bool {
	if cfg == nil || len(cfg.Safety.AllowPaths) == 0 {
		return false
	}
	candidate, err := evalPath(absPath)
	if err != nil {
		candidate = absPath
	}
	candidate = filepath.Clean(candidate)

	for _, allowed := range cfg.Safety.AllowPaths {
		allowedAbs, err := filepath.Abs(allowed)
		if err != nil {
			continue
		}
		allowedResolved, err := evalPath(allowedAbs)
		if err != nil {
			allowedResolved = filepath.Clean(allowedAbs)
		} else {
			allowedResolved = filepath.Clean(allowedResolved)
		}
		if candidate == allowedResolved || strings.HasPrefix(candidate, allowedResolved+string(filepath.Separator)) {
			return true
		}
	}
	return false
}
