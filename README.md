# hellnet-lib-api

Modular, embeddable, framework-neutral API contracts for Go services, with
environment-driven configuration built on
[hellnet-lib-environments](https://github.com/guilhermelinosp/hellnet-lib-environments).

## Guarantees

- No Gin, Kafka, database, telemetry, or environment dependency in the core.
- Small interfaces that adapters can implement for Gin, Chi, Echo, or net/http.
- Strict JSON binding and sanitized errors.
- Applications own middleware order, lifecycle, routes, and configuration.
- All runtime settings are read from environment variables with a single
  source of truth (`config`), fully documented below.

## Packages

| Package       | Purpose                                                                 |
|---------------|-------------------------------------------------------------------------|
| `api`         | Core contracts: `Handler`, `Request`, `Response`, `Route`, `Middleware`, `Router`, platform routes (`/live`, `/ready`, `/health`, `/api/v1`) and JSON helpers. Framework-neutral. |
| `errors`      | HTTP-safe typed errors: `Error`, `New`, `Wrap`, `Map`, `Validation`, `Internal`. |
| `config`      | Environment-driven configuration built on hellnet-lib-environments: `Config`, `Build`, `FromEnv`, `Validate`, `IsProduction`. |
| `adapter`     | HTTP adapter helpers + Gin-backed `api.Router`: `Config` (embeds `config.Config`), `ConfigFromEnv`, `FromConfig`, `JSONHandler[TReq, TResp]`, `New(cfg)`, typed `Request`, `WriteResponse`/`WriteError`, security/CORS/requestID middlewares, `TranslatePath`. |

Framework adapters should be separate packages or modules and may depend on
this core. The core remains stable and easy to embed.

## Environment variables

All variables use the `HELLNET_` prefix, with the legacy `APP_` prefix
honoured as a fallback. `FromEnv` also loads a local `.env` file in
development environments (no-op in production) via
`environments.LoadDotEnv`.

| Variable                         | Type     | Default    | Description                          |
|----------------------------------|----------|------------|--------------------------------------|
| `HELLNET_ENVIRONMENT`            | string   | `""` (dev) | Deployment environment; anything not `development/dev/local/test/testing` is treated as production. |
| `HELLNET_SERVICE`                | string   | **required** | Service name shown in `/`. `FromEnv` fails if empty. |
| `HELLNET_ENV`                    | string   | `Development` | Logical environment label.           |
| `HELLNET_PORT`                   | string   | `8080`     | Listen port (1–65535).               |
| `HELLNET_BODY_LIMIT`             | int      | `1048576` (1 MiB) | Max request body size in bytes. |
| `HELLNET_CORS_ALLOWED_ORIGINS`   | list     | (none)     | Comma-separated allowed CORS origins (`*` allows all). |
| `HELLNET_SHUTDOWN_TIMEOUT`       | duration | `10s`      | Graceful shutdown window.            |
| `HELLNET_READ_TIMEOUT`           | duration | `15s`      | Server read timeout.                 |
| `HELLNET_WRITE_TIMEOUT`          | duration | `30s`      | Server write timeout.                |
| `HELLNET_IDLE_TIMEOUT`           | duration | `120s`     | Server idle timeout.                 |
| `HELLNET_READ_HEADER_TIMEOUT`    | duration | `10s`      | Server read-header timeout.          |
| `HELLNET_RELEASE_MODE`           | bool     | `false`    | Force release mode (gin) regardless of `HELLNET_ENV`. |
| `HELLNET_LOG_LEVEL`              | string   | `info`     | `debug`, `info`, `warn` or `error`.  |
| `HELLNET_LOG_FORMAT`             | string   | `text`     | `json` or `text`.                    |
| `HELLNET_TRUSTED_PROXIES`        | list     | (none)     | Comma-separated trusted proxy CIDRs/addresses. |

Durations accept Go syntax (`15s`, `2m`) and .NET `HH:MM:SS` via
`environments.ParseDuration`.

## Quickstart (Gin)

```go
cfg, err := config.FromEnv(config.Build{
    Version: version, Commit: commit, Date: date,
})
if err != nil {
    log.Fatalf("config: %v", err)
}

logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: cfg.LogLevel}))
adapterCfg, err := adapter.ConfigFromEnv(logger)
if err != nil {
    log.Fatalf("config: %v", err)
}

router := adapter.New(*adapterCfg)
api.RegisterPlatform(router, api.ServiceInfo{
    Name: cfg.Name, Version: cfg.Build.Version,
    Commit: cfg.Build.Commit, BuiltAt: cfg.Build.Date,
}, api.Deps{
    Platform: api.PlatformHandlers{
        Live:   http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
        Ready:  http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) }),
    },
    Routes: []api.Route{{
        Method: http.MethodGet, Path: "/hello/{name}",
        Handler: adapter.JSONHandler[struct{ Name string `json:"name"` }, map[string]string]{
            HandleFunc: func(ctx context.Context, req *struct{ Name string `json:"name"` }) (map[string]string, error) {
                return map[string]string{"hello": req.Name}, nil
            },
        },
    }},
})

http.ListenAndServe(":"+cfg.Port, router)
```

## Development

Production Go-library structure following
[hellnet-lib-template](https://github.com/guilhermelinosp/hellnet-lib-template).

### Checks (CI parity)

```sh
gofmt -l .           # must be empty
go build ./...
go vet ./...
golangci-lint run --config .golangci.yml ./...   # exit 0
```