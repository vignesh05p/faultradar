# FaultRadar

FaultRadar is a read-only Linux diagnostic CLI. Run one command to see what is obviously wrong with your system and what safe command to run next.

## What FaultRadar is

- A single-command health check for common Linux desktop problems
- A structured report of disk, log, systemd, kernel, and memory issues
- A safe diagnostic tool that reads system files and runs read-only query commands

## What FaultRadar is NOT

- Not a daemon or background monitor
- Not an auto-fix or repair tool
- Not a log cleaner or space reclaimer
- Not a replacement for professional monitoring, backups, SMART tooling, or sysadmin judgment
- Not guaranteed to work on every Linux distribution

## Safety guarantee

FaultRadar is **read-only**. It does not delete, truncate, repair, restart, or modify your system. It may suggest safe inspection commands such as `df -h /`, `free -h`, or `systemctl status`.

## Install and build

```bash
git clone <repository-url>
cd faultradar
./scripts/build.sh
```

The binary is written to `bin/faultradar`.

Optional install to `/usr/local/bin`:

```bash
./scripts/install.sh
```

## Usage

```bash
faultradar doctor
faultradar doctor --json
faultradar version
faultradar help
faultradar doctor --help
```

Unsupported commands print usage and exit non-zero.

## Example human output

```text
FaultRadar v1.0.0

WARNING

[1] Large log storage detected
    /var/log uses 1.42 GB on disk.
    Suggestion: Inspect the largest logs and fix the source before deleting or truncating files.
    Check:
      sudo du -h -d 1 /var/log | sort -h
    Details:
      Actual disk usage: 1.42 GB
      Apparent size: 3.91 GB
      Largest directories:
        - /var/log/journal: 920.00 MB
        - /var/log/mongodb: 275.08 MB
      Largest files:
        - /var/log/mongodb/mongod.log: 275.08 MB
      Sparse files detected:
        - /var/log/lastlog appears sparse; apparent size may be misleading.

OK

[2] Root disk usage looks normal
    Root disk usage is 42%.
    Check:
      df -h /
    Details:
      Mount: /
      Used: 42%
```

## Example JSON output

```json
[
  {
    "id": "disk.root.usage",
    "severity": "ok",
    "title": "Root disk usage looks normal",
    "summary": "Root disk usage is 42%.",
    "check_command": "df -h /",
    "details": [
      "Mount: /",
      "Used: 42%"
    ]
  }
]
```

## Checks performed

1. **Root disk usage** — warns at 90% used, critical at 97%
2. **`/var/log` disk usage** — based on actual allocated blocks, not apparent file size
3. **Large log files and directories** — top 5 by actual disk usage
4. **Failed systemd units** — important services, normal services, snap mounts, and other units reported separately
5. **Kernel errors in current boot** — classified critical vs warning patterns
6. **Memory and swap health** — low available memory and missing swap
7. **Skipped or restricted checks** — reported when permissions or tools are unavailable

## Permissions

Some checks may be restricted for non-root users, especially kernel journal access and protected log files. FaultRadar reports restricted checks as skipped or warning instead of crashing.

## Limitations

- Tested primarily on Ubuntu-like Linux systems using systemd
- Requires `systemctl` and `journalctl` for full coverage; missing tools are reported as skipped
- Sparse log files (such as `lastlog`) can show misleading apparent sizes; thresholds use actual disk usage
- Snap mount failures are common noise and are reported separately from important service failures
- Config regex patterns that fail to compile produce a warning finding, not a crash

## Configuration

Optional config file locations (first found wins):

- `~/.config/faultradar/config.json`
- `/etc/faultradar/config.json`

See `examples/config.json` for the supported format. Defaults work without any config file.

## Development

```bash
./scripts/test.sh
./scripts/build.sh
./scripts/build-release.sh
```

`test.sh` runs gofmt, go vet, tests, and build.

## Exit codes

- `0` — only info, skipped, or ok findings
- `1` — at least one warning, no critical findings
- `2` — at least one critical finding

## License

See [LICENSE](LICENSE).
