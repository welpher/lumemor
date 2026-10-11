# openapi

OpenAPI spec-first: `openapi.yaml` is the single source of truth for all three clients.

- oapi-codegen → Go server types + chi stubs (server)
- openapi-typescript → TS client (web)
- openapi-generator (dart-dio) → Dart client (mobile)

CI regenerates all clients and fails on drift. Never hand-edit generated files.
