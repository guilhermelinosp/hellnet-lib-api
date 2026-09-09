# hellnet-lib-api-template

Modular, embeddable API contracts scaffolded from `hellnet-lib-template`.

## Design

The core has no framework or infrastructure dependencies: no Gin, Kafka,
database, or telemetry. It uses only the standard library. Applications choose
their own HTTP adapter and lifecycle.

Packages:

- `api`: Handler, Request, Response, Route, Middleware, Router and JSON helpers.
- `errors`: sanitized HTTP errors and mapping.

Adapters such as Gin, Chi, security middleware, request IDs and CORS should be
published separately. The core does not require environment variables.

Source template: https://github.com/guilhermelinosp/hellnet-lib-template
