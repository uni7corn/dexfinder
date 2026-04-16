// Package config handles .dexfinder.yaml configuration file loading.
package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
)

// Config holds configuration loaded from .dexfinder.yaml.
type Config struct {
	DexFile    string `yaml:"dex-file"`
	Query      string `yaml:"query"`
	Format     string `yaml:"format"`
	Layout     string `yaml:"layout"`
	Style      string `yaml:"style"`
	Mapping    string `yaml:"mapping"`
	ShowObf    bool   `yaml:"show-obf"`
	ApiFlags   string `yaml:"api-flags"`
	ClassFilter string `yaml:"class-filter"`
	ExcludeApiLists string `yaml:"exclude-api-lists"`
	Trace      bool   `yaml:"trace"`
	Depth      int    `yaml:"depth"`
	Scope      string `yaml:"scope"`
	Color      string `yaml:"color"`
	FailOn     string `yaml:"fail-on"`
	Output     string `yaml:"output"`
}

// Load searches for .dexfinder.yaml in the current directory and parent
// directories up to the filesystem root. Returns an empty Config if no
// file is found.
func Load() *Config {
	path := findConfigFile()
	if path == "" {
		return &Config{}
	}
	cfg, err := parseConfigFile(path)
	if err != nil {
		return &Config{}
	}
	return cfg
}

// LoadFromFile loads config from a specific path.
func LoadFromFile(path string) (*Config, error) {
	return parseConfigFile(path)
}

func findConfigFile() string {
	dir, err := os.Getwd()
	if err != nil {
		return ""
	}
	for {
		candidate := filepath.Join(dir, ".dexfinder.yaml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		candidate = filepath.Join(dir, ".dexfinder.yml")
		if _, err := os.Stat(candidate); err == nil {
			return candidate
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			break
		}
		dir = parent
	}
	return ""
}

// parseConfigFile parses a simple YAML-like config file.
// We implement a minimal parser to avoid external dependencies.
// Supports: key: value (string), key: true/false (bool), key: 123 (int)
func parseConfigFile(path string) (*Config, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	cfg := &Config{}
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx == -1 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		// Remove surrounding quotes
		val = strings.Trim(val, "\"'")

		switch key {
		case "dex-file":
			cfg.DexFile = val
		case "query":
			cfg.Query = val
		case "format":
			cfg.Format = val
		case "layout":
			cfg.Layout = val
		case "style":
			cfg.Style = val
		case "mapping":
			cfg.Mapping = val
		case "show-obf":
			cfg.ShowObf = parseBool(val)
		case "api-flags":
			cfg.ApiFlags = val
		case "class-filter":
			cfg.ClassFilter = val
		case "exclude-api-lists":
			cfg.ExcludeApiLists = val
		case "trace":
			cfg.Trace = parseBool(val)
		case "depth":
			cfg.Depth = parseInt(val)
		case "scope":
			cfg.Scope = val
		case "color":
			cfg.Color = val
		case "fail-on":
			cfg.FailOn = val
		case "output":
			cfg.Output = val
		}
	}
	return cfg, scanner.Err()
}

func parseBool(s string) bool {
	s = strings.ToLower(s)
	return s == "true" || s == "yes" || s == "1"
}

func parseInt(s string) int {
	n := 0
	for _, c := range s {
		if c >= '0' && c <= '9' {
			n = n*10 + int(c-'0')
		} else {
			break
		}
	}
	return n
}

// ApplyDefaults sets flag values from config, but only for flags that
// weren't explicitly set on the command line.
// Returns a function to call after flag.Parse() that applies defaults.
// flagSet maps flag name → whether it was explicitly set by the user.
func (c *Config) ApplyToFlags(flagSet map[string]bool, flags map[string]*string, boolFlags map[string]*bool, intFlags map[string]*int) {
	setStr := func(name, cfgVal string) {
		if cfgVal != "" && !flagSet[name] {
			if p, ok := flags[name]; ok {
				*p = cfgVal
			}
		}
	}
	setBool := func(name string, cfgVal bool) {
		if cfgVal && !flagSet[name] {
			if p, ok := boolFlags[name]; ok {
				*p = cfgVal
			}
		}
	}
	setInt := func(name string, cfgVal int) {
		if cfgVal > 0 && !flagSet[name] {
			if p, ok := intFlags[name]; ok {
				*p = cfgVal
			}
		}
	}

	setStr("dex-file", c.DexFile)
	setStr("query", c.Query)
	setStr("format", c.Format)
	setStr("layout", c.Layout)
	setStr("style", c.Style)
	setStr("mapping", c.Mapping)
	setStr("api-flags", c.ApiFlags)
	setStr("class-filter", c.ClassFilter)
	setStr("exclude-api-lists", c.ExcludeApiLists)
	setStr("scope", c.Scope)
	setStr("color", c.Color)
	setStr("fail-on", c.FailOn)
	setStr("output", c.Output)
	setBool("show-obf", c.ShowObf)
	setBool("trace", c.Trace)
	setInt("depth", c.Depth)
}
