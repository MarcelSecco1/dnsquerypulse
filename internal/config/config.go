package config

import (
	"bytes"
	"fmt"
	"os"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

type Config struct {
	Resolvers []string `yaml:"resolvers" json:"resolvers"`
	Timeout   string   `yaml:"timeout" json:"timeout"`
	Format    string   `yaml:"format" json:"format"`
}

func Defaults() Config {
	return Config{Resolvers: []string{"1.1.1.1", "8.8.8.8"}, Timeout: "3s", Format: "table"}
}
func Load(path string) (Config, error) {
	c := Defaults()
	if path == "" {
		return c, nil
	}
	b, e := os.ReadFile(path)
	if e != nil {
		return c, fmt.Errorf("read config: %w", e)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if e = dec.Decode(&c); e != nil {
		return c, fmt.Errorf("decode config: %w", e)
	}
	if e = c.Validate(); e != nil {
		return c, e
	}
	return c, nil
}
func (c Config) Validate() error {
	if len(c.Resolvers) == 0 {
		return fmt.Errorf("at least one resolver is required")
	}
	if len(c.Resolvers) > 64 {
		return fmt.Errorf("no more than 64 resolvers may be configured")
	}
	for _, r := range c.Resolvers {
		if strings.TrimSpace(r) == "" {
			return fmt.Errorf("resolver cannot be empty")
		}
	}
	d, e := time.ParseDuration(c.Timeout)
	if e != nil || d <= 0 {
		return fmt.Errorf("timeout must be a positive duration such as 3s")
	}
	switch c.Format {
	case "table", "json", "csv":
	default:
		return fmt.Errorf("format must be table, json, or csv")
	}
	return nil
}
func (c Config) TimeoutDuration() time.Duration {
	d, e := time.ParseDuration(c.Timeout)
	if e != nil || d <= 0 {
		return 3 * time.Second
	}
	return d
}

const Example = `# dnsquerypulse configuration
resolvers:
  - 1.1.1.1
  - 8.8.8.8
timeout: 3s
format: table
`
