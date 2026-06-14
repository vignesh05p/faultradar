# FaultRadar

FaultRadar is a lightweight, read-only system diagnostic CLI tool for Linux/Ubuntu. It runs checks on disk space, log sizes, failed systemd services, kernel logs, and memory status, providing a clear health report to identify system bottlenecks and failures.

## What FaultRadar is
- An early diagnostic CLI to quickly inspect the health of a Linux system.
- An accurate classifier for kernel errors and systemd failures, reducing noise from non-critical messages (e.g. ACPI BIOS warnings, snap mounts).
- A safe diagnostic tool that reads system files and runs safe query commands.

## What FaultRadar is NOT
- It is **not** a production-ready stable public release (it is currently v0.2.0).
- It is **not** a daemon or a background monitor.
- It is **not** a system repair or configuration auto-fixing tool.
- It is **not** a log cleaner or space reclaimer.

## Why it does not auto-fix problems
FaultRadar adheres strictly to a **read-only diagnostic philosophy**. Automatically fixing system errors (such as deleting massive logs, restarting services, or modifying mount points) can cause unexpected downtime, data corruption, or service failures. Diagnosing the underlying cause and applying deliberate manual actions is safer and prevents repeat issues.

---

## Installation & Script Commands

- **Run Verification Suite (gofmt, vet, test, build)**:
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

---

## Core Diagnostics & Features

### 1. Kernel Error Classification
Navigates the output of `journalctl -k -p 3 -b --no-pager` and classifies errors:
- **CRITICAL**: Significant issues (e.g., `I/O error`, `EXT4-fs error`, `OOM killer`, `kernel panic`, hardware/MCE faults).
- **WARNING**: Milder messages (e.g., `ACPI Error`, `Can't lookup blockdev`, missing non-essential firmware).
- **INFO**: Low count of unrecognized error lines (below the configured threshold).
- **OK**: No priority-3 kernel errors detected.

### 2. Systemd Grouping & Snap Noise Handling
Parses failed units via `systemctl --failed` and separates findings to avoid noisy alerts:
- **Failed systemd services check**: Reports failures of standard services, timers, and sockets. If an important service (e.g., `mysql`, `postgresql`, `docker`, `ssh`, `NetworkManager`) fails, severity is promoted to **CRITICAL**.
- **Failed snap mount units check**: Failed snap mounts are grouped and reported separately as **WARNING** unless configuration says otherwise.
- Human output is formatted to only list a few snap unit examples and a count to keep the output readable.

### 3. Log Analysis
Walks `/var/log` recursively to calculate actual disk usage and apparent size:
- Warning and critical thresholds are based on actual disk usage, preventing false alerts on sparse files.
- Identifies sparse log files (such as `/var/log/lastlog`) and prints a notice to clarify size discrepancies.
- Reports the top 5 largest directories and top 5 largest files by actual disk usage.

---

## Configuration Example

Save this to `~/.config/faultradar/config.json` or `/etc/faultradar/config.json` to customize thresholds:

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
    "unknown_error_warning_count": 10,
    "ignore_patterns": [
      "some-noise-pattern"
    ],
    "downgrade_patterns": [
      "some-acpi-issue"
    ]
  },
  "systemd": {
    "ignore_units": [
      "bluetooth.service"
    ],
    "ignore_unit_patterns": [
      "snap-*.mount"
    ],
    "important_units": [
      "mysql.service",
      "postgresql.service",
      "docker.service",
      "ssh.service"
    ]
  },
  "memory": {
    "available_warning_percent": 15,
    "available_critical_percent": 5
  }
}
```

---

## CLI Output Examples

### Human-Readable Output
```text
FaultRadar v0.2.0

WARNING

[1] Large log files detected
    Total actual log size is 1080.50 MB (threshold: 1024 MB).
    Suggestion: Inspect the largest logs and fix the source before deleting or truncating files.
    Check:
      find /var/log -type f -exec du -sh {} +
    Details:
      Total actual size of /var/log: 1080.50 MB (apparent size: 3941.23 MB)
      Largest directories:
        - /var/log/journal: 950.00 MB
        - /var/log/mongodb: 130.50 MB
      Largest files:
        - /var/log/journal/123/system.journal: 950.00 MB
        - /var/log/lastlog: 0.10 MB (apparent: 2860.73 MB)
      Note: lastlog, btmp, and wtmp may be sparse or misleading. Use du -h for disk usage.

[2] Failed snap mount units found
    3 failed snap mount unit(s) detected.
    Suggestion: These may be temporary or noisy snap environment issues. Check snapd status if persistent.
    Check:
      systemctl --failed --no-pager --plain
    Details:
      Failed snap mount units:
        - snap-chromium-3235.mount
        - snap-chromium-3265.mount
        - snap-code-215.mount

OK

[3] Root disk usage looks normal
    Root disk usage is 42%.
    Check:
      df -h /
    Details:
      Root filesystem is 42% used.

[4] No failed systemd services found
    All services and system units are running normally.
    Check:
      systemctl --failed --no-pager --plain
```

### JSON Output
```json
[
  {
    "id": "disk.root.usage",
    "severity": "ok",
    "title": "Root disk usage looks normal",
    "summary": "Root disk usage is 42%.",
    "check_command": "df -h /",
    "details": [
      "Root filesystem is 42% used."
    ]
  },
  {
    "id": "logs.varlog.size",
    "severity": "warning",
    "title": "Large log files detected",
    "summary": "Total actual log size is 1080.50 MB (threshold: 1024 MB).",
    "suggestion": "Inspect the largest logs and fix the source before deleting or truncating files.",
    "check_command": "find /var/log -type f -exec du -sh {} +",
    "details": [
      "Total actual size of /var/log: 1080.50 MB (apparent size: 3941.23 MB)",
      "Largest directories:",
      "  - /var/log/journal: 950.00 MB",
      "  - /var/log/mongodb: 130.50 MB",
      "Largest files:",
      "  - /var/log/journal/123/system.journal: 950.00 MB",
      "  - /var/log/lastlog: 0.10 MB (apparent: 2860.73 MB)"
    ]
  }
]
```

---

## Known Limitations

- **Platform Scope**: Built specifically for system diagnostics on Linux/Ubuntu.
- **Root Permissions**: Some checks may be restricted for non-root users, especially kernel journal access and protected log files.
- **Systemd Dependency**: The systemd checks require the `systemctl` CLI tool to be installed and available in the system PATH.
