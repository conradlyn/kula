# Terminal UI (TUI)

Kula includes a terminal dashboard for when you're on a server over SSH and don't want to open
a browser. It's built with [Bubble Tea](https://github.com/charmbracelet/bubbletea) and
[Lipgloss](https://github.com/charmbracelet/lipgloss).

## Launch

```bash
./kula tui
```

The TUI runs **independently** of the `serve` daemon — it spins up its own collector and
samples the system directly. You don't need `kula serve` running to use it. (It does not read
from or write to the storage tier files.)

## Tabs

The TUI presents seven views; navigate between them with the keyboard:

| Tab | Shows |
|-----|-------|
| **Overview** | A condensed summary of all key metrics |
| **CPU** | Per-aspect CPU usage and load averages |
| **Memory** | Memory and swap breakdown |
| **Network** | Per-interface throughput and TCP/socket stats |
| **Disk** | Per-device I/O and filesystem usage |
| **Processes** | Running / sleeping / blocked / zombie counts and threads |
| **GPU** | GPU load, power, VRAM, temperature |

Each view uses progress bars and a responsive layout that adapts to your terminal size. The
theme is a dark purple/slate palette.

## Refresh rate

The refresh interval is set separately from the collection interval:

```yaml
tui:
  refresh_rate: 1s
```

## Quitting

Press `q` or `Ctrl+C` to exit. Application monitoring (Postgres, MySQL, nginx, Apache2,
containers) is started for the TUI session as well, so those collectors initialize on launch.

## System info

The TUI banner shows OS, kernel, and architecture unless `global.show_system_info` is `false`,
in which case those fields are hidden.

Next: [Authentication](07-authentication.md).
