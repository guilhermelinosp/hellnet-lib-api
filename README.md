# hellnet-lib-api

Modular, embeddable, framework-neutral API contracts for Go services.

## Guarantees

- No Gin, Kafka, database, telemetry, or environment dependency in the core.
- Small interfaces that adapters can implement for Gin, Chi, Echo, or net/http.
- Strict JSON binding and sanitized errors.
- Applications own middleware order, lifecycle, routes, and configuration.

## Packages

- `api`: Handler, Request, Response, Route, Middleware, Router and JSON helpers.
- `errors`: HTTP-safe error constructors and internal-cause mapping.

Framework adapters should be separate packages or modules and may depend on this
core. The core remains stable and easy to embed.

The repository follows the production Go-library structure from
https://github.com/guilhermelinosp/hellnet-lib-template.
