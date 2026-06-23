// Package config resolves layer rules, root rules, and ignore globs from a YAML
// file (PRD §11). Roots and layers are reused from archview where possible.
package config

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

// Load reads a config file, falling back to Default when path is empty.
//
// TODO(M2): parse YAML (gopkg.in/yaml.v3). For now returns Default.
func Load(path string) (Config, error) {
	if path == "" {
		return Default(), nil
	}
	return Default(), nil
}
