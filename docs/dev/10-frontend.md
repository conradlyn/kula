# Frontend (SPA & TUI)

Kula has two user interfaces: the embedded **web SPA** and the **terminal UI**. Both are part of
the single binary.

---

## Web dashboard (SPA)

Source: [`internal/web/static/`](../../internal/web/static/), embedded via `//go:embed`.

It is a dependency-light, framework-free single-page app built on **Chart.js** (bundled,
vendored under `js/chartjs/`) with custom SVG/bar gauges. It connects over WebSocket for live
data and falls back to the history REST API for longer ranges.

### Asset layout

```
static/
├── index.html              # main dashboard template (CSP nonce + SRI injected by server)
├── style.css               # dashboard styles
├── kula.svg / favicon.ico  # branding
├── game.html / game.css / game.js   # Space Invaders easter egg
├── fonts/
│   ├── Inter/              # UI font (OFL-1.1)
│   └── Press_Start_2P/     # game font (OFL-1.1)
└── js/
    ├── chartjs/            # vendored Chart.js + zoom + date-fns adapter (min.js)
    └── app/               # Kula's own ES6 modules
```

### ES6 modules (`js/app/`)

Modules are plain ES6 (no bundler). Load order matters: `state.js` first, `main.js` last.

| Module | Responsibility |
|--------|----------------|
| `state.js` | Shared app state, color palette, global Chart.js config (**load first**) |
| `url-state.js` | Restore/sync time window, custom range, and aggregation to the URL query string (shareable views) |
| `main.js` | Entry point; wires event listeners, starts auth + WebSocket (**load last**) |
| `api.js` | URL helpers that prepend `window.KULA_BASE_PATH` (base-path support) |
| `auth.js` | Auth status check, config fetch, login/logout |
| `websocket.js` | WebSocket connect, reconnect, live-queue drain |
| `charts-init.js` | Chart.js instance creation; full dashboard init; app-chart teardown |
| `charts-data.js` | Sample ingestion, chart updates, zoom sync, gap insertion, device selectors |
| `gauges.js` | Bar gauges, sparkline backgrounds, live gauge updates |
| `controls.js` | Pause/resume, layout toggle, time-range selection, history fetch |
| `focus-mode.js` | Select/persist a subset of chart cards |
| `split.js` | Per-device/interface graph splitting |
| `header.js` | Header bar + chart subtitle updates |
| `theme.js` | Dark/light theme apply + toggle |
| `alerts.js` | Alert evaluation (clock sync, low entropy, overload) + dropdown |
| `i18n.js` | Fetches translations from `/api/i18n` and applies to the DOM |
| `ollama.js` | AI assistant panel; SSE streaming from `/api/ollama/chat` |
| `ui-actions.js` | Hover-pause on cards, expand/collapse, per-chart Y-axis settings |
| `utils.js` | Formatting helpers |

### Data flow

```
WebSocket /ws ──► websocket.js ──► charts-data.js ──► Chart.js + gauges.js
                                       ▲
Time-range select ── controls.js ──► /api/history ──┘ (downsampled history)
/api/config ── auth.js ──► state.js (theme, langs, graph bounds, custom metrics, ollama)
```

### Base-path awareness

The server injects `window.KULA_BASE_PATH` into the HTML template; `api.js` prepends it to every
request so the SPA works unchanged when mounted under a reverse-proxy prefix.

### Adding a chart for a new metric type

When you add an application metric type, the frontend side is: define an `APP_ORDER_*` constant
in `charts-data.js`, create the chart dynamically on first data (`if (s.apps?.foo) { ... }`), and
register the card ID in `charts-init.js`'s `destroyAppCharts()` for cleanup. See
[Adding a Metric Type](14-adding-metrics.md#11-frontend-charts).

### Easter egg

A Space Invaders clone (`game.html`/`game.js`, Press Start 2P font) is reachable from a header
button when `global.easter_egg` is true.

---

## Terminal UI (TUI)

Package: [`internal/tui`](../../internal/tui/), built with **Bubble Tea** + **Lipgloss**.

| File | Role |
|------|------|
| [`tui.go`](../../internal/tui/tui.go) | Bubble Tea model: rolling metric rings, tab navigation, refresh loop |
| [`view.go`](../../internal/tui/view.go) | The 7 tab views (Overview, CPU, Memory, Network, Disk, Processes, GPU) with progress bars and responsive layout |
| [`styles.go`](../../internal/tui/styles.go) | Dark purple/slate theme with style caching for performance |

`tui.RunHeadless(collector, refreshRate, osName, kernel, arch, version, showSystemInfo)` drives
it. The TUI runs its own collector and does **not** read the storage tiers — it samples live.
See the user-facing [Terminal UI](../user/06-tui.md) page.

Next: [Internationalization](11-i18n.md).
