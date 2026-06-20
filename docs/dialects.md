# Dialect Notes

The supported dialect constructors are:

```go
dialect.NewPostgres()
dialect.NewMysql()
dialect.NewOracle()
```

## PostgreSQL

- Dialect name: `postgres`
- Placeholders: `$1`, `$2`, ...
- Pagination: `LIMIT n OFFSET m`
- Identifier quoting returns the identifier unchanged in the current dialect.
- `SupportReturning()` is `true`.
- Advisory lock: `pg_try_advisory_xact_lock($1)`.
- Bulk insert uses query mode to scan returned primary keys.

## MySQL

- Dialect name: `mysql`
- Placeholders: `?`
- Pagination: `LIMIT n OFFSET m`
- Offset without limit uses `LIMIT 18446744073709551615 OFFSET n`.
- Identifier quoting uses backticks.
- `SupportReturning()` is `false`.
- Advisory lock: `GET_LOCK(?, 0)` and `RELEASE_LOCK(?)`.
- Advanced insert returning uses MySQL-specific follow-up scan behavior in the
  implementation.

## Oracle

- Dialect name: `oracle`
- Placeholders: `:1`, `:2`, ...
- Pagination: `OFFSET n ROWS FETCH NEXT m ROWS ONLY`.
- Identifier quoting uses double quotes.
- `SupportReturning()` is `true`.
- Advisory lock uses `DBMS_LOCK`.
- Count wrappers use subquery alias syntax without `AS`.

## Placeholder Mode

`config.PlaceholderAuto` uses named-style placeholders for Oracle and numbered
placeholders for the other supported dialects. The query builder rebinding keeps
raw `?` condition placeholders aligned with each dialect.
