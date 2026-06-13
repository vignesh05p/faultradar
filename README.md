# FaultRadar

FaultRadar is a lightweight, read-only system diagnostic CLI tool for Linux/Ubuntu. It runs checks on disk space, log sizes, failed systemd services, kernel logs, and memory status, providing a clear health report to identify system bottlenecks and failures.

## Features

- **Root Disk Usage Check**: Monitors root filesystem (`/`) space using standard `df` logic and custom thresholds.
- **Log Files Size Check**: Recursively sums `/var/log` space and details the top 5 largest files.
- **Failed Systemd Services Check**: Scans for failed services while respecting a configurable ignore list.
- **Kernel Errors Check**: Counts kernel errors in the current boot and prints the first 5 samples.
- **Memory & Swap Check**: Reads `/proc/meminfo` to verify available memory levels and alerts if no swap space is configured.

## CLI Usage

### Doctor Command (Human-Readable Output)
Displays system health grouped by severity levels: `CRITICAL`, `WARNING`, `INFO`, `OK`, and `SKIPPED`.
```bash
faultradar doctor
```

### Doctor Command (JSON Output)
Outputs a deterministic JSON array of findings for integration and automation:
```bash
faultradar doctor --json
```

### Version Command
Prints the current tool version:
```bash
faultradar version
```

### Help Commands
```bash
faultradar help
faultradar doctor --help
```

## Configuration

FaultRadar loads settings in the following order:
1. `~/.config/faultradar/config.json`
2. `/etc/faultradar/config.json`
3. Built-in defaults

### Config Schema

Create a JSON file at `~/.config/faultradar/config.json` with the following structure to override default thresholds:

```json
{
  "disk": {
    "root_warning_percent": 85,
    "root_critical_percent": 95
  },
  "logs": {
    "varlog_warning_mb": 1024,
    "varlog_critical_mb": 5120
  },
  "kernel": {
    "max_error_lines_warning": 5,
    "max_error_lines_critical": 25
  },
  "systemd": {
    "ignore_units": [
      "bluetooth.service"
    ]
  },
  "memory": {
    "available_warning_percent": 15,
    "available_critical_percent": 5
  }
}
```

## Installation and Maintenance Scripts

All scripts are located in the `scripts/` directory:

- **Run Verification Suite**: Executes `go vet`, tests with coverage, and compiles a test binary.
  ```bash
  ./scripts/test.sh
  ```
- **Install Tool**: Compiles the binary, copies it to `/usr/local/bin`, and places the default config at `~/.config/faultradar/config.json`.
  ```bash
  ./scripts/install.sh
  ```
- **Uninstall Tool**: Removes the binary. Use `--purge` to also delete configuration files.
  ```bash
  ./scripts/uninstall.sh [--purge]
  ```
- **Build Release Binaries**: Compiles cross-platform `linux/amd64` and `linux/arm64` release binaries into the `bin/` directory.
  ```bash
  ./scripts/build-release.sh
  ```

## License

This project is licensed under the MIT License. See [LICENSE](LICENSE) for details.
