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

### PostgreSQL Array Arguments

The DB and transaction executors normalize non-byte slice and array arguments
with `pq.Array` before calling `database/sql`. This supports PostgreSQL array
parameters such as `[]int64`, `[]uint32`, and `[]float64` without requiring the
caller to wrap every value manually.

```go
_, err := executor.ExecContext(
    ctx,
    "UPDATE users SET role_ids = $1 WHERE id = $2",
    []int64{10, 20},
    userID,
)
```

Existing `driver.Valuer` arguments are preserved, `[]byte` remains a binary
argument, and a nil pointer is passed as `NULL`. This normalization is only
applied to PostgreSQL; MySQL and Oracle args retain their existing behavior.

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

## Logged Values

The default logger renders PostgreSQL slices and arrays as `ARRAY[...]`. MySQL
and Oracle retain parenthesized collection rendering. Long scalar and collection
values are limited to 120 bytes with `...(truncated)...` in the middle while
preserving their beginning and ending. This affects log output only; executed
SQL and parameter values are unchanged.
