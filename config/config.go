package config

import (
	"fmt"
	"gopkg.in/yaml.v2"
	"io/ioutil"
)

// Example config: See ./example/config.yml

func LoadConfigFile(filename string) (*Config, error) {
	content, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, fmt.Errorf("Failed to load %v: %v", filename, err.Error())
	}
	cfg, err := LoadConfigString(content)
	if err != nil {
		return nil, fmt.Errorf("Failed to load %v: %v", filename, err.Error())
	}
	return cfg, nil
}

func LoadConfigString(content []byte) (*Config, error) {
	cfg := &Config{}
	err := yaml.Unmarshal(content, cfg)
	if err != nil {
		return nil, err
	}
	cfg.setDefaults()
	err = cfg.validate()
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func (cfg *Config) String() string {
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return fmt.Sprintf("ERROR: Failed to marshal config: %v", err.Error())
	}
	return string(out)
}

type InputConfig struct {
	Type    string `yaml:",omitempty"`
	Path    string `yaml:",omitempty"`
	Readall bool   `yaml:",omitempty"`
}

type GrokConfig struct {
	PatternsDir string   `yaml:"patterns_dir,omitempty"`
	Patterns    []string `yaml:",omitempty"`
}

type Label struct {
	GrokFieldName   string `yaml:"grok_field_name,omitempty"`
	PrometheusLabel string `yaml:"prometheus_label,omitempty"`
}

type MetricConfig struct {
	Type   string  `yaml:",omitempty"`
	Name   string  `yaml:",omitempty"`
	Help   string  `yaml:",omitempty"`
	Match  string  `yaml:",omitempty"`
	Labels []Label `yaml:",omitempty"`
}

type MetricsConfig []*MetricConfig

type ServerConfig struct {
	Protocol string `yaml:",omitempty"`
	Port     int    `yaml:",omitempty"`
	Cert     string `yaml:",omitempty"`
	Key      string `yaml:",omitempty"`
}

type Config struct {
	Input   *InputConfig   `yaml:",omitempty"`
	Grok    *GrokConfig    `yaml:",omitempty"`
	Metrics *MetricsConfig `yaml:",omitempty"`
	Server  *ServerConfig  `yaml:",omitempty"`
}

func (cfg *Config) setDefaults() {
	if cfg.Input == nil {
		cfg.Input = &InputConfig{}
	}
	cfg.Input.setDefaults()
	if cfg.Grok == nil {
		cfg.Grok = &GrokConfig{}
	}
	cfg.Grok.setDefaults()
	if cfg.Metrics == nil {
		metrics := MetricsConfig(make([]*MetricConfig, 0))
		cfg.Metrics = &metrics
	}
	cfg.Metrics.setDefaults()
	if cfg.Server == nil {
		cfg.Server = &ServerConfig{}
	}
	cfg.Server.setDefaults()
}

func (c *InputConfig) setDefaults() {
	if c.Type == "" {
		c.Type = "stdin"
	}
}

func (c *GrokConfig) setDefaults() {}

func (c *MetricsConfig) setDefaults() {}

func (c *ServerConfig) setDefaults() {
	if c.Protocol == "" {
		c.Protocol = "http"
	}
	if c.Port == 0 {
		c.Port = 9144
	}
}

func (cfg *Config) validate() error {
	err := cfg.Input.validate()
	if err != nil {
		return err
	}
	err = cfg.Grok.validate()
	if err != nil {
		return err
	}
	err = cfg.Metrics.validate()
	if err != nil {
		return err
	}
	err = cfg.Server.validate()
	if err != nil {
		return err
	}
	return nil
}

func (c *InputConfig) validate() error {
	switch {
	case c.Type == "stdin":
		if c.Path != "" {
			return fmt.Errorf("Cannot use 'input.path' when 'input.type' is stdin.")
		}
	case c.Type == "file":
		if c.Path == "" {
			return fmt.Errorf("'input.path' is required for input type \"file\".")
		}
	default:
		return fmt.Errorf("Unsupported 'input.type': %v", c.Type)
	}
	return nil
}

func (c *GrokConfig) validate() error {
	if c.PatternsDir == "" && len(c.Patterns) == 0 {
		return fmt.Errorf("No patterns defined: One of 'grok.patterns_dir' and 'grok.patterns' must be configured.")
	}
	return nil
}

func (c *MetricsConfig) validate() error {
	if len(*c) == 0 {
		return fmt.Errorf("'metrics' must not be empty.")
	}
	metricNames := make(map[string]bool)
	for _, metric := range *c {
		_, exists := metricNames[metric.Name]
		if exists {
			return fmt.Errorf("%v defined twice.", metric.Name)
		}
		metricNames[metric.Name] = true
		err := metric.validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (c *MetricConfig) validate() error {
	switch {
	case c.Type != "counter":
		return fmt.Errorf("Invalid 'metrics.type': '%v'. We currently only support 'counter'.", c.Type)
	case c.Name == "":
		return fmt.Errorf("'metrics.name' must not be empty.")
	case c.Help == "":
		return fmt.Errorf("'metrics.help' must not be empty.")
	case c.Match == "":
		return fmt.Errorf("'metrics.match' must not be empty.")
	}
	if c.Labels == nil {
		return fmt.Errorf("Cannot find 'metrics.label' configuration.")
	}
	for _, label := range c.Labels {
		err := label.validate()
		if err != nil {
			return err
		}
	}
	return nil
}

func (l *Label) validate() error {
	switch {
	case l.GrokFieldName == "":
		return fmt.Errorf("'metrics.label.grok_field_name' must not be empty.")
	case l.PrometheusLabel == "":
		return fmt.Errorf("'metrics.label.prometheus_label' must not be empty.")
	default:
		return nil
	}
}

func (c *ServerConfig) validate() error {
	switch {
	case c.Protocol != "https" && c.Protocol != "http":
		return fmt.Errorf("Invalid 'server.protocol': '%v'. Expecting 'http' or 'https'.", c.Protocol)
	case c.Port <= 0:
		return fmt.Errorf("Invalid 'server.port': '%v'.", c.Port)
	case c.Protocol == "https":
		if c.Cert != "" && c.Key == "" {
			return fmt.Errorf("'server.cert' must not be specified without 'server.key'")
		}
		if c.Cert == "" && c.Key != "" {
			return fmt.Errorf("'server.key' must not be specified without 'server.cert'")
		}
	case c.Protocol == "http":
		if c.Cert != "" || c.Key != "" {
			return fmt.Errorf("'server.cert' and 'server.key' can only be configured for protocol 'https'.")
		}
	}
	return nil
}
