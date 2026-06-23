// Package config resolves layer rules, root rules, and ignore globs from a YAML
// file (PRD §11). Roots and layers are reused from archview where possible.
package config

import (
	"os"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

// Roots declares what counts as an entrypoint for the reachability pass.
type Roots struct {
	Routes      bool     `yaml:"routes"`       // registered HTTP routes are roots
	Main        bool     `yaml:"main"`         // func main is a root
	ExportedAPI bool     `yaml:"exported_api"` // exported package API are roots (library modules)
	Keep        []string `yaml:"keep"`         // allowlist for reflective/known entrypoints
}

// Config is the parsed arch-diff.yaml.
type Config struct {
	Layers map[string]string `yaml:"layers"` // layer name -> path glob
	Roots  Roots             `yaml:"roots"`
	Ignore []string          `yaml:"ignore"` // globs excluded from dead set and roots
}

// Default returns a config matching the PRD example.
func Default() Config {
	return Config{
		Layers: map[string]string{
			"handler": "internal/**/handler",
			"service": "internal/**/service",
			"repo":    "internal/**/repo",
			"domain":  "internal/domain/**",
			"infra":   "internal/infra/**",
		},
		Roots:  Roots{Routes: true, Main: true},
		Ignore: []string{"**/*_test.go", "**/mock_*.go", "vendor/**"},
	}
}

// Load reads a config file, falling back to Default when path is empty. Any
// field the file omits keeps its Default value.
func Load(path string) (Config, error) {
	cfg := Default()
	if path == "" {
		return cfg, nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return cfg, err
	}
	// Start empty so the file fully controls the maps/slices it sets.
	var fromFile Config
	if err := yaml.Unmarshal(data, &fromFile); err != nil {
		return cfg, err
	}
	if fromFile.Layers != nil {
		cfg.Layers = fromFile.Layers
	}
	if fromFile.Ignore != nil {
		cfg.Ignore = fromFile.Ignore
	}
	cfg.Roots = fromFile.Roots
	return cfg, nil
}

// MatchGlob reports whether a doublestar-style glob matches a slash-separated
// path. `**` spans path separators, `*` and `?` do not.
func MatchGlob(pattern, path string) bool {
	re, ok := globCache[pattern]
	if !ok {
		re = compileGlob(pattern)
		globCache[pattern] = re
	}
	return re.MatchString(path)
}

var globCache = map[string]*regexp.Regexp{}

func compileGlob(pattern string) *regexp.Regexp {
	var b strings.Builder
	b.WriteString("^")
	for i := 0; i < len(pattern); i++ {
		c := pattern[i]
		switch c {
		case '*':
			if i+1 < len(pattern) && pattern[i+1] == '*' {
				b.WriteString(".*") // ** : across separators
				i++
				if i+1 < len(pattern) && pattern[i+1] == '/' {
					i++ // swallow the slash so "**/x" also matches "x"
				}
			} else {
				b.WriteString("[^/]*")
			}
		case '?':
			b.WriteString("[^/]")
		case '.', '+', '(', ')', '|', '[', ']', '{', '}', '^', '$', '\\':
			b.WriteByte('\\')
			b.WriteByte(c)
		default:
			b.WriteByte(c)
		}
	}
	b.WriteString("$")
	re, err := regexp.Compile(b.String())
	if err != nil {
		return regexp.MustCompile(`$^`) // never matches
	}
	return re
}
