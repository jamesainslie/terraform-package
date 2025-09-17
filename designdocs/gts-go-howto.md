## GTS Go HOWTO: Logging, Metrics, and Tracing

This guide shows how to implement GTS in Go applications for:
- Structured logging (both GTS event logs and plain logs)
- OpenTelemetry metrics
- OpenTelemetry tracing
- HTTP/gRPC middleware instrumentation

It focuses on the Go client in this repo: `github.com/geico-private/gts/clients/go`.

---

### Prerequisites
- Go 1.25.1+ recommended (uses `log/slog`)
- Access to an OTLP (OpenTelemetry) collector at `host:4317` (e.g., Local Titan Sandbox or otel-tui)

Add the dependency:

```
go get github.com/geico-private/gts/clients/go@v0.6.28
```

---

## 1) Minimal, end-to-end setup

This example initializes logging, tracing, and metrics, shows both GTS event logs and plain logs, and demonstrates a simple metric and span.

```go
package main

import (
    "context"
    "errors"
    "net/http"
    "time"

    "log/slog"

    "github.com/geico-private/gts/clients/go/gtcore"
    "github.com/geico-private/gts/clients/go/tlm/ignorelist"
    "github.com/geico-private/gts/clients/go/tlm/lg"
    "github.com/geico-private/gts/clients/go/tlm/lg/gtslog"
    "github.com/geico-private/gts/clients/go/tlm/lg/lga"
    "github.com/geico-private/gts/clients/go/tlm/mtr"
    "github.com/geico-private/gts/clients/go/tlm/trc"
    "github.com/geico-private/gts/clients/go/middleware/httpmw"

    "go.opentelemetry.io/otel"
    "go.opentelemetry.io/otel/attribute"
)

func main() {
    ctx := context.Background()

    // 1) Root logger (writes to stdout; can also export via OTLP if configured)
    ctx, root := gtslog.New(ctx)
    root = root.With(
        slog.String(lga.ServiceName, "orders-api"),
        slog.String(lga.ServiceVersion, "1.2.3"),
        slog.String(lga.ServiceInstanceID, "instance-abc123"),
    )
    ctx = lg.NewContext(ctx, root)
    defer gtslog.DefaultGtsLogDefer(ctx)

    // 2) Tracing (requires env vars below)
    var err error
    ctx, err = trc.Start(ctx)
    if err != nil {
        root.Error("could not set up tracing", lga.Error, err)
    } else {
        defer trc.DefaultTraceDefer(ctx)
    }

    // 3) Metrics (requires env vars below)
    ctx, err = mtr.Start(ctx)
    if err != nil {
        root.Error("could not set up metrics", lga.Error, err)
    } else {
        defer mtr.DefaultMetricsDefer(ctx)
    }

    // 4) Optional: ignore endpoints from telemetry (health, metrics, etc.)
    ctx = ignorelist.NewContext(ctx, []string{"/healthz", "/metrics"})

    // 5) GTS core event logs + plain logs
    gtcore.SysStartup(ctx, 0, slog.String("bind_address", ":8080"))
    root.Info("service is starting", slog.String("env", "dev"))

    // 6) Manual span + metric
    ctx, log, span := trc.StartSpan(ctx, "startup")
    span.SetAttributes(attribute.String("phase", "init"))
    counter, _ := otel.Meter("orders").Int64Counter("orders_startups_total")
    counter.Add(ctx, 1)
    log.Info("initialized counters and spans")
    span.End()

    // 7) HTTP server with middleware
    mux := http.NewServeMux()
    mux.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
        ctx := r.Context()
        log := lg.FromContext(ctx)

        gtcore.AppRequestInitiated(ctx)
        defer gtcore.AppRequestCompleted(ctx)

        log.Info("handling hello", slog.String("path", r.URL.Path))
        w.WriteHeader(http.StatusOK)
        _, _ = w.Write([]byte("hello"))
    })

    srv := &http.Server{Addr: ":8080", Handler: mux}
    if err := httpmw.ConfigureServer(ctx, srv); err != nil {
        root.Error("failed to configure server", lga.Error, err)
    }
    go func() {
        if err := srv.ListenAndServe(); !errors.Is(err, http.ErrServerClosed) {
            root.Error("server error", lga.Error, err)
        }
    }()

    time.Sleep(5 * time.Second)
    _ = srv.Close()
}
```

---

## 2) Logging

### 2.1 GTS core event logs
Use `gtcore` for standardized, compliant events (pre-defined types and attributes). Common examples:

```go
// Service lifecycle
gtcore.SysStartup(ctx, 0, slog.String("note", "booting"))

// Request lifecycle
gtcore.AppRequestInitiated(ctx)
// ... handler work ...
gtcore.AppRequestCompleted(ctx, slog.Int("status_code", 200))

// Errors
if err != nil {
    gtcore.AppError(ctx, err)
}
```

Notes:
- Functions are generated from the standard event schema and enforce structure/consistency.
- You can attach extra attributes with `slog.*` (e.g., `slog.String`, `slog.Int`, `slog.Any`).

### 2.2 Plain logs (P.O.L.E. — Plain Old Logger Events)
You can also emit ordinary logs via the request-scoped GTS logger (a `*slog.Logger`).

```go
log := lg.FromContext(ctx)
log.Debug("debug details", slog.String("key", "value"))
log.Info("something happened", slog.Int("count", 3))
log.Warn("be careful", slog.String("why", "rate limit"))
log.Error("oops", lga.Error, err)
```

Notes:
- The logger writes to stdout and, if configured, also exports to OTLP over gRPC.
- The logger is automatically decorated by middleware with trace/request attributes.

---

## 3) Tracing

Initialize tracing once at startup:

```go
ctx, err := trc.Start(ctx)
if err != nil {
    // handle
}
defer trc.DefaultTraceDefer(ctx)
```

