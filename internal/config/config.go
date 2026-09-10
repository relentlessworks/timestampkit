package config

import (
	"crypto/rand"
	"encoding/hex"
	"flag"
	"fmt"
	"os"
)

// Config holds all service configuration.
type Config struct {
	Addr   string
	Secret string
}

// Load reads configuration from defaults, env vars, and flags.
// Priority: defaults < env vars < flags.
func Load() *Config {
	c := &Config{
		Addr:   ":7290",
		Secret: "",
	}

	if v := os.Getenv("TIMESTAMPKIT_ADDR"); v != "" {
		c.Addr = v
	}
	if v := os.Getenv("TIMESTAMPKIT_SECRET"); v != "" {
		c.Secret = v
	}

	flag.StringVar(&c.Addr, "addr", c.Addr, "listen address")
	flag.StringVar(&c.Secret, "secret", c.Secret, "token signing secret (auto-generated if empty)")
	flag.Parse()

	if c.Secret == "" {
		b := make([]byte, 32)
		rand.Read(b)
		c.Secret = hex.EncodeToString(b)
	}

	return c
}

func (c *Config) String() string {
	return fmt.Sprintf("addr=%s secret=%d-char", c.Addr, len(c.Secret))
}
