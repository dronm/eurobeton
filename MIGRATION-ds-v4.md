# Migration to ds/v4

The application now uses `github.com/dronm/ds/v4` throughout its database services. Database acquisition uses each operation's context, including the WebSocket method timeout and HTTP request timeout.

## Dependencies

| Module | Verified version/source |
| --- | --- |
| `ds/v4` | Supplied v4 source; commit `1d1425e7a834`, version `v4.0.0-20260612024620-1d1425e7a834` |
| `gobizapp` | Supplied `gobizapp(1).zip`, which already uses `ds/v4` |
| `crudifier/v2` | `v2.0.0-20260616013530-315cbf1baad3` |
| `pgx/v5` | `v5.8.0`, as required by the supplied libraries |

The application's existing local `replace` directives are retained. The ds replacement now targets `/home/andrey/go/ds/v4`. Ensure the local `gobizapp` folder contains the supplied v4-compatible code. Its filters require the five-argument `ModelToDBFilters` API from the newer `crudifier/v2`; update that local folder or remove its `replace` directive to use the pinned published version.

The retained `gobizapp` version in `require` is overridden by its local replacement. Removing that replacement would select the old library, which is incompatible with this migration. Keep the replacement until a matching v4-compatible release is selected.

## Application changes

- Service fields, constructors and RPC initializers use `ds.Provider`.
- Database helpers use `ds.Conn`, and row lookup checks use `ds.ErrNoRows`.
- Acquisitions receive `ctx`; request cancellation and method timeouts can interrupt pool waits.
- Startup uses the new `database.Initialize(ctx, storage, callback)` signature and a bounded startup context.
- PostgreSQL sessions and LISTEN/NOTIFY retain their native pgx pool, obtained through v4's `PrimaryPool()` API.
- Production Upload, Ping and Error operations reuse a single database lease for updates, mappings, logging and event notifications.
- Material replacement uses an explicit transaction, retaining the previous batch's atomic delete/insert behavior.
- Cache initialization acquires a connection before taking its map lock.
- Failover shutdown cancels replay and waits for the worker. The notification listener releases its lease on cancellation before the provider closes.
- Password email generation queries through `ds.Conn` because the existing notifier's SQL helper still accepts a native pgx connection.

No database schema or configuration format changes are required.

## Validation

The application built successfully and passed `go test -race ./...` and `go vet ./...`, using temporary module replacements for the supplied libraries and the published versions listed above. Regression tests cover context cancellation, missing-user lookup, one-lease production operations with cold mapping caches, material commit/rollback, and failover cancellation. No live production database was used.

The supplied ds core package tests pass. Its pgds test files reference obsolete internal types; those library test files were left unchanged.

After applying the source and aligning the local dependencies, run:

```bash
go mod tidy
go test -race ./...
go build ./...
```

WebSocket write deadlines and the subscriber/event-lock blocking paths remain follow-up work in the hang investigation.

## Commit message

```text
refactor(database): migrate eurobeton to ds/v4

Propagate operation contexts to database acquisition and adapt services,
startup and PostgreSQL pool access to the v4 provider interfaces.
Reuse production upload leases, preserve atomic material replacement,
and cancel background replay before closing the provider.
```
