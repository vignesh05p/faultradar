# FaultRadar

FaultRadar is a lightweight, read-only system diagnostic CLI tool for Linux/Ubuntu. It runs checks on disk space, log sizes, failed systemd services, kernel logs, and memory status, providing a clear health report to identify system bottlenecks and failures.

## What FaultRadar is
- A fast, zero-dependency command line tool to quickly inspect the health of a Linux system.
- An accurate classifier for kernel errors and systemd failures, reducing noise from non-critical messages (e.g. ACPI BIOS warnings, snap mounts).
- A safe diagnostic tool that reads system files and runs safe query commands.

## What FaultRadar is NOT
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
Parses failed units via `systemctl --failed` and groups them into categories: `service`, `mount`, `snap mount`, `timer`, `socket`, and `other`.
- Filters out non-critical failures like `snap-*.mount` from upgrading severity levels to critical.
- Promotes failures of **important services** (e.g. `mysql`, `postgresql`, `docker`, `ssh`) to **CRITICAL**.
- Supports exact ignoring of service names and glob-style ignore patterns (`snap-*.mount`).

### 3. Log Analysis
Walks `/var/log` recursively to calculate total apparent size and reports:
- Top 5 largest directories and top 5 largest files.
- Warnings if sparse log files like `lastlog`, `btmp`, or `wtmp` might misrepresent actual disk space.
- Configurable warning and critical thresholds (default: warning = 1024 MB, critical = 5120 MB).

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
FaultRadar v1.0.0

WARNING

[1] Large log files detected
    Total log size is 3941.23 MB (threshold: 1024 MB).
    Suggestion: Inspect the largest logs and fix the source before deleting or truncating files.
    Check:
      find /var/log -type f -exec du -sh {} +
    Details:
      Total apparent size of /var/log: 3941.23 MB
      Largest directories:
        - /var/log/journal: 3432.00 MB
        - /var/log/mongodb: 275.08 MB
      Largest files:
        - /var/log/mongodb/mongod.log: 275.08 MB
      Note: lastlog, btmp, and wtmp may be sparse or misleading. Use du -h for disk usage.

OK

[2] Root disk usage looks normal
    Root disk usage is 42%.
    Check:
      df -h /
    Details:
      Root filesystem is 42% used.
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
    "summary": "Total log size is 3941.23 MB (threshold: 1024 MB).",
    "suggestion": "Inspect the largest logs and fix the source before deleting or truncating files.",
    "check_command": "find /var/log -type f -exec du -sh {} +",
    "details": [
      "Total apparent size of /var/log: 3941.23 MB",
      "Largest directories:",
      "  - /var/log/journal: 3432.00 MB",
      "  - /var/log/mongodb: 275.08 MB",
      "Largest files:",
      "  - /var/log/mongodb/mongod.log: 275.08 MB"
    ]
  }
]
```

---

## Known Limitations

- **Sparse File Sizing**: Apparent sizing might report extremely large values for sparse log files (such as `/var/log/lastlog`). A notice is automatically included in reports when sparse logs are present.
- **Root Permissions**: Running without root permissions might prevent the tool from parsing kernel logs via `journalctl` or walking restricted directories in `/var/log`.
- **Systemd Dependency**: The systemd checks require the `systemctl` CLI tool to be installed and available in the system PATH.
