package exporter

import (
	"os"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/collectors"

	"github.com/thineshsubramani/github-runner-prometheus-exporter/collector"
	"github.com/thineshsubramani/github-runner-prometheus-exporter/config"
	"github.com/thineshsubramani/github-runner-prometheus-exporter/internal/platform"
)

type Exporter struct {
	Registry *prometheus.Registry
}

func New(cfg *config.Config) (*Exporter, error) {
	reg := prometheus.NewRegistry()
	runner, err := cfg.SelectedRunner()
	if err != nil {
		return nil, err
	}

	hostname, _ := os.Hostname()
	labels := prometheus.Labels{
		"hostname": hostname,
		"os":       platform.GetOS(),
	}

	// add custom labels from config
	if runner.Labels != nil {
		for k, v := range runner.Labels {
			labels[k] = v
		}
	}

	wrappedReg := prometheus.WrapRegistererWith(labels, reg)

	runner_name := runner.Name
	group_name := runner.Group
	runnerWrappedReg := prometheus.WrapRegistererWith(
		prometheus.Labels{"runner_name": runner_name, "runner_group": group_name},
		wrappedReg,
	)
	// reg.MustRegister(collectors.NewGoCollector())

	runnerWrappedReg.MustRegister(collectors.NewProcessCollector(collectors.ProcessCollectorOpts{}))
	// wrappedReg.MustRegister(collector.NewInfoCollector(cfg))
	runnerWrappedReg.MustRegister(collector.NewDiskCollector())
	// Custom collectors
	if runner.Metrics.EnableJob && runner.Logs.Worker != "" {
		runnerWrappedReg.MustRegister(collector.NewWorkerCollector(runner.Logs.Worker))
	}
	if runner.Metrics.EnableEvent {
		runnerWrappedReg.MustRegister(collector.NewEventCollector(cfg))
	}

	return &Exporter{Registry: reg}, nil
}
