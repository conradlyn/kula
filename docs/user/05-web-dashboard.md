# Web Dashboard

The dashboard is a single-page application embedded directly in the Kula binary. It is built
on Chart.js with custom SVG gauges, connects over WebSocket for live updates, and falls back
to the history REST API for longer time ranges.

Open it at `http://localhost:27960` (or your configured address). If `web.ui` is `false`, the
dashboard is disabled and only `/metrics` and `/health` remain.

## Layout

- **Header** — hostname, system info (OS / kernel / arch), Kula version, theme toggle,
  language selector, and (when enabled) the AI assistant 🤖 button and the Space Invaders
  easter-egg button.
- **Gauges** — at-a-glance circular gauges for the headline metrics (CPU, memory, etc.).
- **Chart cards** — one card per metric group: CPU, Load, Memory, Swap, Network, Disk I/O,
  Filesystems, Thermals, GPU, Processes, Battery, and any enabled Applications / Custom
  metrics.

## Features

### Time range & history

By default the dashboard streams live 1-second data over WebSocket. Selecting a longer time
window switches to the history API, which serves downsampled data from the appropriate
storage tier (1-minute or 5-minute aggregates for older data).

You can pick a **preset window** (1m … 30d) or a **custom range** with explicit from/to
timestamps.

### Shareable URLs

The selected time window, custom range, and aggregation are reflected in the URL query string,
so any view can be bookmarked or shared. On load, Kula restores the view from these parameters:

- `range=<seconds>` — a preset window (e.g. `?range=3600` for the last hour). Only the values
  exposed by the preset buttons are accepted.
- `from=<ISO>&to=<ISO>` — a custom range (RFC 3339 timestamps); takes precedence over `range`.
- `agg=avg|min|max` — the [aggregation](#aggregation-selector) mode. A value here overrides
  both the saved local preference and `web.default_aggregation` for that visit. It is only
  added to the URL for windows of **3 hours or longer** (shorter windows show raw data, where
  aggregation does not apply) and only when it differs from the configured default.

Example: `http://localhost:27960/?range=86400&agg=max` opens the last 24 hours showing
per-window maxima.

### Interactive zoom

Drag-select on any chart to zoom into a time window. Zooming **auto-pauses** the live stream
so the view stays put while you inspect. Resume to return to live.

### Focus mode

Hide everything except the charts you care about. Useful when investigating a specific
subsystem.

### Y-axis bounds

Charts can auto-scale to data peaks, or you can impose fixed maxima. For CPU temperature, disk
temperature, and network, the bound mode (`off` / `on` / `auto`) is set in
[configuration](04-configuration.md#web) under `web.graphs`. In `auto` mode Kula tries to
detect a hardware limit (CPU TjMax, disk thermal max, NIC link speed).

### Per-device selectors & split view

Network, Disk I/O, Disk space, Disk temperature, and GPU charts can show all devices on one
chart or be **split** into one chart per device/interface. Toggle this per-chart with the
split (⊟) button, or set defaults under `web.graphs.split`.

### Layout toggle

Switch between a **grid** layout and a **stacked list** layout.

### Aggregation selector

For historical data, choose whether each downsampled point shows the **average**, **minimum**,
or **maximum** of its window. The default comes from `web.default_aggregation`.

### Gap handling

By default, gaps between measurements (e.g. after a restart) are drawn as empty space. Set
`web.join_metrics: true` to connect across gaps instead.

### Alerts

The dashboard raises in-UI alerts for:

- **Clock not synchronized** — system time isn't synced to an NTP source.
- **Low entropy** — the kernel entropy pool is depleted.
- **System overload** — sustained high load.

### Theme & language

- Light / dark / auto theme (set the default with `global.default_theme`).
- 11 UI languages (`ar de en es fr hi ja ko pl pt zh`); selector can be hidden/forced via
  `web.lang`.

### AI assistant

When Ollama is enabled, a 🤖 button opens a local AI analysis panel. See
[AI Assistant](10-ai-assistant.md).

## REST API & WebSocket

The dashboard is driven by a small JSON API. If you want to build your own client or scripts,
the endpoints are documented for developers in [Web Server & API](../dev/07-web-server-api.md).
The headline endpoints:

| Endpoint | Purpose |
|----------|---------|
| `GET /api/current` | The latest sample |
| `GET /api/history?from=&to=&points=` | Time-range history (downsampled) |
| `GET /api/config` | UI configuration (theme, langs, graph bounds, custom metrics) |
| `GET /ws` | WebSocket live stream |
| `GET /metrics` | Prometheus exposition (if enabled) |
| `GET /health`, `GET /status` | Liveness |

Next: [Terminal UI](06-tui.md).