Create manual spans when helpful (middleware already creates spans around requests):

```go
ctx, log, span := trc.StartSpan(ctx, "processOrder")
span.SetAttributes(attribute.String("order.id", orderID))
log.Info("started order processing")
// ... work ...
span.End()
```

Important:
- Tracing only exports data if enabled/configured via environment variables (see Section 6).
- The span-aware logger `log` is decorated with span/trace IDs for correlation.

---

## 4) Metrics

Start metrics once at startup:

```go
ctx, err = mtr.Start(ctx)
if err != nil {
    // handle
}
defer mtr.DefaultMetricsDefer(ctx)
```

Emit application metrics via OpenTelemetry:

```go
meter := otel.Meter("orders")
requests, _ := meter.Int64Counter("orders_requests_total")
requests.Add(ctx, 1)
```

The HTTP middleware automatically publishes request duration histograms and related metrics.

---

## 5) HTTP and gRPC Middleware

### 5.1 net/http (stdlib)

```go
srv := &http.Server{Addr: ":8080", Handler: mux}
if err := httpmw.ConfigureServer(ctx, srv); err != nil {
    // handle
}
```

Effects:
- Ensures the server runs with the root context you initialized
- Adds/propagates trace IDs; creates spans if tracing is enabled
- Decorates the logger with request-scoped attributes
- Emits duration metrics per handler

### 5.2 Echo

```go
e := echo.New()
// Option A: full telemetry configuration helper
if err := echomw.ConfigureServer(ctx, e); err != nil {
    // handle
}
// Option B: logger injection only
// e.Use(echomw.NewMiddleware(rootLogger))
```

### 5.3 Gin

```go
r := gin.New()
ginmw.ConfigureServer(ctx, r)
```

### 5.4 gRPC
See `clients/go/middleware/grpcmw` for interceptors to get similar logging/metrics/tracing behavior for gRPC servers/clients.

---

## 6) Configuration (Environment Variables)

The Go client reads configuration from environment variables with the prefix `GEICO_GTS_`.

Tracing:
- `GEICO_GTS_TR_ENABLED`: `true`/`false` (default: `false`)
- `GEICO_GTS_TR_GRPC_URL`: OTLP endpoint for traces (default: `127.0.0.1:4317`)
- `GEICO_GTS_TR_TIMEOUT`: seconds (default: `10.0`)

Metrics:
- `GEICO_GTS_MT_GRPC_URL`: OTLP endpoint for metrics (default: `127.0.0.1:4317`)
- `GEICO_GTS_MT_TIMEOUT`: seconds (default: `10.0`)
- `GEICO_GTS_MT_INTERVAL`: export interval seconds (default: `15.0`)
- `GEICO_GTS_MT_INHIBIT_PATH`: `true` to omit path labels (default: `false`)

Logging:
- `GEICO_GTS_LOG_GRPC_URL`: OTLP endpoint for logs; if unset, logs go to stdout only

Notes:
- Defaults are sensible for local development (4317 is the default OTLP gRPC port).
- All variables are optional except `GEICO_GTS_TR_ENABLED` which controls tracing on/off.

---

## 7) Verifying locally

Option A: otel-tui (text UI for OTLP data)
- Ensure nothing else is listening on `:4317`.
- Install and run: `club exec otel-tui` (see repo README for details)
- Start your app; watch logs/metrics/traces appear in the TUI

Option B: Local Titan Sandbox (Grafana, Loki, Mimir, Tempo)
- Follow the instructions in the repo README (Local Titan Sandbox section)
- Redirect your app stdout to the file Titan monitors (see README)
- Open Grafana at http://localhost:3000 and explore logs and metrics

---

## 8) Best practices
- Always add service resource attributes on the root logger:
  - `service.name`, `service.version`, `service.instance.id` via `lga` keys
- Use `gtcore` for business/system events; use plain logs for ad hoc details
- Prefer middleware instrumentation; add manual spans only where needed
- Use `ignorelist` to exclude noisy endpoints (health/metrics)
- Always `defer` the `Default*Defer` functions to flush/export on shutdown
- Never let telemetry failures affect business logic; handle errors but continue

---

## 9) Troubleshooting
- “No traces appearing”
  - Ensure `GEICO_GTS_TR_ENABLED=true`
  - Verify `GEICO_GTS_TR_GRPC_URL` is reachable and no firewall issues
- “Metrics not visible”
  - Check `GEICO_GTS_MT_GRPC_URL` and `GEICO_GTS_MT_INTERVAL`
  - Confirm your code calls `mtr.Start(ctx)`
- “Logs only in stdout”
  - Set `GEICO_GTS_LOG_GRPC_URL` to export logs via OTLP
- “Port 4317 in use”
  - Stop other collectors (e.g., Local Titan Sandbox) when using otel-tui

---

## 10) Useful imports (reference)

```go
import (
  "log/slog"
  "github.com/geico-private/gts/clients/go/gtcore"
  "github.com/geico-private/gts/clients/go/middleware/httpmw"
  "github.com/geico-private/gts/clients/go/middleware/echomw"
  "github.com/geico-private/gts/clients/go/middleware/ginmw"
  "github.com/geico-private/gts/clients/go/tlm/ignorelist"
  "github.com/geico-private/gts/clients/go/tlm/lg"
  "github.com/geico-private/gts/clients/go/tlm/lg/gtslog"
  "github.com/geico-private/gts/clients/go/tlm/lg/lga"
  "github.com/geico-private/gts/clients/go/tlm/mtr"
  "github.com/geico-private/gts/clients/go/tlm/trc"
  "go.opentelemetry.io/otel"
)
```

For more examples, see `clients/go/examples` and the Go README in `clients/go/README.md`.
