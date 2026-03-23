package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

type Config struct {
	Server  Server   `mapstructure:"server"`
	Runners []Runner `mapstructure:"runners"`
}

type Server struct {
	ListenAddress string `mapstructure:"listen_address"`
}

type Runner struct {
	Name   string            `mapstructure:"name"`
	Group  string            `mapstructure:"group"`
	Enable bool              `mapstructure:"enable"`
	Mode   string            `mapstructure:"mode"` // prod / test
	Labels map[string]string `mapstructure:"labels"`

	Logs struct {
		Runner string `mapstructure:"runner"`
		Worker string `mapstructure:"worker"`
		Event  string `mapstructure:"event"`
	} `mapstructure:"logs"`

	Test struct {
		RunnerPath string `mapstructure:"runner_path"`
		EventPath  string `mapstructure:"event_path"`
		WorkerPath string `mapstructure:"worker_path"`
	} `mapstructure:"test"`

	Metrics struct {
		EnableRunner bool `mapstructure:"enable_runner"`
		EnableJob    bool `mapstructure:"enable_job"`
		EnableEvent  bool `mapstructure:"enable_event"`
	} `mapstructure:"metrics"`
}

func Load() (*Config, error) {
	v := viper.New()

	v.SetConfigName("github-runner") // github-runner.yaml
	v.SetConfigType("yaml")
	v.AddConfigPath(".")
	v.AddConfigPath("/etc/github-runner-exporter/")
	v.AutomaticEnv()

	// Defaults
	v.SetDefault("server.listen_address", "9200")
	v.SetDefault("mode", "prod")

	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unable to decode config into struct: %w", err)
	}

	cfg.Normalize()

	return &cfg, nil
}

func (c *Config) Normalize() {
	addr := strings.TrimSpace(c.Server.ListenAddress)
	switch {
	case addr == "":
		c.Server.ListenAddress = ":9200"
	case strings.HasPrefix(addr, ":"):
		c.Server.ListenAddress = addr
	case !strings.Contains(addr, ":"):
		c.Server.ListenAddress = ":" + addr
	default:
		c.Server.ListenAddress = addr
	}
}

func (c *Config) SelectedRunner() (*Runner, error) {
	if len(c.Runners) == 0 {
		return nil, fmt.Errorf("no runners configured")
	}

	if runnerName := strings.TrimSpace(os.Getenv("RUNNER_NAME")); runnerName != "" {
		for i := range c.Runners {
			if c.Runners[i].Name == runnerName {
				if !c.Runners[i].Enable {
					return nil, fmt.Errorf("runner %q is disabled", runnerName)
				}
				return &c.Runners[i], nil
			}
		}
		return nil, fmt.Errorf("runner %q not found in config", runnerName)
	}

	var selected *Runner
	for i := range c.Runners {
		if !c.Runners[i].Enable {
			continue
		}
		if selected != nil {
			return nil, fmt.Errorf("multiple enabled runners found; set RUNNER_NAME or enable only one runner")
		}
		selected = &c.Runners[i]
	}

	if selected != nil {
		return selected, nil
	}

	if len(c.Runners) == 1 {
		return &c.Runners[0], nil
	}

	return nil, fmt.Errorf("no enabled runner found; set RUNNER_NAME or enable one runner")
}
