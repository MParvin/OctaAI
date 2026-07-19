package config

import (
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"log"
)

// EnsureBrowserToken generates a random WebSocket token when browser automation is enabled.
func EnsureBrowserToken(cfg *Config) (string, error) {
	if cfg.Browser.Token != "" {
		return cfg.Browser.Token, nil
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("failed to generate browser token: %w", err)
	}
	cfg.Browser.Token = hex.EncodeToString(buf)
	log.Printf("Generated browser WebSocket token (%d hex chars); set browser.token in config and the Firefox extension (value not logged)", len(cfg.Browser.Token))
	return cfg.Browser.Token, nil
}
