# GitHub Runner Prometheus Exporter

`github-runner-prometheus-exporter` exposes Prometheus metrics for a self-hosted GitHub Actions runner by reading local runner state and logs on the runner host itself.

This project is currently optimized for one runner per exporter process. It is useful when GitHub's UI and APIs are not enough for host-level observability, runner utilization, and workflow timing on self-hosted runners.

## What It Does

- Exposes runner busy/idle state
- Exposes workflow timing metrics from `Worker_*.log`
- Exposes lightweight event metadata from `event.json`
- Exposes process metrics for the exporter itself
- Exposes disk usage for `/` and `/tmp`

## Current Model

- One exporter process monitors one runner
- The runner is selected from config
- If multiple runners are configured, only one should be enabled, or `RUNNER_NAME` must be set
- The exporter is expected to run on the same machine as the GitHub runner

## How It Works

The exporter combines two runner-local sources:

1. `event.json`
   - Used for busy/idle state
   - Used for repository / workflow event metadata
   - This file is ephemeral and may only exist while a job is active

2. `Worker_*.log`
   - Used for workflow start time
   - Used for workflow end time
   - Used for workflow duration
   - Used for `run_id`, `repository`, `repository_owner`, and `workflow`

In addition, Prometheus `ProcessCollector` is registered for exporter process metrics, and a custom collector reports disk usage for `/` and `/tmp`.

## Metrics

All metrics are exported with common runner labels through the registry wrapper:

- `hostname`
- `os`
- `runner_name`
- `runner_group`
- any custom labels configured under `runners[].labels`

Prometheus will also attach scrape labels such as `job` and `instance`.

### Runner Metrics

| Metric | Extra Labels | Description |
|---|---|---|
| `github_runner_state` | `state` | Runner state where `busy=1` or `idle=1` |

### Workflow Metrics

| Metric | Extra Labels | Description |
|---|---|---|
| `github_workflow_start_timestamp_seconds` | `run_id`, `repository`, `repository_owner`, `workflow` | Start time of the latest parsed workflow run |
| `github_workflow_end_timestamp_seconds` | `run_id`, `repository`, `repository_owner`, `workflow` | End time of the latest parsed workflow run |
| `github_workflow_duration_seconds` | `run_id`, `repository`, `repository_owner`, `workflow` | Duration of the latest parsed workflow run |

### Event Metrics

| Metric | Extra Labels | Description |
|---|---|---|
| `github_event_triggered_total` | `repository`, `repository_owner`, `workflow` | Workflow events observed while `event.json` exists |
| `github_event_triggered_timestamp_seconds` | `repository`, `repository_owner`, `workflow` | Last observed event timestamp from `event.json` |

### Host / Exporter Metrics

| Metric | Extra Labels | Description |
|---|---|---|
| `disk_usage_bytes` | `mount`, `type` | Disk usage for `/` and `/tmp` with `type=total|used|free|used_percent` |
| `process_cpu_seconds_total` | none | Exporter CPU time |
| `process_resident_memory_bytes` | none | Exporter RSS memory |
| `process_open_fds` | none | Exporter open file descriptors |
| `process_network_receive_bytes_total` | none | Exporter process receive bytes |
| `process_network_transmit_bytes_total` | none | Exporter process transmit bytes |
| `process_start_time_seconds` | none | Exporter start time |
| `process_virtual_memory_bytes` | none | Exporter virtual memory |
| `process_virtual_memory_max_bytes` | none | Exporter virtual memory limit |
| `process_max_fds` | none | Exporter max file descriptors |

## Known Limitations

- Workflow metrics are currently derived from the latest available worker log parsing model, not from a perfect run-event model
- If a run produces multiple worker log files, multiple time series for the same `run_id` may exist historically
- `event.json` is ephemeral, so event-based metrics are best treated as runtime/context signals rather than durable historical accounting
- Multi-runner support in a single exporter process is not implemented

## Configuration

The exporter reads `github-runner.yaml` from the working directory or `/etc/github-runner-exporter/`.

Example:

```yaml
server:
  listen_address: ":9200"

runners:
  - name: hetzner-github-runner-11
    group: Default
    enable: true
    mode: prod
    labels:
      provider: hetzner

    logs:
      worker: /opt/actions-runner/_diag
      event: /opt/actions-runner/_work/_temp/_github_workflow/event.json

    metrics:
      enable_job: true
      enable_event: true
```

### Important Notes

- `logs.worker` must point to the directory containing `Worker_*.log`
- `logs.event` must point to the runner's `event.json` path
- The `event.json` parent directory may not exist while the runner is idle
- If multiple runners are present in config, enable only one or set `RUNNER_NAME`

## Build

From the repository root:

```bash
go build -o github-runner-prometheus-exporter ./cmd/exporter
```

This produces:

```bash
./github-runner-prometheus-exporter
```

If you install it globally:

```bash
sudo mv github-runner-prometheus-exporter /usr/local/bin/github-runner-prometheus-exporter
```

## Run

```bash
./github-runner-prometheus-exporter
```

The exporter serves:

- `/metrics`
- `/health`

By default it listens on `:9200`.

## Prometheus Configuration

Example scrape config:

```yaml
- job_name: github-runner-exporter
  static_configs:
    - targets:
        - 127.0.0.1:9200
```

## Grafana

A dashboard JSON is provided at:

- [grafana/github-runner-exporter.json](/home/francesco/clones/github-runner-prometheus-exporter/grafana/github-runner-exporter.json)

The dashboard is built around the current exporter metrics and expects:

- `job="github-runner-exporter"` to exist in Prometheus
- workflow metrics under `github_workflow_*`
- repository filtering via the `repository` label

Recommended import flow:

1. Import the dashboard JSON as a new dashboard
2. Select the Prometheus datasource during import
3. Set `Job = github-runner-exporter`
4. Narrow by `Runner Group`, `Provider`, `Hostname`, `Workflow`, or `Repository` as needed

## Testing

A simple way to exercise the runner is to add a manual GitHub Actions workflow using `workflow_dispatch` and let it:

- print `GITHUB_EVENT_PATH`
- create temporary disk activity under `/tmp`
- sleep for a few minutes so Prometheus can observe busy state

Useful PromQL checks during testing:

```promql
github_runner_state{job="github-runner-exporter"}
```

```promql
github_workflow_duration_seconds{job="github-runner-exporter"}
```

```promql
disk_usage_bytes{job="github-runner-exporter", mount="/", type="used_percent"}
```

```promql
process_resident_memory_bytes{job="github-runner-exporter"}
```

## Why Use This Instead of Only GitHub

GitHub gives control-plane visibility:

- runner online/offline status
- workflow and job history
- queue and execution status

This exporter adds data-plane visibility on the runner host:

- local busy/idle state
- workflow timing from runner logs
- exporter process health
- disk usage on the runner host

## Status

This project is still under active iteration. The metric contract and dashboard are now aligned with the current implementation, but workflow parsing and event modeling are still being refined.
