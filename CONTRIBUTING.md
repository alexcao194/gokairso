# Contributing Guide

This document summarizes the conventions to follow when contributing to `gpgo`.

## Environment Setup

- Install the Go version required by `go.mod`.
- Install Docker and Docker Compose, because the dev environment runs in containers.
- Create a suitable `.env` file for your local environment if you need to run the app, migrations, or database.
- Start the dev environment with:

```sh
make dev
```

- Stop the dev environment with:

```sh
make dev-down
```

## Workflow

1. Create a dedicated branch for your change.
2. Inspect the existing code before editing, especially the related feature under `internal/features`.
3. Keep changes scoped to the requested behavior.

## Feature Structure

Each feature under `internal/features/<feature>` must keep that feature's data types and mapping logic in its own `dto.go` file.

Keep the following in `dto.go`:

- request structs
- response structs
- service DTOs
- repository DTOs
- mapping structs
- mapping functions
- mapping logic

Do not move feature-specific mapping logic into another file.

Example feature structure:

```text
internal/features/user/
  dto.go
  handler.go
  repository.go
  service.go
```

## Database and Code Generation

If you change the database schema:

- Add a new migration under `db/migrations`.
- Update queries under `db/queries` if needed.
- Run SQLC when query or schema changes require regenerated code:

```sh
make sqlc
```

To create a new migration:

```sh
make migrate-create name=<migration_name>
```

## Swagger

If you add or change an HTTP handler, write or update the Swagger annotations for that handler before pushing.

If you change the API contract or Swagger annotations, regenerate Swagger docs with:

```sh
make swagger
```

## Code Style

- Follow the repository's existing patterns before adding new abstractions.
- Keep request, response, service, repository, and mapping names clear and feature-oriented.
- Prefer structured APIs or parsers over manual string handling when a good option exists.
- Avoid changing files outside the requested scope unless required.
- Do not revert someone else's changes unless explicitly requested.

## Merge Request Checklist

- [ ] The change is scoped to the requested behavior.
- [ ] The feature's DTOs and mapping logic are in `dto.go`.
- [ ] Swagger annotations were added or updated for changed HTTP handlers.
- [ ] `make swagger` was run if handlers, API contracts, or Swagger annotations changed.
- [ ] No local files, caches, or secrets are committed.
